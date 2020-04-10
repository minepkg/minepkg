.PHONY: build run

build:
	go build -ldflags="-s -w" -o ./out/minepkg

run-docker:
	docker run --rm -it \
		-v ${PWD}/out/minepkg:/usr/bin/minepkg \
		-v ${PWD}/.tmp/docker-configs:/root/.minepkg \
		 golang:latest bash

docker: build run-docker

