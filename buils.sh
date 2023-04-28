#!/bin/bash

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .
docker build --platform=linux/amd64  -t 1135189009/es-index:1.1 .
docker push 1135189009/es-index:1.1

# CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build .
# docker build --platform=linux/arm64  -t 1135189009/es_drop:latest .
# docker push 1135189009/es_drop:latest


