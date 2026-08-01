package soham_whatsapp_keys

import "os/exec"

var ServerConfigFilePath = ""
var ServerCmd *exec.Cmd
var HomeDir string
var ConfigDir string

const QRCODEURL = "http://localhost:4000/v1/qr_code"
const LoginStatusURL = "http://localhost:4000/v1/status"
