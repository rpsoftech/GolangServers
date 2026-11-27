package soham_whatsapp_client_env

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/rpsoftech/golang-servers/env"
	whatsapp_config "github.com/rpsoftech/golang-servers/functions/whatsapp/config"
)

const SOCKET_URL_KEY = "SOCKET_URL"

// LOCAL_MEDIA_DIR_KEY names the optional directory that SEND_FILE_PATH_MEDIA
// requests may read from. When it is unset, every such request is rejected.
const LOCAL_MEDIA_DIR_KEY = "LOCAL_MEDIA_DIR"

var SocketUrl *url.URL

// LocalMediaDir is the absolute, symlink-resolved LOCAL_MEDIA_DIR, or "" when
// the setting is not configured.
var LocalMediaDir string

func InialiseSohamWhatsappClientEnv() {
	env.LoadEnv(filepath.Join(env.FindAndReturnCurrentDir(), "whatsapp-client.env"))
	whatsapp_config.InitaliseWhatsappEnvAndConfig()
	urlString := env.Env.GetEnv(SOCKET_URL_KEY)
	if urlString == "" {
		panic("SOCKET_URL not defined in env file")
	}
	urlObject, err := url.ParseRequestURI(urlString)
	if err != nil {
		panic("SOCKET_URL is not a valid URL")
	}
	SocketUrl = urlObject

	dir, err := ParseLocalMediaDir(env.Env.GetEnv(LOCAL_MEDIA_DIR_KEY))
	if err != nil {
		panic(err.Error())
	}
	LocalMediaDir = dir
}

// ParseLocalMediaDir validates a LOCAL_MEDIA_DIR value. An empty value returns
// "" with no error (the feature stays disabled). Otherwise the value must name
// an existing directory; the result is absolute and has symlinks resolved.
func ParseLocalMediaDir(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	abs, err := filepath.Abs(filepath.Clean(value))
	if err != nil {
		return "", fmt.Errorf("%s %q is invalid: %w", LOCAL_MEDIA_DIR_KEY, value, err)
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("%s %q is invalid: %w", LOCAL_MEDIA_DIR_KEY, value, err)
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("%s %q is invalid: %w", LOCAL_MEDIA_DIR_KEY, value, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s %q is not a directory", LOCAL_MEDIA_DIR_KEY, value)
	}
	return resolved, nil
}
