FROM golang:alpine as builder

WORKDIR /app

COPY go.mod go.mod
COPY go.sum go.sum

RUN ["go", "mod", "download"]

COPY main.go main.go
COPY cmd cmd
COPY internal internal

RUN ["apk", "add", "gcc", "musl-dev"]
RUN ["go", "build", "-o", "gitstream"]

FROM alpine

RUN ["apk", "add", "ca-certificates"]

COPY --from=builder /app/gitstream /usr/local/bin
