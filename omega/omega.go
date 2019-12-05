package main 

import (
	"fmt"
	"context"
	"time"

	"golang.org/x/exp/io/i2c"
	"github.com/quhar/bme280"
)

type Omega struct {
	r 		*Reporter
	i2c		*i2c.Device
	bme280  *bme280.BME280
}

func NewOmega(r *Reporter) (*Omega, error) {
	// Init sensor
	i2c, err := i2c.Open(&i2c.Devfs{Dev: "/dev/i2c-0"}, bme280.I2CAddr)
	if err != nil {
		panic(err)
	}

	bme280 := bme280.New(i2c)
	err = bme280.Init()
	
	return &Omega{r: r, i2c: i2c}, nil
}

func (o *Omega) Run(ctx context.Context) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	defer o.Close(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			// Read temperature from bme280
			if temp, hum, press, err := o.bme280.EnvData(); err == nil {
				o.r.Report("temperature", "", temp)
				fmt.Print(hum)
				fmt.Print(press)
			}
		}
	}
}

func (o *Omega) Close(ctx context.Context) error {
	o.i2c.Close()
	return nil
}