package whatsapp_core

import (
	"log"
	"sync"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

// Reserve atomically claims token for a new connection. It returns false
// when the token already has a connection or a pending reservation.
//
// A reservation is stored as a nil entry, so Get returns (nil, true) until
// ConnectAndGetQRCode stores the real connection. Callers of Get must check
// for a nil connection.
func (cm *IWhatsappConnectionMap) Reserve(token string) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if _, exists := cm.connections[token]; exists {
		return false
	}
	cm.connections[token] = nil
	return true
}

// release removes the entry for token only while it still holds expected.
// Pass nil to drop a reservation that never got a connection.
func (cm *IWhatsappConnectionMap) release(token string, expected *WhatsappConnection) bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	current, exists := cm.connections[token]
	if !exists || current != expected {
		return false
	}
	delete(cm.connections, token)
	return true
}

// snapshot returns the non-nil connections currently in the map.
func (cm *IWhatsappConnectionMap) snapshot() []*WhatsappConnection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	connections := make([]*WhatsappConnection, 0, len(cm.connections))
	for _, connection := range cm.connections {
		if connection != nil {
			connections = append(connections, connection)
		}
	}
	return connections
}

// ConnectEach reserves every token and runs connect for each reserved token
// in its own goroutine, so a token that waits for a QR scan does not block
// the others. Tokens that are already reserved or connected are skipped.
// The returned WaitGroup completes when every started connect returns.
func (cm *IWhatsappConnectionMap) ConnectEach(tokens []string, jidOf func(token string) string, connect func(jidString, token string)) *sync.WaitGroup {
	var wg sync.WaitGroup
	for _, token := range tokens {
		if !cm.Reserve(token) {
			log.Printf("[whatsapp] skip auto connect for %s: already reserved or connected", token)
			continue
		}
		jidString := jidOf(token)
		wg.Add(1)
		go func() {
			defer wg.Done()
			connect(jidString, token)
		}()
	}
	return &wg
}

// ConnectReserved connects a token that the caller reserved with
// ConnectionMap.Reserve. It blocks while a new device waits for a QR scan,
// so run it in a goroutine. If the connect fails or the QR scan times out,
// the dead connection is removed so a later start call can try again.
func ConnectReserved(jidString string, token string, sqlContainer *sqlstore.Container) {
	connection := newConnection(jidString, token, sqlContainer)
	client := connection.getClient()
	paired := client.Store.ID != nil

	connection.ConnectAndGetQRCode()

	// A paired device should now hold an open socket. A new device returns
	// once the QR channel closes; its Store.ID is set only if pairing worked.
	failed := !client.IsConnected()
	if !paired {
		failed = client.Store.ID == nil
	}
	if !failed {
		return
	}
	log.Printf("[whatsapp] connect for %s did not finish, releasing it", token)
	client.Disconnect()
	ConnectionMap.release(token, connection)
}

// DisconnectAll closes the websocket of every stored connection. Call it on
// shutdown, after the HTTP server stops taking requests.
func DisconnectAll() {
	for _, connection := range ConnectionMap.snapshot() {
		if client := connection.getClient(); client != nil {
			client.Disconnect()
		}
	}
}
