package whatsapp_core

import (
	"context"
	"fmt"
	"path/filepath"

	// Pure Go SQLite driver (registers "sqlite"), so builds need no CGO.
	_ "modernc.org/sqlite"
	"github.com/rpsoftech/golang-servers/env"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var sqlContainer *sqlstore.Container
var globalBackground = context.Background()

func InitSqlContainer() *sqlstore.Container {
	if sqlContainer == nil {

		dbLog := waLog.Stdout("Database", "WARN", true)
		// modernc.org/sqlite takes pragmas as _pragma=name(value); whatsmeow
		// needs foreign keys on. The on-disk format is unchanged from mattn.
		var err error
		sqlContainer, err = sqlstore.New(globalBackground, "sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.Join(env.FindAndReturnCurrentDir(), "WhatsappSuperSecrete.db")), dbLog)
		if err != nil {
			panic(err)
		}
	}
	return sqlContainer
}

func ConnectToNumber(jidString string, token string, sqlContainer *sqlstore.Container) {
	// SqlContainer.PutDevice()
	if deviceStores, _ := sqlContainer.GetAllDevices(globalBackground); true {
		for _, deviceStore := range deviceStores {
			println(deviceStore.ID.User)
		}
	}
	var JID types.JID
	if jidString != "" {
		JID, _ = types.ParseJID(jidString)
	}
	var deviceStore *store.Device
	if !JID.IsEmpty() {
		var err error
		// sqlContainer.DeleteDevice()
		deviceStore, err = sqlContainer.GetDevice(globalBackground, JID)
		if err != nil {
			println(err.Error())
		}
	}
	if deviceStore == nil {
		deviceStore = sqlContainer.NewDevice()
		// deviceStore = types.DEv(number, types.DefaultUserServer)
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.EnableAutoReconnect = true
	println(client.LastSuccessfulConnect.String())
	connection := &WhatsappConnection{
		client:           client,
		connectionStatus: 0,
		syncFinished:     false,
		Token:            token,
		ParentData: &ParentData{
			DeviceStore:  deviceStore,
			SqlContainer: sqlContainer,
		}}
	client.AddEventHandler(connection.eventHandler)
	connection.ConnectAndGetQRCode()
}
