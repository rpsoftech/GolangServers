package main

import (
	"compress/gzip"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron/v3"
	coreEnv "github.com/rpsoftech/golang-servers/env"
	env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup-cmd/env"
	interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup-cmd/interfaces"
)

var DeferFunctionSlice []func() = []func(){}
var CRON *cron.Cron

func deferFunc() {
	println("deferFunc")
	for _, v := range DeferFunctionSlice {
		v()
	}
}

func main() {
	// env.LoadEnv("telegram-server.env")
	CRON = cron.New()
	for _, v := range env.ConnectionConfig.ServerConfig {
		cccc := &interfaces.ConfigWithConnection{ServerConfig: &v}
		if err := interfaces.ValidateAllConnectionsAndAssign(cccc); err != nil {
			fmt.Printf("Error In Validating Connectino %s", v.Name)
			println(err.Error())
		}
		if env.ServerEnv.IsDev && env.ServerEnv.Env.APP_ENV == coreEnv.APP_ENV_LOCAL {
			DoBackupAndUpload(cccc)
			os.Exit(0)
		} else {
			CRON.AddFunc(v.Cron, func() {
				DoBackupAndUpload(cccc)
			})
		}
	}
	CRON.Start()
	// defer deferFunc()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	deferFunc()
	os.Exit(1)
}

func DoBackupAndUpload(c *interfaces.ConfigWithConnection) {
	timeStamoForFileName := time.Now().Unix()
	filename := fmt.Sprintf("%s-%d.sql.gz", c.ServerConfig.Name, timeStamoForFileName)
	f, _ := os.Create(filepath.Join(c.BaseDir, filename))

	fmt.Printf("Starting backup for %s at %s \n", c.ServerConfig.Name, time.Now().Format(time.RFC3339))
	gzipWriter := gzip.NewWriter(f)
	gzipWriter.Write([]byte("SET foreign_key_checks = 0;\n"))
	var err error
	// --add-drop-table --no-data --single-transaction
	cmd := exec.Command("mysqldump",
		"-h", c.ServerConfig.MysqlConfig.MYSQL_HOST,
		"-u", c.ServerConfig.MysqlConfig.MYSQL_USERNAME,
		"-p"+c.ServerConfig.MysqlConfig.MYSQL_PASSWORD,
		"--add-drop-table",
		"--single-transaction",
		c.ServerConfig.MysqlConfig.MYSQL_DATABASE)
	cmd.Stdout = gzipWriter
	err = cmd.Run()
	gzipWriter.Write([]byte("SET foreign_key_checks = 1;\n"))
	if err != nil {
		println(err.Error())
	}
	fmt.Printf("backup for %s at %s Completed\n", c.ServerConfig.Name, time.Now().Format(time.RFC3339))
	gzipWriter.Close()
	f.Close()
	if err != nil {
		println(err.Error())
	}
	f, _ = os.Open(filepath.Join(c.BaseDir, filename))
	defer f.Close()
	fmt.Printf("Uploading Backup File for %s at %s \n", c.ServerConfig.Name, time.Now().Format(time.RFC3339))
	c.SFileServerConfig.Upload(f, c, c.ServerConfig.Name)
	if err != nil {
		println(err.Error())
	}
}
