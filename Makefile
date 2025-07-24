BINARY=golang-dhcpcd

all: generate build

.PHONY: generate

generate:
	go generate ./...

.PHONY: build

build:
	go build -o $(BINARY) main.go

.PHONY: run

run:
	./$(BINARY)

.PHONY: clean

clean:
	rm -f $(BINARY) version.txt
