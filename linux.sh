GOOS=linux GOARCH=amd64 go build -o "http-dump.o" -ldflags="-s -w" -v ./servers/http-dump/dump-server/main.go
GOOS=linux GOARCH=amd64 go build -o "message-dump.o" -ldflags="-s -w" -v ./servers/http-dump/message-dump/main.go
GOOS=linux GOARCH=amd64 go build -o "telegram-server.o" -ldflags="-s -w" -v ./servers/telegram-server/main.go