FROM golang:1.22-alpine as golang
LABEL org.opencontainers.image.source https://github.com/rssnyder/discord-bot-cryptoprices

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /discord-bot

ENTRYPOINT /discord-bot
