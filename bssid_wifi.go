package main

import (
	"fmt"
	"net"
	"bytes"
	"time"
)

func doEveryMin(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func getMacAddr() string {
	var addr string
	interfaces, err := net.Interfaces()
	if err == nil {
		for _, i := range interfaces {
			if i.Flags&net.FlagUp != 0 && bytes.Compare(i.HardwareAddr, nil) != 0 {
				addr = i.HardwareAddr.String()
				break
			} 
		}			
	}
	return addr
}

func printMacAddr(t time.Time) {
	fmt.Println("bssid:", getMacAddr())
}


func main() {
	fmt.Println("bssid:", getMacAddr())
	doEveryMin(time.Minute, printMacAddr)
}