#!/usr/bin/env bash

rm -fv Commander
rm -fv Worker

protoc --go_out=plugins=grpc:. internal/src/pbMessages/messages.proto

export GOPATH=$HOME/go:$(pwd)/internal
for CMD in `ls cmd`; do
	echo "Building $CMD"
	go build -v ./cmd/$CMD
	echo "Done!
"
done
