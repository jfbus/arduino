#!/usr/bin/env bash
GOARCH=mipsle go build -o omega *.go
scp omega root@192.168.3.1:.
