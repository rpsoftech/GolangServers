package updater

// Component names. Each is part of the KV key <ENV>_<component>_<os>_<arch>,
// so renaming one moves every install of it to a new, empty channel.
const (
	WhatsappProjectName            = "whatsapp-server"
	MysqlBackupCmdProjectName      = "mysql_backup"
	SohamWhatsappClientProjectName = "soham-whatsapp-client"
)
