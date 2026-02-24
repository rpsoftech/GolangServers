GOOS=linux GOARCH=amd64 go build ./...
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/http-dump.o" -ldflags="-s -w" -v ./servers/http-dump/dump-server/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/message-dump.o" -ldflags="-s -w" -v ./servers/http-dump/message-dump/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/telegram-server.o" -ldflags="-s -w" -v ./servers/telegram-server/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/boozv3.o" -ldflags="-s -w" -v ./servers/boozv3/main-server/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/bullion.o" -ldflags="-s -w" -v ./servers/bullion/main-server/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/function-management.o" -ldflags="-s -w" -v ./servers/function-management/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/link-shortner.o" -ldflags="-s -w" -v ./servers/link-shortner/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/whatsapp-server.o" -ldflags="-s -w" -v ./servers/whatsapp-server/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/mysql-to-surreal.o" -ldflags "-s -w" -v ./servers/jwelly/mysql-to-surreal/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/main-server.o" -ldflags "-s -w" -v ./servers/jwelly/main-server/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/mysql-backup.o" -ldflags "-s -w" -v ./servers/jwelly/mysql-backup/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/mysql-to-mysql.o" -ldflags "-s -w" -v ./servers/jwelly/mysql-to-mysql/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/linux-amd64/mysql-backup-cmd.o" -v -ldflags="-s -w" ./servers/jwelly/mysql-backup-cmd/main.go
GOOS=linux GOARCH=amd64 go build -o "dist/windows-amd64/soham/whatsapp-server.o" -v -ldflags="-s -w" ./servers/soham/whatsapp-server/main.go