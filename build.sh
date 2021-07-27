#!/usr/bin/env sh

export GOARCH=amd64
export GOOS=linux
export CGO_ENABLED=0

rm -rf .build
mkdir .build
go build -o .build/sync-to-zk cmd/sync-to-zk/*.go

echo 'ouput to .build/sync-to-zk'