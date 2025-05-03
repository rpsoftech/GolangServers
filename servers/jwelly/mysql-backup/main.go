package main

type MysqlConnectionConfig struct {
	Host         string `json:"host" validate:"required"`
	Port         int    `json:"port" validate:"required,port"`
	User         string `json:"user" validate:"required"`
	Pass         string `json:"pass" validate:"required"`
	DatabaseName string `json:"DatabaseName" validate:"required"`
}

type FileServerConfig struct {
	Url    string            `json:"url" validate:"required"`
	Token  string            `json:"token" validate:"required"`
	Params map[string]string `json:"params" validate:"required"`
}

type Config struct {
}

func main() {
	// TODO: implement me
}
