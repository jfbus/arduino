package main

import (
	"context"
	"time"

	"adc"

	"github.com/pkg/errors"
	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/conn/physic"
	"periph.io/x/periph/devices/bmxx80"
	"periph.io/x/periph/host"
)

type Enviro struct {
	r *Reporter

	i2c   i2c.BusCloser
	bm280 *bmxx80.Dev

	adc   *adc.Dev
}

func NewEnviro(r *Reporter) (*Enviro, error) {
	if _, err := host.Init(); err != nil {
		return nil, errors.Wrap(err, "open host")
	}

	// Use i2creg I²C bus registry to find the first available I²C bus.
	i2c, err := i2creg.Open("")
	if err != nil {
		return nil, errors.Wrap(err, "open i2c bus")
	}

	bm280, err := bmxx80.NewI2C(i2c, 0x76, &bmxx80.DefaultOpts)
	if err != nil {
		return nil, errors.Wrap(err, "init bme280")
	}

	adc,err := adc.NewADC(i2c, nil)
	if err != nil {
		return nil, errors.Wrap(err, "init adc")
	}
	return &Enviro{r: r, i2c: i2c, bm280: bm280, adc: adc}, nil
}

func (e *Enviro) Run(ctx context.Context) error {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	defer e.Close(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			// Read temperature & humidity from BM280
			env := physic.Env{}
			if err := e.bm280.Sense(&env); err == nil {
				e.r.Report(time.Now(), "temp", "", env.Temperature.Celsius())
				e.r.Report(time.Now(), "humidity", "", float64(env.Humidity/physic.PercentRH))
			}
			// Read noise from ADC
		}
	}
}

func (e *Enviro) Close(ctx context.Context) error {
	e.i2c.Close()
	return nil
}
