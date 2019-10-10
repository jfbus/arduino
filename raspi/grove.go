package main

import (
	"bytes"
	"context"
	"net"
	"sync/atomic"
	"time"

	"adc"

	"github.com/pkg/errors"
	"periph.io/x/periph/conn/gpio"
	"periph.io/x/periph/conn/gpio/gpioreg"
	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/i2c/i2creg"
	"periph.io/x/periph/experimental/devices/mcp9808"
	"periph.io/x/periph/host"
)

type GroveConfig struct {
	PIRMaxAge time.Duration `envconfig:"PIR_MAX_AGE" default:"5m"`
}

type Grove struct {
	cfg GroveConfig

	r *Reporter

	i2c       i2c.BusCloser
	mcp9808   *mcp9808.Dev
	pir       gpio.PinIO
	pirStatus int64
	adc       *adc.Dev
}

func NewGrove(cfg GroveConfig, r *Reporter) (*Grove, error) {
	if _, err := host.Init(); err != nil {
		return nil, errors.Wrap(err, "open host")
	}

	// Use i2creg I²C bus registry to find the first available I²C bus.
	i2c, err := i2creg.Open("")
	if err != nil {
		return nil, errors.Wrap(err, "open i2c bus")
	}

	mcp9808, err := mcp9808.New(i2c, &mcp9808.DefaultOpts)
	if err != nil {
		return nil, errors.Wrap(err, "init mcp9808")
	}

	p := gpioreg.ByName("16")
	if p == nil {
		return nil, errors.New("unable to get PIR pin")
	}
	if err = p.In(gpio.PullNoChange, gpio.RisingEdge); err != nil {
		return nil, errors.Wrap(err, "setup PIR pin")
	}

	adc, err := adc.NewADC(i2c, nil)
	if err != nil {
		return nil, errors.Wrap(err, "init adc")
	}
	return &Grove{r: r, i2c: i2c, mcp9808: mcp9808, pir: p, adc: adc, cfg: cfg}, nil
}

func (g *Grove) Run(ctx context.Context) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	defer g.Close(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			// Read temperature from mcp9808
			if temp, err := g.mcp9808.SenseTemp(); err == nil {
				g.r.Report("temperature", "",temp.Celsius())
			}
			// Read PIR status
			pirStatus := 0.
			if time.Since(time.Unix(atomic.LoadInt64(&g.pirStatus), 0)) > g.cfg.PIRMaxAge {
				pirStatus = 1
			}
			g.r.Report("pir", "", pirStatus)
			// Read luminosity
			if lum, err := g.adc.Read(0); err == nil {
				g.r.Report("luminosity", "", lum)
			}

			// Read loudness
			if mic, err := g.adc.Read(2); err == nil {
				g.r.Report("loudness", "", mic)
			}

			// Read bssid
            active := isNull(getMacAddr())
			g.r.Report("wifi1", "", active)
		}
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

func isNull(str string) float64 {
    if len(str) > 0 {
        return 1
    }
    return 0
}

func (g *Grove) monitorPIR(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		g.pir.WaitForEdge(-1)
		if g.pir.Read() == gpio.High {
			atomic.StoreInt64(&g.pirStatus, time.Now().Unix())
		}
	}
}

func (g *Grove) Close(ctx context.Context) error {
	g.i2c.Close()
	return nil
}
