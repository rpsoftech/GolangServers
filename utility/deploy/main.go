// Command deploy is the single release pipeline for every component in
// components.go. It builds, signs and publishes; the same command runs on a
// developer machine and in .github/workflows/release.yml.
//
//	go run ./utility/deploy -component whatsapp-server -env STAGING
//	go run ./utility/deploy -component all -env PRODUCTION
//	go run ./utility/deploy -component soham-whatsapp-client -env PRODUCTION -version 48
//	go run ./utility/deploy -component all -dry-run   # build only, no upload
//	go run ./utility/deploy keygen                    # one-time signing key setup
package main

import (
	"bytes"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/rpsoftech/golang-servers/utility/updater"
)

const (
	FileServerURL   = "https://files.rpso.in/upload/"
	FileStaticURL   = "https://files.rpso.in/static/"
	KeyValueURL     = updater.DefaultKVBaseURL
	buildEnvLDFlag  = "github.com/rpsoftech/golang-servers/utility/updater.BuildEnv"
	productionEnv   = "PRODUCTION"
	buildOutputRoot = "build"
)

var allowedEnvs = []string{"PRODUCTION", "STAGING", "DEVELOP"}

type options struct {
	components []Component
	env        string
	version    int
	dryRun     bool
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "keygen" {
		keygen()
		return
	}

	opts := parseFlags()

	fileServerToken := os.Getenv("FILE_SERVER_TOKEN")
	kvToken := os.Getenv("KV_TOKEN")
	var signingKey ed25519.PrivateKey
	if !opts.dryRun {
		if fileServerToken == "" || kvToken == "" {
			log.Fatal("FATAL: FILE_SERVER_TOKEN and KV_TOKEN environment variables are missing")
		}
		// Fail before building anything if releases cannot be signed with the
		// key the shipped updater trusts.
		var err error
		signingKey, err = updater.ParseSigningKey(os.Getenv("UPDATE_SIGNING_KEY"))
		if err != nil {
			log.Fatalf("FATAL: UPDATE_SIGNING_KEY: %v (run `go run ./utility/deploy keygen` once)", err)
		}
	}

	for _, comp := range opts.components {
		if err := release(comp, opts, signingKey, fileServerToken, kvToken); err != nil {
			log.Fatalf("❌ %s: %v", comp.Name, err)
		}
	}
}

func parseFlags() options {
	componentFlag := flag.String("component", "", `component name(s), comma separated, or "all"`)
	envFlag := flag.String("env", os.Getenv("DEPLOY_ENV"), "release channel: "+strings.Join(allowedEnvs, ", ")+" (default $DEPLOY_ENV, else STAGING)")
	versionFlag := flag.Int("version", 0, "version number to publish; 0 = next after the highest published")
	dryRun := flag.Bool("dry-run", false, "build, compress and hash only; no signing, upload or KV update")
	flag.Parse()

	var opts options
	opts.dryRun = *dryRun
	opts.version = *versionFlag

	opts.env = strings.ToUpper(strings.TrimSpace(*envFlag))
	if opts.env == "" {
		opts.env = "STAGING"
		log.Println("⚠️ -env not set. Defaulting to STAGING.")
	}
	if !slices.Contains(allowedEnvs, opts.env) {
		log.Fatalf("FATAL: -env %q is not one of %s", opts.env, strings.Join(allowedEnvs, ", "))
	}

	switch name := strings.TrimSpace(*componentFlag); name {
	case "":
		log.Fatalf("FATAL: -component is required. Known: %s, or all", knownComponentNames())
	case "all":
		opts.components = components
	default:
		for _, n := range strings.Split(name, ",") {
			c, ok := findComponent(strings.TrimSpace(n))
			if !ok {
				log.Fatalf("FATAL: unknown component %q. Known: %s", n, knownComponentNames())
			}
			opts.components = append(opts.components, c)
		}
	}
	if opts.version != 0 && len(opts.components) > 1 {
		log.Fatal("FATAL: -version only makes sense with a single -component")
	}
	return opts
}

func knownComponentNames() string {
	names := make([]string, len(components))
	for i, c := range components {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}

// kvKeys lists every KV key one target is published under.
func kvKeys(comp Component, envName string, t Target) []string {
	keys := []string{updater.GetFileKey(envName, comp.Name, t.OS, t.Arch)}
	if comp.LegacyKey != nil && envName == productionEnv {
		keys = append(keys, comp.LegacyKey(t))
	}
	return keys
}

// release builds every target of comp and publishes them under one version.
func release(comp Component, opts options, signingKey ed25519.PrivateKey, fileServerToken, kvToken string) error {
	version, err := resolveVersion(comp, opts)
	if err != nil {
		return err
	}
	log.Printf("🚀 %s v%d → %s (%d targets)", comp.Name, version, opts.env, len(comp.Targets))

	for _, t := range comp.Targets {
		binName := comp.Binary
		if t.OS == "windows" {
			binName += ".exe"
		}
		outDir := filepath.Join(buildOutputRoot, comp.Name, opts.env, t.String())
		if err := os.MkdirAll(outDir, 0755); err != nil {
			return err
		}
		binPath := filepath.Join(outDir, binName)

		// 1. Build. No CGO, so one machine cross-compiles every target.
		ldflags := fmt.Sprintf("-s -w -X main.version=%d -X %s=%s", version, buildEnvLDFlag, opts.env)
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", binPath, comp.Package)
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+t.OS, "GOARCH="+t.Arch)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("build %s: %w", t, err)
		}

		// 2. Compress and hash.
		gzPath := binPath + ".gz"
		if err := compress(binPath, gzPath); err != nil {
			return fmt.Errorf("compress %s: %w", t, err)
		}
		hash, err := hashFile(gzPath)
		if err != nil {
			return fmt.Errorf("hash %s: %w", t, err)
		}
		log.Printf("📦 %s %s sha256=%s", comp.Name, t, hash)
		if opts.dryRun {
			continue
		}

		// 3. Upload to <component>/<ENV>/<os>_<arch>/<binary>.gz.
		uploadFolder := fmt.Sprintf("%s/%s/%s", comp.Name, opts.env, t)
		uploadName := binName + ".gz"
		if err := uploadFile(gzPath, uploadName, uploadFolder, fileServerToken); err != nil {
			return fmt.Errorf("upload %s: %w", t, err)
		}
		fileURL := FileStaticURL + uploadFolder + "/" + uploadName

		// 4. Sign and publish. Each KV key gets its own signature because the
		// key name is part of the signed message.
		for _, kvKey := range kvKeys(comp, opts.env, t) {
			sig := ed25519.Sign(signingKey, updater.SignedMessage(kvKey, version, hash))
			entry := updater.KVResponse{Version: version, URL: fileURL, SHA256: hash, Signature: base64.StdEncoding.EncodeToString(sig)}
			if err := putKV(kvKey, entry, kvToken); err != nil {
				return fmt.Errorf("publish %s: %w", kvKey, err)
			}
			log.Printf("✅ %s = v%d", kvKey, version)
		}
	}
	return nil
}

// resolveVersion picks the version for this release. Updaters only move to a
// higher version, so the new one must be above every key it will overwrite,
// legacy keys included.
func resolveVersion(comp Component, opts options) (int, error) {
	highest := 0
	for _, t := range comp.Targets {
		for _, kvKey := range kvKeys(comp, opts.env, t) {
			v, err := getPublishedVersion(kvKey)
			if err != nil {
				if opts.dryRun {
					log.Printf("⚠️ %s: %v (ignored in dry run)", kvKey, err)
					continue
				}
				return 0, err
			}
			highest = max(highest, v)
		}
	}
	if opts.version == 0 {
		return highest + 1, nil
	}
	if opts.version <= highest {
		return 0, fmt.Errorf("-version %d is not above the published v%d; installs would ignore it", opts.version, highest)
	}
	return opts.version, nil
}

// getPublishedVersion returns the version stored under kvKey, 0 when nothing
// is published yet. Any other failure is an error: guessing low here would
// publish a version every install ignores.
func getPublishedVersion(kvKey string) (int, error) {
	req, err := http.NewRequest(http.MethodGet, KeyValueURL+kvKey, nil)
	if err != nil {
		return 0, err
	}
	setCommonHeaders(req)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", kvKey, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("read %s: HTTP %d", kvKey, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", kvKey, err)
	}
	if len(bytes.TrimSpace(body)) == 0 || string(bytes.TrimSpace(body)) == "null" {
		return 0, nil
	}
	var entry updater.KVResponse
	if err := json.Unmarshal(body, &entry); err != nil {
		return 0, fmt.Errorf("read %s: unexpected body: %w", kvKey, err)
	}
	return entry.Version, nil
}

func putKV(kvKey string, entry updater.KVResponse, kvToken string) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, KeyValueURL+kvKey, bytes.NewReader(data))
	if err != nil {
		return err
	}
	setCommonHeaders(req)
	req.Header.Set("Authorization", "Bearer "+kvToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("KV returned %d: %s", resp.StatusCode, body)
	}
	return nil
}

// keygen prints a new release signing key pair. Paste the public key into
// updater.ReleasePublicKey and store the private key as UPDATE_SIGNING_KEY on
// the deploy machine and in CI secrets only. Never commit the private key.
func keygen() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("keygen failed: %v", err)
	}
	fmt.Println("Public key  (paste into utility/updater/signing.go ReleasePublicKey):")
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	fmt.Println()
	fmt.Println("Private key (set as UPDATE_SIGNING_KEY; keep secret, do not commit):")
	fmt.Println(base64.StdEncoding.EncodeToString(priv))
}

func compress(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	w := gzip.NewWriter(out)
	if _, err := io.Copy(w, in); err != nil {
		out.Close()
		return err
	}
	// Close the gzip writer before the file so the trailer is written.
	return errors.Join(w.Close(), out.Close())
}

func hashFile(src string) (string, error) {
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func uploadFile(path, filename, uploadPathFolder, token string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)

	part, err := writer.CreateFormFile(filename, filename)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.WriteField("path", uploadPathFolder); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, FileServerURL+filename, payload)
	if err != nil {
		return err
	}
	setCommonHeaders(req)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func setCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
}
