#!/bin/bash

go get github.com/dbdevs/drm
go get github.com/codegangsta/cli
go get github.com/barkerd427/dockerclient

if [ "$1" == "mac" ]; then
  export GOOS=darwin
  export GOARCH=amd64
elif [ "$1" == "linux" ]; then
  export GOOS=linux
  export GOARCH=amd64
fi

go build -v -x /go/src/github.com/dbdevs/drm/drm.go

if [ "$1" != "mac" ] || [ "$1" != "linux" ]; then
  exec "$@"
fi