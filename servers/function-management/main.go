package main

import (
	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

func deferMainFunc() {
	println("Closing...")
	mongodb.DeferFunction()
	redis.DeferFunction()
}

func main() {
	defer deferMainFunc()
	env.LoadEnv("function-management.env")
	println("Function Management Server Started")

}
