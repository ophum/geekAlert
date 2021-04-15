FROM golang:1.16
WORKDIR /go/src/github.com/ophum/geekAlert
COPY ./ .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o bin/geekAlert main.go

FROM alpine:latest
WORKDIR /app/
RUN apk add sqlite
COPY --from=0 /go/src/github.com/ophum/geekAlert/bin/geekAlert .
ENTRYPOINT ["./geekAlert"]