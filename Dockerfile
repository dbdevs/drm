FROM golang:1.4-cross

COPY entrypoint.sh /
COPY function.sh /

ENTRYPOINT ["/entrypoint.sh"]
