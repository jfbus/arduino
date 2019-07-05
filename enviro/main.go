package main

import (
	"context"
	"fmt"
	"log"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"periph.io/x/periph/host/rpi"
)

func main() {
	cfg := struct {
		Reporter ReporterConfig
		BT       BTScannerConfig
	}{}
	envconfig.MustProcess("", &cfg)
	fmt.Printf("Starting with config: %+v\n", cfg)
	if rpi.Present() {
		fmt.Println("running on a raspberry pi")
	}

	d, err := dev.NewDevice("default")
	if err != nil {
		log.Fatalf("can't create BLE device : %s", err)
	}
	ble.SetDefaultDevice(d)

	r := NewReporter(cfg.Reporter)
	sc := NewBTScanner(cfg.BT, r)
	e, err := NewEnviro(r)
	if err != nil {
		log.Fatalf("can't init enviro : %s", err)
	}
	// Scan for specified durantion, or until interrupted by user.
	// ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), *du))
	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error { return sc.Run(ctx) })
	g.Go(func() error { return r.Run(ctx) })
	g.Go(func() error { return ble.Scan(ctx, true, sc.Discover, nil) })
	g.Go(func() error { return e.Run(ctx) })
	chkErr(g.Wait())
}

func advHandler(a ble.Advertisement) {
	if a.Connectable() {
		fmt.Printf("[%s] C %3d:", a.Addr(), a.RSSI())
	} else {
		fmt.Printf("[%s] N %3d:", a.Addr(), a.RSSI())
	}
	comma := ""
	if len(a.LocalName()) > 0 {
		fmt.Printf(" Name: %s", a.LocalName())
		comma = ","
	}
	if len(a.Services()) > 0 {
		fmt.Printf("%s Svcs: %v", comma, a.Services())
		comma = ","
	}
	if len(a.ManufacturerData()) > 0 {
		fmt.Printf("%s MD: %X", comma, a.ManufacturerData())
	}
	fmt.Printf("\n")
}

func chkErr(err error) {
	switch errors.Cause(err) {
	case nil:
	case context.DeadlineExceeded:
		fmt.Printf("done\n")
	case context.Canceled:
		fmt.Printf("canceled\n")
	default:
		log.Fatalf(err.Error())
	}
}
