VERSION=v0.1.0
MAIN_PKG=nftgen
TARGET=_build/$(MAIN_PKG)
PKG=$(shell go list)
LDFLAGS=-ldflags "-X '$(PKG)/cmd/$(MAIN_PKG).version=$(VERSION)'"


clean:
	rm -f $(TARGET)

build:
	go build -o $(TARGET) $(LDFLAGS) main.go

install: build
	cp $(TARGET) ${GOPATH}/bin/