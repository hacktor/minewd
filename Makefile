all: build-all-bin

build-all-bin: build-linux-bin build-windows-bin

build-linux-bin:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o minewd-bin minewd.go binpacket.go

build-windows-bin:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o minewd-bin.exe minewd.go binpacket.go
