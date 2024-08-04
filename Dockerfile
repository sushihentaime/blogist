#build stage
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags '-s -w' -o app ./app

#final stage
FROM alpine:latest
RUN apk update && apk add --no-cache ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/app/app .
COPY --from=builder /go/src/app/.env .

RUN chmod +x ./app
CMD ["./app"]

LABEL Name=blogist Version=0.0.1
