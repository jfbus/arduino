package main 

import (
	"context"
	"time"
	//"reflect"

	"github.com/pkg/errors"
	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/devices/bmxx80"
	"periph.io/x/periph/host"
)

type Omega struct {
	r 		*Reporter
	i2c		i2c.BusCloser
	bme280  *bmxx80.Dev
}

func NewOmega(r *Reporter) (*Omega, error) {
	// Init sensor
	if _, err := host.Init(); err != nil {
		return nil, errors.Wrap(err, "open host")
	}

	// Use i2creg I²C bus registry to find the first available I²C bus.
	i2c, err := i2creg.Open("")
	if err != nil {
		return nil, errors.Wrap(err, "open i2c bus")
	}

	bme280, err := bmxx80.NewI2C(i2c, 0x77, &bmxx80.DefaultOpts)
	if err != nil {
		return nil, errors.Wrap(err, "init mcp9808")
	}
	
	return &Omega{r: r, i2c: i2c, bme280: bme280}, nil
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
			env := physic.Env{}
			if err := o.bme280.Sense(&env); err == nil {
				o.r.Report("temp", "", env.Temperature.Celsius())
			}
		}
	}
}

func (o *Omega) Close(ctx context.Context) error {
	o.i2c.Close()
	return nil
}