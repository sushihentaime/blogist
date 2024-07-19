#build stage
FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /go/src/app
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . . 
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app ./app

#final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/app/app .
COPY --from=builder /go/src/app/.env .
RUN chmod +x ./app
CMD ["./app"]

LABEL Name=blogist Version=0.0.1
EXPOSE 8000


