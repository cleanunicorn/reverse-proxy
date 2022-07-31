VERSION := "$(shell git describe --abbrev=0 --tags 2> /dev/null || echo 'v0.0.0')+$(shell git rev-parse --short HEAD 2> /dev/null || echo 'unknown')"

build:
	go build -ldflags "-X main.buildVersion=$(VERSION)"

run:
	go run main.go

keys-clear:
	rm -r cert

keys:
	mkdir cert
	openssl ecparam -genkey -name secp384r1 -out cert/server.key
	openssl req -new -x509 -sha256 -key cert/server.key -out cert/server.pem -days 3650
