set CGO_ENABLED=0

set GOOS=windows
set GOARCH=amd64
go build -o elake-api-gateway-windows.exe ./cmd/server

set GOOS=linux
set GOARCH=amd64
go build -o elake-api-gateway-linux ./cmd/server