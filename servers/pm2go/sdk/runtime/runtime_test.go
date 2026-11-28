package runtime

import (
	"testing"
)

func TestFallbackListenerWithoutPM2GO(t *testing.T) {
	t.Setenv("LISTEN_FDS", "")
	ln, err := ListenerOn("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ln.Close()
}

func TestReadyIsNoopWithoutPipe(t *testing.T) {
	t.Setenv("PMGO_READY_FD", "")
	if err := Ready(); err != nil {
		t.Fatal(err)
	}
}
