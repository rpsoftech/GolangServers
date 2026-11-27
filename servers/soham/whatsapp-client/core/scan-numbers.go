package whatsapp_client_core

import (
	"sync"

	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
)

// scanNumbersMu guards whatsapp_config.WhatsappNumberToIDMap. The map is
// written at startup and by POST /v1/start while /scan/:id reads it, so every
// access in this client goes through SetScanNumber and ScanNumber.
var scanNumbersMu sync.RWMutex

// SetScanNumber records the WhatsApp number shown on /scan/<token>.
func SetScanNumber(token string, number string) {
	scanNumbersMu.Lock()
	defer scanNumbersMu.Unlock()
	if whatsapp_config.WhatsappNumberToIDMap == nil {
		whatsapp_config.WhatsappNumberToIDMap = map[string]string{}
	}
	whatsapp_config.WhatsappNumberToIDMap[token] = number
}

// ScanNumber returns the number recorded for token by SetScanNumber.
func ScanNumber(token string) (string, bool) {
	scanNumbersMu.RLock()
	defer scanNumbersMu.RUnlock()
	number, ok := whatsapp_config.WhatsappNumberToIDMap[token]
	return number, ok
}
