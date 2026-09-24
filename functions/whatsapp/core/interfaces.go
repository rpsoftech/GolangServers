package whatsapp_core

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/rpsoftech/golang-servers/env"
	whatsapp_functions "github.com/rpsoftech/golang-servers/functions/whatsapp"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
	"github.com/rpsoftech/golang-servers/interfaces"
	utility_functions "github.com/rpsoftech/golang-servers/utility/functions"
	whatsapp_utility "github.com/rpsoftech/golang-servers/utility/whatsapp/utility"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type (
	ParentData struct {
		DeviceStore  *store.Device
		SqlContainer *sqlstore.Container
	}
	// WhatsappConnection is read by HTTP handlers and written by whatsmeow
	// event goroutines. Mutable fields are guarded by mu; use the accessors.
	WhatsappConnection struct {
		*ParentData
		Token string

		mu               sync.RWMutex
		client           *whatsmeow.Client
		number           string
		connectionStatus int
		qrCodeString     string
		syncFinished     bool
	}

	// IWhatsappConnectionMap is a token -> connection registry safe for
	// concurrent use.
	IWhatsappConnectionMap struct {
		mu          sync.RWMutex
		connections map[string]*WhatsappConnection
	}
)

var (
	OutPutFilePath = ""
	ConnectionMap  = &IWhatsappConnectionMap{connections: make(map[string]*WhatsappConnection)}
)

func (cm *IWhatsappConnectionMap) Get(token string) (*WhatsappConnection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	connection, ok := cm.connections[token]
	return connection, ok
}

func (cm *IWhatsappConnectionMap) Set(token string, connection *WhatsappConnection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.connections[token] = connection
}

func (cm *IWhatsappConnectionMap) Delete(token string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.connections, token)
}

func (connection *WhatsappConnection) getClient() *whatsmeow.Client {
	connection.mu.RLock()
	defer connection.mu.RUnlock()
	return connection.client
}

// Status returns 0 (waiting for QR scan), 1 (connected) or -1 (logged out).
func (connection *WhatsappConnection) Status() int {
	connection.mu.RLock()
	defer connection.mu.RUnlock()
	return connection.connectionStatus
}

func (connection *WhatsappConnection) setStatus(status int) {
	connection.mu.Lock()
	defer connection.mu.Unlock()
	connection.connectionStatus = status
}

// QRCode returns the latest QR code string, empty if none was issued.
func (connection *WhatsappConnection) QRCode() string {
	connection.mu.RLock()
	defer connection.mu.RUnlock()
	return connection.qrCodeString
}

func (connection *WhatsappConnection) Number() string {
	connection.mu.RLock()
	defer connection.mu.RUnlock()
	return connection.number
}

func (connection *WhatsappConnection) SyncFinished() bool {
	connection.mu.RLock()
	defer connection.mu.RUnlock()
	return connection.syncFinished
}

func (connection *WhatsappConnection) setSyncFinished(finished bool) {
	connection.mu.Lock()
	defer connection.mu.Unlock()
	connection.syncFinished = finished
}

func (connection *WhatsappConnection) ReturnStatusError() error {
	switch connection.Status() {
	case 0:
		return &interfaces.RequestError{
			StatusCode: http.StatusNotFound,
			Code:       interfaces.ERROR_CONNECTION_NOT_INITIALIZED,
			Message:    "Connection Not Initialized QR SCANNED",
			Name:       "ERROR_CONNECTION_NOT_INITIALIZED",
			Extra:      []string{connection.QRCode()},
		}
	case -1:
		return &interfaces.RequestError{
			StatusCode: http.StatusNotFound,
			Code:       interfaces.ERROR_CONNECTION_LOGGED_OUT,
			Message:    "Connection Logged Out",
			Name:       "ERROR_CONNECTION_LOGGED_OUT",
		}
	}
	return nil
}

func (connection *WhatsappConnection) ConnectAndGetQRCode() {
	ConnectionMap.Set(connection.Token, connection)
	client := connection.getClient()
	if client.Store.ID == nil {
		if whatsapp_config.Env.OPEN_BROWSER_FOR_SCAN {
			go func(token string) {
				log.Printf("Opening Browser for Token %s", token)
				utility_functions.OpenBrowser(fmt.Sprintf("http://127.0.0.1:%s/scan/%s", env.GetServerPort(env.PORT_KEY), token))
			}(connection.Token)
		}
		qrChan, _ := client.GetQRChannel(context.Background())
		err := client.Connect()
		if err != nil {
			println(err.Error())
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Printf("QR code for %s\n", connection.Token)
				connection.mu.Lock()
				connection.qrCodeString = evt.Code
				connection.mu.Unlock()
				if !whatsapp_config.Env.OPEN_BROWSER_FOR_SCAN {
					qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				}
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		println("Connected")
		err := client.Connect()
		if err != nil {
			println(err.Error())
		}
	}
}

func (connection *WhatsappConnection) closeTheConnection() {
	ctx := context.Background()
	oldClient := connection.getClient()
	if err := oldClient.Logout(ctx); err != nil {
		oldClient.Disconnect()
		oldClient.Store.Delete(ctx)
	}
	println(connection.Number(), " Logged Out")
	ConnectionMap.Delete(connection.Token)
	whatsapp_config.WhatsappNumberConfigMap.DeleteJID(connection.Token)
	whatsapp_config.WhatsappNumberConfigMap.Save()

	oldClient.Store.DeleteAllSessions(ctx)

	connection.mu.Lock()
	connection.connectionStatus = -1
	connection.qrCodeString = ""
	oldDeviceStore := connection.ParentData.DeviceStore
	connection.mu.Unlock()

	connection.ParentData.SqlContainer.DeleteDevice(ctx, oldDeviceStore)
	deviceStore := connection.ParentData.SqlContainer.NewDevice()
	client := whatsmeow.NewClient(deviceStore, waLog.Stdout("Client", "ERROR", true))
	client.EnableAutoReconnect = true
	// The new client needs the handler too, otherwise Connected never fires
	// after the user scans the new QR and the connection stays logged out.
	client.AddEventHandler(connection.eventHandler)

	connection.mu.Lock()
	connection.client = client
	connection.ParentData.DeviceStore = deviceStore
	connection.mu.Unlock()

	go connection.ConnectAndGetQRCode()
}

func (connection *WhatsappConnection) eventHandler(evt interface{}) {
	ctx := context.Background()
	switch v := evt.(type) {
	case *events.LoggedOut:
		connection.closeTheConnection()
	case *events.Connected:
		client := connection.getClient()
		client.Store.Save(ctx)
		number := client.Store.ID.User
		jid := client.Store.ID.String()
		connection.mu.Lock()
		connection.number = number
		connection.connectionStatus = 1
		connection.mu.Unlock()
		go func() {
			whatsapp_config.WhatsappNumberConfigMap.SetJID(connection.Token, jid)
			whatsapp_config.WhatsappNumberConfigMap.Save()
		}()
		println(number, " Logged In")
	case *events.OfflineSyncPreview:
		connection.setSyncFinished(false)
	case *events.OfflineSyncCompleted:
		connection.setSyncFinished(true)
	// case *events.Message:
	// evt := evt.(*events.Message)
	// log.Println(evt.Info)
	case *events.Receipt:
		evt := evt.(*events.Receipt)
		log.Println(evt.Chat.User)
		b, _ := json.Marshal(evt)
		log.Println(string(b))
	default:
		fmt.Printf("Event Occurred %s\n", reflect.TypeOf(v))
	}
}

func (connection *WhatsappConnection) SendTextMessage(ctx context.Context, to []string, msg string) *map[string]bool {
	client := connection.getClient()
	response := make(map[string]bool)
	for _, number := range to {
		IsOnWhatsappCheck, err := client.IsOnWhatsApp(ctx, []string{"+" + number})
		if err != nil || len(IsOnWhatsappCheck) == 0 {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Something Went Wrong or Number Not On Whatsapp %#v\n", number, err))
			response[number] = false
			continue
		}
		NumberOnWhatsapp := IsOnWhatsappCheck[0]
		if !NumberOnWhatsapp.IsIn {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Number %s Not On Whatsapp\n", number, number))
			response[number] = false
			continue
		}
		targetJID := NumberOnWhatsapp.JID
		fmt.Printf("sending Text To %s\n", number)
		response[number] = false
		if len(msg) > 0 {
			_, err := client.SendMessage(ctx, targetJID, &waE2E.Message{
				Conversation: proto.String(msg),
			})
			if err == nil {
				response[number] = true
			}
		}
	}
	return &response
}
func (connection *WhatsappConnection) SendMediaFileFromURLs(ctx context.Context, to []string, mediaURL []string, msg string) {
	for _, url := range mediaURL {
		connection.SendMediaFileFromURL(ctx, to, url, "", msg)
		time.Sleep(time.Second * 2)
	}
}

// SendMediaFileFromURL fetches a media file from a HTTP/HTTPS URL and sends it.
func (connection *WhatsappConnection) SendMediaFileFromURL(ctx context.Context, to []string, mediaURL string, fileName string, msg string) *map[string]bool {
	bytesData, extractedFileName, err := whatsapp_functions.FetchFileFromURL(ctx, mediaURL)
	if err != nil {
		AppendToOutPutFile(fmt.Sprintf("false,Error While Downloading File From Web URL %#v\n", err))
		return nil
	}
	if fileName == "" {
		fileName = extractedFileName
	}
	if fileName == "" {
		fileName = "file"
	}
	return connection.sendMediaFile(ctx, to, bytesData, fileName, msg)
}

func (connection *WhatsappConnection) SendMediaFileBase64(ctx context.Context, to []string, base64Data string, fileName string, msg string) *map[string]bool {
	bytesData, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		AppendToOutPutFile(fmt.Sprintf("false,Error While Reading File %#v\n", err))
		return nil
	}
	return connection.sendMediaFile(ctx, to, bytesData, fileName, msg)
}

func (connection *WhatsappConnection) SendMediaFileWithPath(ctx context.Context, to []string, filePath string, fileName string, msg string) *map[string]bool {
	pdfBytes, err := os.ReadFile(filePath)
	if err != nil {
		AppendToOutPutFile(fmt.Sprintf("false,Error While Reading File %#v\n", err))
		return nil
	}
	return connection.sendMediaFile(ctx, to, pdfBytes, fileName, msg)
}

func (connection *WhatsappConnection) sendMediaFile(ctx context.Context, to []string, fileByte []byte, fileName string, msg string) *map[string]bool {
	client := connection.getClient()
	response := make(map[string]bool)
	var docProto *waE2E.Message

	for _, number := range to {
		IsOnWhatsappCheck, err := client.IsOnWhatsApp(ctx, []string{"+" + number})
		if err != nil || len(IsOnWhatsappCheck) == 0 {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Something Went Wrong %#v\n", number, err))
			response[number] = false
			continue
		}
		NumberOnWhatsapp := IsOnWhatsappCheck[0]
		if !NumberOnWhatsapp.IsIn {
			AppendToOutPutFile(fmt.Sprintf("%s,false,Number %s Not On Whatsapp\n", number, number))
			response[number] = false
			continue
		}
		targetJID := NumberOnWhatsapp.JID
		fmt.Printf("sending File To %s\n", number)

		if docProto == nil {
			extensionName := utility_functions.GetMime(fileName)
			if strings.Contains(extensionName, "image") {
				resp, err := client.Upload(ctx, fileByte, whatsmeow.MediaImage)
				if err != nil {
					AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Uploading %#v\n", number, err))
					continue
				}
				jpegThumbnail, err := utility_functions.ImageThumbnail(fileByte)
				if err != nil {
					AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Generating Thumbnail %#v\n", number, err))
				}
				docProto = &waE2E.Message{
					ImageMessage: &waE2E.ImageMessage{
						Caption:       proto.String(msg),
						URL:           &resp.URL,
						Mimetype:      proto.String(extensionName),
						DirectPath:    &resp.DirectPath,
						JPEGThumbnail: jpegThumbnail,
						MediaKey:      resp.MediaKey,
						FileEncSHA256: resp.FileEncSHA256,
						FileSHA256:    resp.FileSHA256,
						FileLength:    &resp.FileLength,
					},
				}
			} else if strings.Contains(extensionName, "audio") {
				resp, err := client.Upload(ctx, fileByte, whatsmeow.MediaAudio)
				if err != nil {
					AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Uploading %#v\n", number, err))
					continue
				}
				docProto = &waE2E.Message{
					AudioMessage: &waE2E.AudioMessage{
						URL:           &resp.URL,
						Mimetype:      proto.String(extensionName),
						DirectPath:    &resp.DirectPath,
						MediaKey:      resp.MediaKey,
						FileEncSHA256: resp.FileEncSHA256,
						FileSHA256:    resp.FileSHA256,
						FileLength:    &resp.FileLength,
					},
				}
			} else if strings.Contains(extensionName, "video") {
				thumbBytes, _ := utility_functions.GenerateVideoThumbnail(fileByte, fileName)
				resp, err := client.Upload(ctx, fileByte, whatsmeow.MediaVideo)
				if err != nil {
					AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Uploading %#v\n", number, err))
					continue
				}
				docProto = &waE2E.Message{
					VideoMessage: &waE2E.VideoMessage{
						Caption:       proto.String(msg),
						URL:           &resp.URL,
						Mimetype:      proto.String(extensionName),
						DirectPath:    &resp.DirectPath,
						MediaKey:      resp.MediaKey,
						FileEncSHA256: resp.FileEncSHA256,
						FileSHA256:    resp.FileSHA256,
						FileLength:    &resp.FileLength,
					},
				}
				if len(thumbBytes) > 0 {
					docProto.VideoMessage.JPEGThumbnail = thumbBytes
				}
			} else {
				resp, err := client.Upload(ctx, fileByte, whatsmeow.MediaDocument)
				if err != nil {
					AppendToOutPutFile(fmt.Sprintf("%s,false,Error While Uploading %#v\n", number, err))
					continue
				}
				docProto = &waE2E.Message{
					DocumentMessage: &waE2E.DocumentMessage{
						Caption:       proto.String(msg),
						URL:           &resp.URL,
						Mimetype:      proto.String(extensionName),
						FileName:      &fileName,
						DirectPath:    &resp.DirectPath,
						MediaKey:      resp.MediaKey,
						FileEncSHA256: resp.FileEncSHA256,
						FileSHA256:    resp.FileSHA256,
						FileLength:    &resp.FileLength,
					},
				}
				if strings.Contains(extensionName, "pdf") {
					thumb, err := whatsapp_utility.ExtractFirstPage(fileByte)
					if err == nil && len(thumb) > 0 {
						docProto.DocumentMessage.JPEGThumbnail = thumb
					}
				}
			}
		}

		response[number] = false
		if docProto != nil {
			_, err := client.SendMessage(ctx, targetJID, docProto)
			if err == nil {
				response[number] = true
			}
		}
	}
	return &response
}

var outPutFileMu sync.Mutex

// AppendToOutPutFile appends one line to the CSV send log. A failure here must
// not take the server down: callers often run in goroutines, where a panic
// kills the whole process. Errors are logged instead.
func AppendToOutPutFile(text string) {
	outPutFileMu.Lock()
	defer outPutFileMu.Unlock()

	f, err := os.OpenFile(OutPutFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		log.Printf("[AppendToOutPutFile] open %q: %v (line: %q)", OutPutFilePath, err, text)
		return
	}
	defer f.Close()

	if _, err = f.WriteString(text); err != nil {
		log.Printf("[AppendToOutPutFile] write %q: %v (line: %q)", OutPutFilePath, err, text)
	}
}
