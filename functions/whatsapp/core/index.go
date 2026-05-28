package whatsapp_core

import (
	"context"
	"fmt"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rpsoftech/golang-servers/env"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var sqlContainer *sqlstore.Container
var ctx = context.Background()

func InitSqlContainer() *sqlstore.Container {
	if sqlContainer == nil {

		dbLog := waLog.Stdout("Database", "WARN", true)
		// Make sure you add appropriate DB connector imports, e.g. github.com/mattn/go-sqlite3 for SQLite
		var err error
		sqlContainer, err = sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", filepath.Join(env.FindAndReturnCurrentDir(), "WhatsappSuperSecrete.db")), dbLog)
		if err != nil {
			panic(err)
		}
	}
	return sqlContainer
}

func ConnectToNumber(jidString string, token string, sqlContainer *sqlstore.Container) {
	// SqlContainer.PutDevice()
	if deviceStores, _ := sqlContainer.GetAllDevices(ctx); true {
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
		deviceStore, err = sqlContainer.GetDevice(ctx, JID)
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
		Client:           client,
		ConnectionStatus: 0,
		SyncFinished:     false,
		Token:            token,
		ParentData: &ParentData{
			DeviceStore:  deviceStore,
			SqlContainer: sqlContainer,
		}}
	client.AddEventHandler(connection.eventHandler)
	connection.ConnectAndGetQRCode()
}
