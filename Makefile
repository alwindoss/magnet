APP_NAME=magnet

.PHONY: help, build, run, test

help:
	@echo "Welcome to magnet build tool"

build:
	go build -o ./bin/

run: build
	./bin/magnet

test:
	go test -v ./...