#! /bin/sh
cd cli
CGO_ENABLED=0 go build -a -ldflags '-w' -o ../nocino
cd ..
docker build . -t kipters/nocino:latest
