#!/usr/bin/env bash
GOARCH=arm GOARM=6 GOOS=linux go build
scp enviro pi@192.168.1.72:.
