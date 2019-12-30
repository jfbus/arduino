# Omega

Omega is a Go library for sending data in mqtt with Google Pubsub using Omega with sensors.

## Installation

```bash
GOARCH=mipsle go build -o omega *.go
scp omega root@192.168.3.1:.
```

## Usage

```bash
./omegainit.sh
```
or
```bash
./omega -ca_certs PemFiles/roots.pem -device Raspi -private_key PemFiles/private_key.pem -project cocoon-iot -region europe-west1 -registry cocoon-iot-registry -format line
```