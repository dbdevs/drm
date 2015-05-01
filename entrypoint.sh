#!/bin/bash

go get github.com/dbdevs/drm
go get github.com/codegangsta/cli
go get github.com/barkerd427/dockerclient

go build -v -x /go/src/github.com/dbdevs/drm/drm.go

exec "$@"