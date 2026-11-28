// pm2go is an internal process manager modelled on PM2.
package main

import (
	"os"

	"github.com/rpsoftech/golang-servers/servers/pm2go/cli"
)

func main() { os.Exit(cli.Main(os.Args[1:])) }
