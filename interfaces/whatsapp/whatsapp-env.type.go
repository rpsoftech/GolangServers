package whatsapp_interfaces

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/rpsoftech/golang-servers/env"
	"github.com/rpsoftech/golang-servers/validator"
)

type (
	EnvInterface struct {
		ALLOW_LOCAL_NO_AUTH      bool `json:"ALLOW_LOCAL_NO_AUTH" validate:"boolean"`
		AUTO_CONNECT_TO_WHATSAPP bool `json:"AUTO_CONNECT_TO_WHATSAPP" validate:"boolean"`
		OPEN_BROWSER_FOR_SCAN    bool `json:"OPEN_BROWSER_FOR_SCAN" validate:"boolean"`
	}
	// IServerConfig is shared between HTTP handlers and whatsmeow event
	// goroutines. Access Tokens and JID only through the methods below.
	IServerConfig struct {
		mu             sync.RWMutex
		Tokens         map[string]string `json:"tokens" validate:"required"`
		JID            map[string]string `json:"JID"`
		configFilePath string            `json:"-"`
	}
)

func ReadConfigFileAndReturniserverConfig(configFilePath string) *IServerConfig {
	config := new(IServerConfig)

	dat, err := os.ReadFile(configFilePath)
	env.Check(err)
	err = json.Unmarshal(dat, config)
	// env.Check(err)
	if errs := validator.Validator.Validate(config); len(errs) > 0 {
		panic(fmt.Errorf("CONFIG_ERROR %#v", errs))
	}
	if config.JID == nil {
		config.JID = make(map[string]string)
	}
	config.configFilePath = configFilePath
	return config
}

func (sc *IServerConfig) GetConfigPath() string {
	return sc.configFilePath
}
func (sc *IServerConfig) SetConfigPath(path string) *IServerConfig {
	sc.configFilePath = path
	return sc
}
// GetToken returns the value stored for token and whether the token exists.
func (sc *IServerConfig) GetToken(token string) (string, bool) {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	value, ok := sc.Tokens[token]
	return value, ok
}

// AddToken stores token with value. It returns false if the token already exists.
func (sc *IServerConfig) AddToken(token string, value string) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if _, ok := sc.Tokens[token]; ok {
		return false
	}
	sc.Tokens[token] = value
	return true
}

// TokensSnapshot returns a copy of the token map that is safe to range over.
func (sc *IServerConfig) TokensSnapshot() map[string]string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	tokens := make(map[string]string, len(sc.Tokens))
	for k, v := range sc.Tokens {
		tokens[k] = v
	}
	return tokens
}

func (sc *IServerConfig) GetJID(token string) string {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.JID[token]
}

func (sc *IServerConfig) SetJID(token string, jid string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.JID[token] = jid
}

func (sc *IServerConfig) DeleteJID(token string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.JID, token)
}

// Save holds the write lock for the whole marshal and write so concurrent
// saves cannot interleave on disk.
func (sc *IServerConfig) Save() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	byteJson, err := json.MarshalIndent(sc, "", "    ")
	if err != nil {
		return
	}
	err = os.WriteFile(sc.configFilePath, byteJson, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
