package main

import (
	"compress/gzip"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/cron/v3"
	coreEnv "github.com/rpsoftech/golang-servers/env"
	env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup/env"
	interfaces "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-backup/interfaces"
	"github.com/rpsoftech/mysqldump"
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
	f, _ := os.Create(filepath.Join(c.BaseDir, fmt.Sprintf("%d.sql.gz", timeStamoForFileName)))
	// f, _ := os.Open(filepath.Join(c.BaseDir, "1749412432.sql.gz"))
	// b := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(f)
	err := mysqldump.Dump(
		c.MysqlDbConncetion.Db,
		c.ServerConfig.MysqlConfig.MYSQL_DATABASE,
		mysqldump.WithDropTable(),        // Option: Delete table before create (Default: Not delete table)
		mysqldump.WithData(),             // Option: Dump Data (Default: Only dump table schema)
		mysqldump.WithTables(),           // Option: Dump Tables (Default: All tables)
		mysqldump.WithWriter(gzipWriter), // Option: Writer (Default: os.Stdout)
	)
	gzipWriter.Close()
	f.Close()
	f, _ = os.Open(filepath.Join(c.BaseDir, fmt.Sprintf("%d.sql.gz", timeStamoForFileName)))
	defer f.Close()
	c.SFileServerConfig.Upload(f, c, c.ServerConfig.Name)
	if err != nil {
		println(err.Error())
	}
}
