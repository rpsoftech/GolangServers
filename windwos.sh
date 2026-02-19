GOOS=windows GOARCH=amd64 go build -v -ldflags="-s -w" ./...
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/http-dump.exe" -v -ldflags="-s -w" ./servers/http-dump/dump-server/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/message-dump.exe" -v -ldflags="-s -w" ./servers/http-dump/message-dump/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/telegram-server.exe" -v -ldflags="-s -w" ./servers/telegram-server/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/boozv3.exe" -v -ldflags="-s -w" ./servers/boozv3/main-server/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/bullion.exe" -v -ldflags="-s -w" ./servers/bullion/main-server/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/function-management.exe" -v -ldflags="-s -w" ./servers/function-management/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/link-shortner.exe" -v -ldflags="-s -w" ./servers/link-shortner/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/whatsapp-server.exe" -v -ldflags="-s -w" ./servers/whatsapp-server/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/mysql-to-surreal.exe" -v -ldflags="-s -w" ./servers/jwelly/mysql-to-surreal/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/main-server.exe" -v -ldflags="-s -w" ./servers/jwelly/main-server/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/mysql-backup.exe" -v -ldflags="-s -w" ./servers/jwelly/mysql-backup/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/mysql-to-mysql.exe" -v -ldflags="-s -w" ./servers/jwelly/mysql-to-mysql/main.go
GOOS=windows GOARCH=amd64 go build -o "dist/windows-amd64/mysql-backup-cmd.exe" -v -ldflags="-s -w" ./servers/jwelly/mysql-backup-cmd/main.go