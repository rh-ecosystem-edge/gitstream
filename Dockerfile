FROM golang:alpine3.16 as builder

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum

RUN ["go", "mod", "download"]

COPY main.go main.go
COPY cmd cmd
COPY internal internal

RUN ["go", "build", "-o", "gitstream"]

FROM alpine:3.16

RUN ["apk", "add", "ca-certificates"]

COPY --from=builder /app/gitstream /usr/local/bin