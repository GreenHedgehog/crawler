.PHONY: install build start stop

build: build.proto build.inject-tags build.crawler

build.proto:
	protoc --proto_path=./ --twirp_out=./rpc --go_out=./rpc  ./rpc/crawler/service.proto

build.inject-tags:
	protoc-go-inject-tag -input=./rpc/crawler/service.pb.go

build.crawler:
	docker build . -f ./build/crawler.Dockerfile -t crawler:latest

start:
	docker-compose -f deployment/docker-compose.yml up -d 

stop:
	docker-compose -f deployment/docker-compose.yml stop

install:
	go get -u -v github.com/golang/protobuf/protoc-gen-go
	go get -u -v github.com/twitchtv/twirp/protoc-gen-twirp
	go get -u -v github.com/favadi/protoc-go-inject-tag