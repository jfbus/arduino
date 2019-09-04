package adc

import (
	"encoding/binary"

	"periph.io/x/periph/conn/i2c"
	"periph.io/x/periph/conn/mmr"
)

type Dev struct {
	m mmr.Dev8
}

type Opts struct {
	Addr int
}

// DefaultOpts is the recommended default options.
var DefaultOpts = Opts{
	Addr: 0x04,
}

func NewADC(bus i2c.Bus, opts *Opts) (*Dev, error) {
	i2cAddress := DefaultOpts.Addr
	if opts != nil {
		i2cAddress = opts.Addr
	}
	dev := &Dev{
		m: mmr.Dev8{
			Conn:  &i2c.Dev{Bus: bus, Addr: uint16(i2cAddress)},
			Order: binary.LittleEndian,
		},
	}
	return dev, nil
}

func (d *Dev) ReadRaw(channel uint8) (uint16, error) {
	addr := 0x10 + channel
	return d.m.ReadUint16(addr)
}

func (d *Dev) Read(channel uint8) (float64, error) {
	addr := 0x30 + channel
	res, err := d.m.ReadUint16(addr)
	if err != nil {
		return -1, err
	}
	return float64(res) / 10, nil
}
