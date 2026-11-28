package cli

import "testing"

func TestStartFlagsListenAndWaitReady(t *testing.T) {
	fs, o := newStartFlagSet()
	pos, rest, err := parseInterspersed(fs, []string{"--listen", ":8080", "app.sh", "--wait-ready", "--listen-timeout", "5000"})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := buildStartConfig(o, pos[0], rest)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Listen != ":8080" || !cfg.WaitReady || cfg.ListenTimeoutMs != 5000 {
		t.Fatalf("listen=%q waitReady=%v timeout=%d", cfg.Listen, cfg.WaitReady, cfg.ListenTimeoutMs)
	}
}
