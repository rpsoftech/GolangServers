package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron/v3"
	coreEnv "github.com/rpsoftech/golang-servers/env"
	env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup-cmd/env"
	interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup-cmd/interfaces"
	"github.com/rpsoftech/golang-servers/utility/updater"
)

var CRON *cron.Cron
var version = "1" // Injected via -ldflags during build

func main() {
	CRON = cron.New()
	var wg sync.WaitGroup

	// 1. Create a cancellable context listening for OS termination signals (or OTA trigger)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 2. The 5-Minute OTA Updater Daemon
	if !env.ServerEnv.IsDev || env.ServerEnv.Env.APP_ENV != coreEnv.APP_ENV_LOCAL {
		go func(versionStr string, workerCtx context.Context, triggerRestart context.CancelFunc) {
			currentVersion, _ := strconv.Atoi(versionStr)
			envName := string(env.ServerEnv.Env.APP_ENV)
			if envName == "" {
				envName = "PRODUCTION"
			}

			runCheck := func() {
				updated, err := updater.CheckAndUpdate(envName, "https://keyvalue.rpso.in/public/", "mysql_backup", currentVersion)
				if err != nil {
					log.Printf("⚠️ OTA Updater: %v\n", err)
					return
				}

				if updated {
					log.Println("🔄 OTA Update applied successfully! Triggering graceful restart...")
					// Instantly alerts <-ctx.Done() on the main thread
					triggerRestart()
				}
			}

			// Run immediately on boot
			runCheck()

			// Schedule to run exactly every 5 minutes
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()

			for {
				select {
				case <-workerCtx.Done(): // Stop checking for updates if shutting down
					return
				case <-ticker.C:
					runCheck()
				}
			}
		}(version, ctx, cancel)
	}

	// 3. Setup Connections and Cron Jobs
	for _, v := range env.ConnectionConfig.ServerConfig {
		cfg := v // Fix loop variable pointer bug
		cccc := &interfaces.ConfigWithConnection{ServerConfig: &cfg}

		if err := interfaces.ValidateAllConnectionsAndAssign(cccc); err != nil {
			log.Fatalf("Error In Validating Connection %s: %v", cfg.Name, err)
		}

		if env.ServerEnv.IsDev && env.ServerEnv.Env.APP_ENV == coreEnv.APP_ENV_LOCAL {
			wg.Go(func() {

				defer wg.Done()
				if err := DoBackupAndUpload(cccc); err != nil {
					log.Printf("Backup failed for %s: %v", cfg.Name, err)
				}
			})
		} else {
			CRON.AddFunc(cfg.Cron, func() {
				if err := DoBackupAndUpload(cccc); err != nil {
					log.Printf("Cron backup failed for %s: %v", cfg.Name, err)
				}
			})
		}
	}

	// 4. Handle Local Dev (One-off Execution)
	if env.ServerEnv.IsDev && env.ServerEnv.Env.APP_ENV == coreEnv.APP_ENV_LOCAL {
		wg.Wait()
		log.Println("✅ Local Dev backups completed. Exiting.")
		return
	}

	CRON.Start()
	log.Println("🚀 Backup Cron Daemon Started...")

	// 5. Block main thread until OS termination signal OR an OTA Update triggers `cancel()`
	<-ctx.Done()
	log.Println("🛑 Termination signal received. Initiating graceful shutdown...")

	// 6. Graceful Shutdown
	// CRON.Stop() stops accepting new jobs and returns a context that fires when running jobs finish.
	// This prevents a backup database dump from being corrupted mid-flight if an OTA update triggers.
	cronCtx := CRON.Stop()
	<-cronCtx.Done()

	log.Println("✅ mysql-backup-cmd shut down safely.")
}

func DoBackupAndUpload(c *interfaces.ConfigWithConnection) error {
	filename := fmt.Sprintf("%s-%d.sql.gz", c.ServerConfig.Name, time.Now().Unix())
	log.Printf("Starting backup for %s\n", c.ServerConfig.Name)

	pr, pw := io.Pipe()
	gzipWriter := gzip.NewWriter(pw)

	cmdArray := []string{"-h", c.ServerConfig.MysqlConfig.MYSQL_HOST}
	if c.ServerConfig.MysqlConfig.MYSQL_PORT != 3306 {
		cmdArray = append(cmdArray, fmt.Sprintf("--port=%d", c.ServerConfig.MysqlConfig.MYSQL_PORT))
	}
	if c.ServerConfig.MysqlConfig.MYSQL_USERNAME != "" {
		cmdArray = append(cmdArray, "-u", c.ServerConfig.MysqlConfig.MYSQL_USERNAME)
	}
	if c.ServerConfig.MysqlConfig.MYSQL_PASSWORD != "" {
		cmdArray = append(cmdArray, "-p"+c.ServerConfig.MysqlConfig.MYSQL_PASSWORD)
	}
	cmdArray = append(cmdArray, "--add-drop-table", "--single-transaction", c.ServerConfig.MysqlConfig.MYSQL_DATABASE)

	cmd := exec.Command("mysqldump", cmdArray...)
	cmd.Stdout = gzipWriter
	cmd.Stderr = os.Stderr

	go func() {
		defer pw.Close()
		defer gzipWriter.Close()

		gzipWriter.Write([]byte("SET foreign_key_checks = 0;\n"))
		if err := cmd.Run(); err != nil {
			log.Printf("mysqldump failed: %v", err)
		}
		gzipWriter.Write([]byte("SET foreign_key_checks = 1;\n"))
	}()

	log.Printf("Uploading %s...\n", filename)
	err := c.SFileServerConfig.Upload(pr, filename, c)
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	log.Printf("Backup & Upload for %s completed successfully\n", c.ServerConfig.Name)
	return nil
}
