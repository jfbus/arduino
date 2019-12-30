# Raspi

Raspi is a Go library for sending data in mqtt with Google Pubsub using Raspberry with sensors.

## Installation

```bash
./build.sh
```

## Usage

```bash
./init
```
or
```bash
./raspi -ca_certs /home/pi/GoogleIoTClient/PemFiles/roots.pem -device Raspi -private_key /home/pi/GoogleIoTClient/PemFiles/private_key.pem -project cocoon-iot -region europe-west1 -registry cocoon-iot-registry -format line
```