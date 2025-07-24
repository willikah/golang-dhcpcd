# Dockerfile to run golang-dhcpcd client for integration testing
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY .. .
RUN go build -o golang-dhcpcd main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/golang-dhcpcd ./golang-dhcpcd
ENTRYPOINT ["/app/golang-dhcpcd"]
