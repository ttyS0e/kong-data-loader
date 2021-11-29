FROM golang:alpine as builder

RUN mkdir -p /data/build
WORKDIR /data/build

COPY go.mod go.mod
COPY go.sum go.sum
COPY main.go main.go

RUN go build -o kongdataloader

#

FROM alpine:3

ENV KONG_PG_HOST=127.0.0.1
ENV KONG_PG_DATABASE=kong
ENV KONG_PG_USER=kong
ENV KONG_PG_PASSWORD=kong

RUN apk add ca-certificates
RUN adduser --disabled-password kong
USER kong
WORKDIR /usr/local/bin
COPY --from=builder /data/build/kongdataloader .

ENTRYPOINT [ "kongdataloader" ]
