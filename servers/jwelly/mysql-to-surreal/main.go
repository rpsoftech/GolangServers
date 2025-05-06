package main

import (
	"fmt"
	"sync"

	"github.com/rpsoftech/golang-servers/env"
	mysql_to_surreal_env "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/env"
	mysql_to_surreal_functions "github.com/rpsoftech/golang-servers/servers/jwelly/mysql-to-surreal/functions"
)

var DeferFunctionSlice []func() = []func(){}

func deferFunc() {
	for _, v := range DeferFunctionSlice {
		v()
	}
}

func main() {
	// defer deferFunc()
	InitaliseAndPopulateTheConnection()
}

func InitaliseAndPopulateTheConnection() {
	// v := mysql_to_surreal_env.ConnectionConfig.ServerConfig
	for _, v := range mysql_to_surreal_env.ConnectionConfig.ServerConfig {
		cccc := &mysql_to_surreal_functions.ConfigWithConnection{ServerConfig: &v, DbConnections: &mysql_to_surreal_functions.ConnectionsToDb{}}
		if err := mysql_to_surreal_functions.ValidateAllConnectionsAndAssign(cccc); err != nil {
			fmt.Printf("Error In Validating Connectino %s", v.Name)
			println(err.Error())
		} else {
			DeferFunctionSlice = append(DeferFunctionSlice, cccc.DbConnections.MysqlDbConncetion.DeferFunction, cccc.DbConnections.SurrealDbConncetion.DeferFunction)
		}
		if env.IsDev {
			DoTheOperation(cccc)
		}
	}
}

func DoTheOperation(c *mysql_to_surreal_functions.ConfigWithConnection) {
	functions := []func(){
		// c.ReadAndStoreTgMaster,
		c.ReadAndStoreCategory,
		c.ReadAndStoreStampTable,
		c.ReadAndStoreUnitTable,
		c.ReadAndStoreItemGroupTable,
		c.ReadAndStoreItemMast,
		c.ReadAndStoreTgm1Table,
	}
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(functions))
	defer waitGroup.Wait()
	// go func() {
	// waitGroup.Done()
	// }()
	// go func() {
	// waitGroup.Done()
	// }()
	for _, f := range functions {
		go func() {
			f()
			waitGroup.Done()
		}()
	}
}
