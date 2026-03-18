FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git
WORKDIR /app

RUN go install github.com/a-h/templ/cmd/templ@latest

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN templ generate && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dungeon-revealer .

FROM alpine:3
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/dungeon-revealer .

VOLUME /data
EXPOSE 3000
ENV DR_DATA_DIR=/data

ENTRYPOINT ["/app/dungeon-revealer"]
