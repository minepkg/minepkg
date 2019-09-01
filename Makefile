.PHONY: build run

build:
	go build -ldflags="-s -w" -o ./out/minepkg

