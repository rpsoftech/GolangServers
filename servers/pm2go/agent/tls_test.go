package agent

import (
	"context"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/rpsoftech/golang-servers/servers/pm2go/common"
)

func TestAgentDialsWSSWithPrivateCA(t *testing.T) {
	up := websocket.Upgrader{}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err == nil {
			c.Close()
		}
	}))
	defer srv.Close()
	u, err := wsURL(srv.URL)
	if err != nil || u[:6] != "wss://" {
		t.Fatalf("wsURL = %q, %v", u, err)
	}

	// Without the CA the self-signed certificate is rejected.
	d, _ := dialer(common.LinkConfig{HubURL: srv.URL})
	if _, _, err := d.DialContext(context.Background(), u, nil); err == nil {
		t.Fatal("self-signed certificate must be rejected without --hub-ca")
	}

	// With the CA file the connection succeeds.
	ca := filepath.Join(t.TempDir(), "ca.pem")
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw})
	if err := os.WriteFile(ca, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	d, err = dialer(common.LinkConfig{HubURL: srv.URL, CAFile: ca})
	if err != nil {
		t.Fatal(err)
	}
	conn, _, err := d.DialContext(context.Background(), u, nil)
	if err != nil {
		t.Fatalf("dial with CA: %v", err)
	}
	conn.Close()

	// A bad CA file is an error, not a silent fallback.
	bad := filepath.Join(t.TempDir(), "bad.pem")
	os.WriteFile(bad, []byte("nope"), 0o600)
	if _, err := dialer(common.LinkConfig{CAFile: bad}); err == nil {
		t.Fatal("garbage CA file must fail")
	}
}
