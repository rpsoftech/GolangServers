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
	tlsCert := fs.String("tls-cert", "", "PEM certificate file for built-in TLS (reloaded when it changes or on SIGHUP)")
	tlsKey := fs.String("tls-key", "", "PEM private key file for built-in TLS")
	acmeDomain := fs.String("acme-domain", "", "obtain a Let's Encrypt certificate for this domain (needs port 80 and 443)")
	acmeHTTP := fs.String("acme-http", "", "plain HTTP listener that redirects to HTTPS and answers ACME HTTP-01 (default :80 with --acme-domain)")
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

	topts := TLSOptions{CertFile: cfg.TLSCert, KeyFile: cfg.TLSKey, ACMEDomain: cfg.ACMEDomain, ACMEHTTP: cfg.ACMEHTTP}
	if *tlsCert != "" {
		topts.CertFile = *tlsCert
	}
	if *tlsKey != "" {
		topts.KeyFile = *tlsKey
	}
	if *acmeDomain != "" {
		topts.ACMEDomain = *acmeDomain
	}
	if *acmeHTTP != "" {
		topts.ACMEHTTP = *acmeHTTP
	}
	if err := topts.Validate(); err != nil {
		return err
	}
	var tsetup *tlsSetup
	if topts.Enabled() {
		if tsetup, err = newTLSSetup(topts, home, cfg.Listen); err != nil {
			return err
		}
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
	handler := api.Handler()
	srv := &http.Server{Addr: cfg.Listen, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	errCh := make(chan error, 2)
	var httpSrv *http.Server
	if tsetup != nil {
		srv.Handler, srv.TLSConfig = hsts(handler), tsetup.config
		httpSrv = tsetup.httpSrv
		go func() { errCh <- srv.ListenAndServeTLS("", "") }()
		if httpSrv != nil {
			go func() { errCh <- httpSrv.ListenAndServe() }()
		}
	} else {
		go func() { errCh <- srv.ListenAndServe() }()
	}
	for _, w := range Warnings(topts, trustProxy, cfg.Listen) {
		log.Print("WARNING: ", w)
	}
	log.Printf("pm2go hub listening on %s, mode: %s", cfg.Listen, Mode(topts, trustProxy))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
wait:
	for {
		select {
		case err := <-errCh:
			close(stop)
			return err
		case sig := <-sigs:
			if sig == syscall.SIGHUP {
				if tsetup != nil && tsetup.reload != nil {
					if err := tsetup.reload(); err != nil {
						log.Printf("pm2go hub: certificate reload failed, keeping the old one: %v", err)
					} else {
						log.Print("pm2go hub: TLS certificate reloaded")
					}
				}
				continue
			}
			break wait
		}
	}
	close(stop)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if httpSrv != nil {
		httpSrv.Shutdown(ctx)
	}
	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}
