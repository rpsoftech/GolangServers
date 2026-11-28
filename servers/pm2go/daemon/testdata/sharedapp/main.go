// Command sharedapp is a test application for the shared socket and wait_ready
// tests. It answers every request with its pid.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	pmrt "github.com/rpsoftech/golang-servers/servers/pm2go/sdk/runtime"
)

func main() {
	if ms, _ := strconv.Atoi(os.Getenv("APP_START_DELAY_MS")); ms > 0 {
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	ln, err := pmrt.Listener()
	if err != nil {
		fmt.Fprintln(os.Stderr, "listen:", err)
		os.Exit(1)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, os.Getpid())
	})}
	go srv.Serve(ln)
	if os.Getenv("APP_NO_READY") == "" {
		if err := pmrt.Ready(); err != nil {
			fmt.Fprintln(os.Stderr, "ready:", err)
		}
	}
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	<-sig
	// Stop accepting, then let a connection accepted at this very moment
	// register with the server before Shutdown looks for active ones.
	ln.Close()
	time.Sleep(200 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}
