.PHONY: build run test

build:
	go build -ldflags="-s -w" -o ./out/minepkg

test:
	go test -v ./...

build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./out/minepkg-linux

run-docker:
	docker run --rm -it \
		-v ${PWD}/out/minepkg-linux:/usr/bin/minepkg \
		-v ${PWD}/.tmp/docker-configs:/root/.minepkg \
		golang:latest bash

MOD=github.com/fiws/minepkg

godoc:
	docker run \
		--rm \
		-e "GOPATH=/tmp/go" \
		-p 127.0.0.1:6060:6060 \
		-v ${PWD}:/tmp/go/src/${MOD} \
		golang \
		bash -c "go get golang.org/x/tools/cmd/godoc && echo http://localhost:6060/pkg/${MOD} && /tmp/go/bin/godoc -http=:6060"

docker: build-linux run-docker

