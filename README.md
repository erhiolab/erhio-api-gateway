## 构建

### Windows

```cmd
set CGO_ENABLED=0 && set GOOS=windows && set GOARCH=amd64 && go build -o elake-api-gateway-windows.exe ./cmd/server
```

```PowerShell
$env:CGO_ENABLED=0; $env:GOOS="windows"; $env:GOARCH="amd64"; go build -o "elake-api-gateway-windows.exe" "./cmd/server"
`````

```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build "elake-api-gateway-windows.exe" "./cmd/server"
```

## Linux

```cmd
set CGO_ENABLED=0 && set GOOS=linux && set GOARCH=amd64 && go build -o elake-api-gateway-linux ./cmd/server
```

```PowerShell
$env:CGO_ENABLED=0; $env:GOOS="linux"; $env:GOARCH="amd64"; go build -o "elake-api-gateway-linux" "./cmd/server"
`````

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build "elake-api-gateway-linux" "./cmd/server"
```
