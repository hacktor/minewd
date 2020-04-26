all: build-all-bin build-all-json

build-all-bin: build-linux-bin build-windows-bin

build-all-json: build-linux-json build-windows-json

build-linux-bin:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o minewd-bin minewd.go binpacket.go

build-windows-bin:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o minewd-bin.exe minewd.go binpacket.go

build-linux-json:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o minewd-json minewd.go jsonpacket.go

build-windows-json:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o minewd-json.exe minewd.go jsonpacket.go

