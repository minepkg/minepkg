.PHONY: build run

build:
	go build -ldflags="-s -w" -o ./out/minepkg

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./out/minepkg-linux

run-docker:
	docker run --rm -it \
		-v ${PWD}/out/minepkg-linux:/usr/bin/minepkg \
		-v ${PWD}/.tmp/docker-configs:/root/.minepkg \
		 golang:latest bash

docker: build-linux run-docker

