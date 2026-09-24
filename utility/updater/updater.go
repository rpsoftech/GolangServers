// Package updater is the single OTA update client for every published
// component. Releases are published by utility/deploy under the KV key
// <ENV>_<component>_<os>_<arch> and verified here before anything is
// downloaded or replaced.
package updater

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

const (
	DefaultKVBaseURL      = "https://keyvalue.rpso.in/public/"
	DefaultCheckInterval  = 5 * time.Minute
	versionFileSuffix     = ".version.json"
	metadataTimeout       = 10 * time.Second
	binaryDownloadTimeout = 10 * time.Minute
)

// BuildEnv is the release channel (PRODUCTION, STAGING, ...) a binary was
// published to. utility/deploy sets it with
// -ldflags "-X github.com/rpsoftech/golang-servers/utility/updater.BuildEnv=<ENV>".
// It is empty for local and dev builds, which never self-update.
var BuildEnv = ""

type KVResponse struct {
	Version   int    `json:"version"`
	URL       string `json:"url"`
	SHA256    string `json:"sha256"`
	Signature string `json:"signature"` // base64 ed25519 over SignedMessage
}

// Config describes which release channel to follow.
type Config struct {
	Component string
	// Env defaults to BuildEnv.
	Env string
	// CurrentVersion is the running version. UpdateFile ignores it and reads
	// the version file next to the target instead.
	CurrentVersion int
	// KVBaseURL defaults to DefaultKVBaseURL.
	KVBaseURL string
	// OnProgress, if set, is called while the binary downloads. total is -1
	// when the server does not send a length.
	OnProgress func(read, total int64)
	// BeforeReplace, if set, runs right before UpdateFile swaps the target,
	// e.g. to stop the process that is running it.
	BeforeReplace func()
}

// Published reports whether this binary came from the release pipeline.
func Published() bool { return BuildEnv != "" }

// ParseVersion turns the -X injected main.version string into a number.
// Anything that is not a positive integer (e.g. "dev") is version 0.
func ParseVersion(v string) int {
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// GetFileKey is the KV key for one component build.
func GetFileKey(envName, component, osName, arch string) string {
	return fmt.Sprintf("%s_%s_%s_%s", envName, component, osName, arch)
}

func (c Config) withDefaults() (Config, error) {
	if c.Env == "" {
		c.Env = BuildEnv
	}
	if c.KVBaseURL == "" {
		c.KVBaseURL = DefaultKVBaseURL
	}
	if c.Env == "" || c.Component == "" {
		return c, errors.New("updater: Env and Component are required (unpublished build?)")
	}
	return c, nil
}

// Key is the KV key this config reads for the running OS and architecture.
func (c Config) Key() string {
	c, _ = c.withDefaults()
	return GetFileKey(c.Env, c.Component, runtime.GOOS, runtime.GOARCH)
}

// Latest fetches the published release and verifies its signature. newer is
// false when the release is not above currentVersion; the release is still
// returned for logging.
func Latest(ctx context.Context, cfg Config, currentVersion int) (rel KVResponse, newer bool, err error) {
	cfg, err = cfg.withDefaults()
	if err != nil {
		return rel, false, err
	}
	kvKey := cfg.Key()

	ctx, cancel := context.WithTimeout(ctx, metadataTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.KVBaseURL+kvKey, nil)
	if err != nil {
		return rel, false, err
	}
	setCommonHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return rel, false, fmt.Errorf("KV server unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return rel, false, fmt.Errorf("KV server returned %d for %s", resp.StatusCode, kvKey)
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return rel, false, fmt.Errorf("decode %s: %w", kvKey, err)
	}
	if rel.Version <= currentVersion {
		return rel, false, nil
	}
	// Verify before downloading: the signature covers the SHA256, and the
	// SHA256 is checked against the downloaded file.
	if err := VerifyRelease(kvKey, rel); err != nil {
		return rel, false, fmt.Errorf("rejecting %s v%d: %w", kvKey, rel.Version, err)
	}
	return rel, true, nil
}

// SelfUpdate replaces the running executable when a newer signed release
// exists. The caller restarts the process when it returns true.
func SelfUpdate(ctx context.Context, cfg Config) (bool, error) {
	exePath, err := os.Executable()
	if err != nil {
		return false, err
	}
	// Windows cannot delete a running exe, so the previous one is renamed to
	// .old during the swap and cleaned up on the next run.
	os.Remove(exePath + ".old")

	rel, newer, err := Latest(ctx, cfg, cfg.CurrentVersion)
	if err != nil || !newer {
		return false, err
	}
	log.Printf("[updater] %s: v%d -> v%d", cfg.Key(), cfg.CurrentVersion, rel.Version)
	if err := installRelease(ctx, cfg, rel, exePath); err != nil {
		return false, err
	}
	writeVersionFile(exePath, rel)
	return true, nil
}

// UpdateFile keeps another binary at target on the newest signed release,
// e.g. a launcher managing the app it starts. The installed version is kept
// in target + ".version.json". A missing target is always downloaded.
func UpdateFile(ctx context.Context, cfg Config, target string) (bool, error) {
	current := 0
	if _, err := os.Stat(target); err == nil {
		current = readVersionFile(target)
	}
	rel, newer, err := Latest(ctx, cfg, current)
	if err != nil || !newer {
		return false, err
	}
	log.Printf("[updater] %s: v%d -> v%d at %s", cfg.Key(), current, rel.Version, target)
	if err := installRelease(ctx, cfg, rel, target); err != nil {
		return false, err
	}
	writeVersionFile(target, rel)
	return true, nil
}

// RunDaemon checks now and then every DefaultCheckInterval until ctx ends.
// After a successful update it calls onUpdated (typically the cancel func of
// the main context, so the server shuts down and its supervisor restarts it)
// and returns. Unpublished builds return immediately.
func RunDaemon(ctx context.Context, cfg Config, onUpdated func()) {
	if !Published() && cfg.Env == "" {
		log.Println("[updater] unpublished build: OTA updates disabled")
		return
	}
	check := func() bool {
		updated, err := SelfUpdate(ctx, cfg)
		if err != nil {
			log.Printf("[updater] %v", err)
			return false
		}
		if updated {
			log.Println("[updater] update installed, restarting")
			onUpdated()
		}
		return updated
	}
	if check() {
		return
	}
	ticker := time.NewTicker(DefaultCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if check() {
				return
			}
		}
	}
}

// installRelease downloads rel, checks its hash, unpacks it next to target
// and swaps it into place.
func installRelease(ctx context.Context, cfg Config, rel KVResponse, target string) error {
	gzTmp := target + ".gz.tmp"
	binTmp := target + ".tmp"
	defer os.Remove(gzTmp)
	defer os.Remove(binTmp)

	if err := download(ctx, rel, gzTmp, cfg.OnProgress); err != nil {
		return err
	}
	if err := extractGzip(gzTmp, binTmp); err != nil {
		return fmt.Errorf("unpack: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binTmp, 0755); err != nil {
			return err
		}
	}
	if cfg.BeforeReplace != nil {
		cfg.BeforeReplace()
	}
	if _, err := os.Stat(target); err == nil {
		os.Remove(target + ".old")
		if err := os.Rename(target, target+".old"); err != nil {
			return fmt.Errorf("move old binary aside: %w", err)
		}
	}
	if err := os.Rename(binTmp, target); err != nil {
		os.Rename(target+".old", target) // put the old binary back
		return fmt.Errorf("install new binary: %w", err)
	}
	if runtime.GOOS != "windows" {
		os.Remove(target + ".old")
	}
	return nil
}

func download(ctx context.Context, rel KVResponse, dst string, onProgress func(read, total int64)) error {
	ctx, cancel := context.WithTimeout(ctx, binaryDownloadTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rel.URL, nil)
	if err != nil {
		return err
	}
	setCommonHeaders(req)
	req.Header.Set("Accept-Encoding", "identity") // keep the .gz bytes as published
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	h := sha256.New()
	var body io.Reader = resp.Body
	if onProgress != nil {
		body = &progressReader{r: resp.Body, total: resp.ContentLength, onProgress: onProgress}
	}
	_, copyErr := io.Copy(io.MultiWriter(f, h), body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != rel.SHA256 {
		return fmt.Errorf("hash mismatch: want %s, got %s", rel.SHA256, got)
	}
	return nil
}

type progressReader struct {
	r          io.Reader
	read       int64
	total      int64
	onProgress func(read, total int64)
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.read += int64(n)
	p.onProgress(p.read, p.total)
	return n, err
}

func setCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
}

func readVersionFile(target string) int {
	data, err := os.ReadFile(target + versionFileSuffix)
	if err != nil {
		return 0
	}
	var rel KVResponse
	if json.Unmarshal(data, &rel) != nil {
		return 0
	}
	return rel.Version
}

func writeVersionFile(target string, rel KVResponse) {
	data, _ := json.Marshal(rel)
	if err := os.WriteFile(target+versionFileSuffix, data, 0644); err != nil {
		log.Printf("[updater] write version file: %v", err)
	}
}

func extractGzip(src, dst string) error {
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	gr, err := gzip.NewReader(sf)
	if err != nil {
		return err
	}
	defer gr.Close()

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(df, gr); err != nil {
		df.Close()
		return err
	}
	return df.Close()
}
