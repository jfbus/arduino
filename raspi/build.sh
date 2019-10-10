#!/usr/bin/env bash
#GOARCH=arm GOARM=6 GOOS=linux go build
#scp raspi pi@192.168.1.73:.

go build -o raspi *.go
