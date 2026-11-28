package hub

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Run starts the hub. args are the flags after "hub".
func Run(args []string) error {
	fs := flag.NewFlagSet("hub", flag.ContinueOnError)
	listen := fs.String("listen", "", "listen address (default from config, :8080)")
	proxy := fs.Bool("trust-proxy", false, "trust X-Forwarded-For/Proto from a reverse proxy (Caddy)")
	showKeys := fs.Bool("show-keys", false, "print the admin password and agent key, then exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	home := Home()
	cfg, created, err := LoadConfig(home)
	if err != nil {
		return err
	}
	if *showKeys || created {
		fmt.Printf("admin password: %s\nagent key:      %s\nconfig:         %s/config.json\n", cfg.AdminPassword, cfg.AgentKey, home)
		if *showKeys {
			return nil
		}
	}
	trustProxy = *proxy || os.Getenv("PMGO_TRUST_PROXY") == "1"
	if *listen != "" {
		cfg.Listen = *listen
	}

	store, err := OpenStore(home)
	if err != nil {
		return err
	}
	defer store.Close()
	issues := NewIssues(store)
	reg := NewRegistry(store, issues)
	notif := NewNotifier(store)
	engine, err := NewEngine(store, reg, notif)
	if err != nil {
		return err
	}
	stop := make(chan struct{})
	go issues.RunFlusher(stop)
	go engine.Run(2*time.Second, stop)
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				if err := store.Prune(time.Duration(cfg.RetentionDays) * 24 * time.Hour); err != nil {
					log.Print(err)
				}
			}
		}
	}()

	api := &API{cfg: cfg, auth: newAuth(cfg), store: store, reg: reg, engine: engine, notif: notif}
	srv := &http.Server{Addr: cfg.Listen, Handler: api.Handler(), ReadHeaderTimeout: 10 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Printf("pm2go hub listening on %s", cfg.Listen)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		close(stop)
		return err
	case <-sigs:
	}
	close(stop)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}
