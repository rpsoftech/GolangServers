// Command deploy is the single release pipeline for every component in
// components.go. It builds, signs and publishes; the same command runs on a
// developer machine and in .github/workflows/release.yml.
//
//	go run ./utility/deploy -component whatsapp-server -env STAGING
//	go run ./utility/deploy -component all -env PRODUCTION
//	go run ./utility/deploy -component soham-whatsapp-client -env PRODUCTION -version 48
//	go run ./utility/deploy -component all -dry-run   # build only, no upload
//	go run ./utility/deploy -component all -parallel 2 # build 2 components per target at once (default 3)
//	go run ./utility/deploy keygen                 # one-time signing key setup
//	go run ./utility/deploy affected <before> <after> # components changed between two commits
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
	"sync"
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
	// parallel is how many components of one target build at once.
	parallel int
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "keygen" {
		keygen()
		return
	}

	if len(os.Args) == 4 && os.Args[1] == "affected" {
		if err := affected(os.Args[2], os.Args[3]); err != nil {
			log.Fatal(err)
		}
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

	if err := releaseAll(opts, signingKey, fileServerToken, kvToken); err != nil {
		log.Fatalf("❌ %v", err)
	}
}

func parseFlags() options {
	componentFlag := flag.String("component", "", `component name(s), comma separated, or "all"`)
	envFlag := flag.String("env", os.Getenv("DEPLOY_ENV"), "release channel: "+strings.Join(allowedEnvs, ", ")+" (default $DEPLOY_ENV, else STAGING)")
	versionFlag := flag.Int("version", 0, "version number to publish; 0 = next after the highest published")
	dryRun := flag.Bool("dry-run", false, "build, compress and hash only; no signing, upload or KV update")
	parallelFlag := flag.Int("parallel", 3, "how many components of one target to build at once")
	flag.Parse()

	var opts options
	opts.dryRun = *dryRun
	opts.version = *versionFlag
	opts.parallel = max(1, *parallelFlag)

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

// artifact is one built, compressed binary waiting to be published.
type artifact struct {
	comp    Component
	target  Target
	version int
	binName string
	gzPath  string
	hash    string
}

// buildJob is one component build for a target.
type buildJob struct {
	comp    Component
	version int
}

// releaseAll resolves versions, builds everything, then publishes.
//
// Builds are grouped by target and the targets run one after another. Go
// caches compiled packages per GOOS/GOARCH, so the first build of a target
// compiles the shared libraries alone, then the remaining components of that
// target build in parallel on the warm cache. Nothing is published unless
// every build succeeded.
func releaseAll(opts options, signingKey ed25519.PrivateKey, fileServerToken, kvToken string) error {
	versions, err := resolveVersions(opts)
	if err != nil {
		return err
	}
	for _, comp := range opts.components {
		log.Printf("🚀 %s v%d → %s (%d targets)", comp.Name, versions[comp.Name], opts.env, len(comp.Targets))
	}

	// Group by target, keeping first-seen order.
	var order []Target
	groups := map[Target][]buildJob{}
	for _, comp := range opts.components {
		for _, t := range comp.Targets {
			if _, ok := groups[t]; !ok {
				order = append(order, t)
			}
			groups[t] = append(groups[t], buildJob{comp, versions[comp.Name]})
		}
	}

	var (
		mu        sync.Mutex
		artifacts []artifact
	)
	build := func(t Target, job buildJob) error {
		art, err := buildArtifact(job.comp, t, job.version, opts)
		if err != nil {
			return fmt.Errorf("%s %s: %w", job.comp.Name, t, err)
		}
		mu.Lock()
		artifacts = append(artifacts, art)
		mu.Unlock()
		return nil
	}
	for _, t := range order {
		jobs := groups[t]
		// First build alone: parallel builds of one target would all compile
		// the same libraries at once instead of sharing the cache.
		if err := build(t, jobs[0]); err != nil {
			return err
		}
		rest := jobs[1:]
		if err := runLimited(opts.parallel, len(rest), func(i int) error { return build(t, rest[i]) }); err != nil {
			return err
		}
	}
	if opts.dryRun {
		return nil
	}

	// Uploads are network bound, so more of them run at once than builds.
	return runLimited(max(opts.parallel, 4), len(artifacts), func(i int) error {
		a := artifacts[i]
		if err := publishArtifact(a, opts.env, signingKey, fileServerToken, kvToken); err != nil {
			return fmt.Errorf("%s %s: %w", a.comp.Name, a.target, err)
		}
		return nil
	})
}

// runLimited calls fn(0..n-1) with at most limit running at once. After the
// first error it starts no new calls, waits for the running ones and returns
// every error seen.
func runLimited(limit, n int, fn func(i int) error) error {
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		errs   []error
		failed bool
		sem    = make(chan struct{}, limit)
	)
	for i := range n {
		sem <- struct{}{}
		mu.Lock()
		stop := failed
		mu.Unlock()
		if stop {
			<-sem
			break
		}
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(i); err != nil {
				mu.Lock()
				errs = append(errs, err)
				failed = true
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return errors.Join(errs...)
}

var logMu sync.Mutex

// buildArtifact builds, compresses and hashes one component for one target.
func buildArtifact(comp Component, t Target, version int, opts options) (artifact, error) {
	binName := comp.Binary
	if t.OS == "windows" {
		binName += ".exe"
	}
	outDir := filepath.Join(buildOutputRoot, comp.Name, opts.env, t.String())
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return artifact{}, err
	}
	binPath := filepath.Join(outDir, binName)

	// 1. Build. No CGO, so one machine cross-compiles every target. Output is
	// captured and printed in one piece so parallel builds do not interleave.
	ldflags := fmt.Sprintf("-s -w -X main.version=%d -X %s=%s", version, buildEnvLDFlag, opts.env)
	cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", binPath, comp.Package)
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+t.OS, "GOARCH="+t.Arch)
	start := time.Now()
	out, err := cmd.CombinedOutput()
	if len(out) > 0 {
		logMu.Lock()
		os.Stdout.Write(out)
		logMu.Unlock()
	}
	if err != nil {
		return artifact{}, fmt.Errorf("build: %w", err)
	}

	// 2. Compress and hash.
	gzPath := binPath + ".gz"
	if err := compress(binPath, gzPath); err != nil {
		return artifact{}, fmt.Errorf("compress: %w", err)
	}
	hash, err := hashFile(gzPath)
	if err != nil {
		return artifact{}, fmt.Errorf("hash: %w", err)
	}
	log.Printf("📦 %s %s built in %s sha256=%s", comp.Name, t, time.Since(start).Round(100*time.Millisecond), hash)
	return artifact{comp, t, version, binName, gzPath, hash}, nil
}

// publishArtifact uploads to <component>/<ENV>/<os>_<arch>/<binary>.gz, then
// signs and writes every KV key. Each key gets its own signature because the
// key name is part of the signed message.
func publishArtifact(a artifact, env string, signingKey ed25519.PrivateKey, fileServerToken, kvToken string) error {
	uploadFolder := fmt.Sprintf("%s/%s/%s", a.comp.Name, env, a.target)
	uploadName := a.binName + ".gz"
	if err := uploadFile(a.gzPath, uploadName, uploadFolder, fileServerToken); err != nil {
		return fmt.Errorf("upload: %w", err)
	}
	fileURL := FileStaticURL + uploadFolder + "/" + uploadName

	for _, kvKey := range kvKeys(a.comp, env, a.target) {
		sig := ed25519.Sign(signingKey, updater.SignedMessage(kvKey, a.version, a.hash))
		entry := updater.KVResponse{Version: a.version, URL: fileURL, SHA256: a.hash, Signature: base64.StdEncoding.EncodeToString(sig)}
		if err := putKV(kvKey, entry, kvToken); err != nil {
			return fmt.Errorf("publish %s: %w", kvKey, err)
		}
		log.Printf("✅ %s = v%d", kvKey, a.version)
	}
	return nil
}

// resolveVersions picks a version for every component, looking them up in
// parallel.
func resolveVersions(opts options) (map[string]int, error) {
	versions := make(map[string]int, len(opts.components))
	var mu sync.Mutex
	err := runLimited(8, len(opts.components), func(i int) error {
		comp := opts.components[i]
		v, err := resolveVersion(comp, opts)
		if err != nil {
			return fmt.Errorf("%s: %w", comp.Name, err)
		}
		mu.Lock()
		versions[comp.Name] = v
		mu.Unlock()
		return nil
	})
	return versions, err
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
