FROM golang:1.4-cross

COPY entrypoint.sh /

ENV GOOS=darwin GOARCH=amd64

ENTRYPOINT ["/entrypoint.sh"]
