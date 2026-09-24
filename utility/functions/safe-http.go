package utility_functions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"syscall"
	"time"
)

// MaxUserURLDownloadBytes caps how much FetchUserURL reads into memory.
const MaxUserURLDownloadBytes int64 = 100 << 20 // 100 MB

var (
	ErrBlockedAddress  = errors.New("destination address is not allowed")
	ErrDownloadTooBig  = fmt.Errorf("download is larger than %d bytes", MaxUserURLDownloadBytes)
	cgnatPrefix        = netip.MustParsePrefix("100.64.0.0/10")
	userURLHTTPTimeout = 2 * time.Minute
)

// userURLClient fetches URLs supplied by API callers. The dialer checks every
// resolved IP right before connecting, so redirects and DNS rebinding cannot
// reach loopback, private, link-local (cloud metadata) or CGNAT addresses.
var userURLClient = &http.Client{
	Timeout: userURLHTTPTimeout,
	Transport: &http.Transport{
		// No proxy: a proxy would make the dial-time IP check see only the proxy.
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
			Control: blockNonPublicAddress,
		}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
	},
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("stopped after 5 redirects")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return fmt.Errorf("redirect to scheme %q is not allowed", req.URL.Scheme)
		}
		return nil
	},
}

func blockNonPublicAddress(_ string, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return err
	}
	addr = addr.Unmap()
	if !addr.IsGlobalUnicast() || addr.IsPrivate() || cgnatPrefix.Contains(addr) {
		return fmt.Errorf("%w: %s", ErrBlockedAddress, addr)
	}
	return nil
}

// FetchUserURL downloads an http(s) URL supplied by an API caller. It only
// connects to public addresses and reads at most MaxUserURLDownloadBytes.
func FetchUserURL(ctx context.Context, rawURL string) ([]byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("URL scheme %q is not allowed", parsedURL.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := userURLClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file, HTTP status code: %d", resp.StatusCode)
	}
	if resp.ContentLength > MaxUserURLDownloadBytes {
		return nil, ErrDownloadTooBig
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxUserURLDownloadBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if int64(len(data)) > MaxUserURLDownloadBytes {
		return nil, ErrDownloadTooBig
	}
	return data, nil
}
