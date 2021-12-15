FROM golang:1.17-bullseye
COPY . .
go mod tidy
go build
