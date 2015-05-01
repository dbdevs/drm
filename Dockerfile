FROM golang:1.4-cross

COPY entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
