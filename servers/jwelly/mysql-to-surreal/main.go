package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	coreEnv "github.com/rpsoftech/golang-servers/env"
	env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/env"
	mysql_to_surreal_functions "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/functions"
)

var DeferFunctionSlice []func() = []func(){}
var CRON *cron.Cron

func deferFunc() {
	for _, v := range DeferFunctionSlice {
		v()
	}
}

func main() {
	coreEnv.LoadEnv(".env-mysql-to-surreal")
	defer deferFunc()
	CRON = cron.New()
	InitaliseAndPopulateTheConnection()
}

func InitaliseAndPopulateTheConnection() {
	// v := mysql_to_surreal_env.ConnectionConfig.ServerConfig
	for _, v := range env.ConnectionConfig.ServerConfig {
		cccc := &mysql_to_surreal_functions.ConfigWithConnection{ServerConfig: &v, DbConnections: &mysql_to_surreal_functions.ConnectionsToDb{}}
		if err := mysql_to_surreal_functions.ValidateAllConnectionsAndAssign(cccc); err != nil {
			fmt.Printf("Error In Validating Connectino %s", v.Name)
			println(err.Error())
		} else {
			DeferFunctionSlice = append(DeferFunctionSlice, cccc.DbConnections.MysqlDbConncetion.DeferFunction, cccc.DbConnections.SurrealDbConncetion.DeferFunction)
		}
		CRON.AddFunc(v.Cron, func() {
			DoTheOperation(cccc)
		})
		if env.ServerEnv.IsDev && env.ServerEnv.Env.APP_ENV == coreEnv.APP_ENV_LOCAL {
			os.Exit(0)
		}
	}
}

func DoTheOperation(c *mysql_to_surreal_functions.ConfigWithConnection) {
	fmt.Printf("Operation of %s Started\n", c.ServerConfig.Name)
	if env.ServerEnv.RefreshDatabase {
		c.ClearSurrealDbAndInsert()
	}
	fmt.Printf("Tables Remvoed And Created For %s\n", c.ServerConfig.Name)
	startTime := time.Now()
	functions := []func(){
		c.ReadAndStoreCategory,
		c.ReadAndStoreStampTable,
		c.ReadAndStoreUnitTable,
		c.ReadAndStoreItemGroupTable,
		c.ReadAndStoreItemMast,
		c.ReadAndStoreTgm1Table,
		c.ReadAndStoreSiteTable,
		c.ReadAndStoreItemTransTable,
		c.ReadAndStoreAccMastTable,
		c.ReadAndStoreTrans1Table,
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(functions))
	for _, f := range functions {
		go func() {
			f()
			waitGroup.Done()
		}()
	}
	waitGroup.Wait()
	fmt.Printf("Operation of %s Completed in Duration of %s\n", c.ServerConfig.Name, time.Since(startTime))
}
