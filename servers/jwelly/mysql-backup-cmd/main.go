package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron/v3"
	coreEnv "github.com/rpsoftech/golang-servers/env"
	env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup-cmd/env"
	interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup-cmd/interfaces"
)

var CRON *cron.Cron

func main() {
	CRON = cron.New()
	var wg sync.WaitGroup

	for _, v := range env.ConnectionConfig.ServerConfig {
		cfg := v // Fix loop variable pointer bug
		cccc := &interfaces.ConfigWithConnection{ServerConfig: &cfg}

		if err := interfaces.ValidateAllConnectionsAndAssign(cccc); err != nil {
			log.Fatalf("Error In Validating Connection %s: %v", cfg.Name, err)
		}

		if env.ServerEnv.IsDev && env.ServerEnv.Env.APP_ENV == coreEnv.APP_ENV_LOCAL {
			wg.Go(func() {
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

	if env.ServerEnv.IsDev && env.ServerEnv.Env.APP_ENV == coreEnv.APP_ENV_LOCAL {
		wg.Wait()
		os.Exit(0)
	}

	CRON.Start()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	os.Exit(1)
}

func DoBackupAndUpload(c *interfaces.ConfigWithConnection) error {
	filename := fmt.Sprintf("%s-%d.sql.gz", c.ServerConfig.Name, time.Now().Unix())
	log.Printf("Starting backup for %s\n", c.ServerConfig.Name)

	// Setup pipeline: mysqldump -> gzip -> pipeWriter | pipeReader -> HTTP Request
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
	cmd.Stderr = os.Stderr // Route stderr to capture mysqldump errors

	// Run mysqldump in a goroutine so it streams concurrently with the upload
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
