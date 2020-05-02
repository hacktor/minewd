all: build-all

build-all: build-linux build-windows

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o minewd minewd.go binary.go json.go

build-windows:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o minewd.exe minewd.go binary.go json.go

