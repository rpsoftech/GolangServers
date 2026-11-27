package whatsapp_core

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"

	"github.com/rpsoftech/golang-servers/env"

	// Pure Go SQLite driver (registers "sqlite"), so builds need no CGO.
	_ "modernc.org/sqlite"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var (
	sqlContainer     *sqlstore.Container
	sqlContainerOnce sync.Once
	globalBackground = context.Background()
)

// InitSqlContainer opens the device store once. It is safe to call from
// several goroutines.
func InitSqlContainer() *sqlstore.Container {
	sqlContainerOnce.Do(func() {
		dbLog := waLog.Stdout("Database", "WARN", true)
		// modernc.org/sqlite takes pragmas as _pragma=name(value); whatsmeow
		// needs foreign keys on. The on-disk format is unchanged from mattn.
		container, err := sqlstore.New(globalBackground, "sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", filepath.Join(env.FindAndReturnCurrentDir(), "WhatsappSuperSecrete.db")), dbLog)
		if err != nil {
			panic(err)
		}
		sqlContainer = container
	})
	return sqlContainer
}

// ConnectToNumber builds a client for token and connects it. For a new
// device it blocks until the QR code is scanned or times out.
func ConnectToNumber(jidString string, token string, sqlContainer *sqlstore.Container) {
	newConnection(jidString, token, sqlContainer).ConnectAndGetQRCode()
}

// newConnection loads the stored device for jidString, or creates a new
// device when there is none, and wraps it in a WhatsappConnection. It does
// not connect.
func newConnection(jidString string, token string, sqlContainer *sqlstore.Container) *WhatsappConnection {
	var JID types.JID
	if jidString != "" {
		var err error
		JID, err = types.ParseJID(jidString)
		if err != nil {
			log.Printf("[whatsapp] token %s: invalid stored JID %q: %v", token, jidString, err)
		}
	}
	var deviceStore *store.Device
	if !JID.IsEmpty() {
		var err error
		deviceStore, err = sqlContainer.GetDevice(globalBackground, JID)
		if err != nil {
			log.Printf("[whatsapp] token %s: load device %s failed, a new QR scan will be needed: %v", token, JID, err)
		} else if deviceStore == nil {
			log.Printf("[whatsapp] token %s: device %s not found in store, a new QR scan will be needed", token, JID)
		}
	}
	if deviceStore == nil {
		deviceStore = sqlContainer.NewDevice()
	}

	clientLog := waLog.Stdout("Client", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.EnableAutoReconnect = true
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
	return connection
}
