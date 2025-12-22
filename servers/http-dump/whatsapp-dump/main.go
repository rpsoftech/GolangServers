package whatsappdump

import (
	_ "github.com/rpsoftech/golang-servers/servers/http-dump/whatsapp-dump/env"
	"github.com/rpsoftech/golang-servers/utility/mongodb"
	"github.com/rpsoftech/golang-servers/utility/redis"
)

func deferMainFunc() {
	println("Closing...")
	redis.DeferFunction()
	mongodb.DeferFunction()
}
func main() {
	defer deferMainFunc()
	println("WhatsApp Dump Server Started")

}
