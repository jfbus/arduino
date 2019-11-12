package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-ble/ble"
	"golang.org/x/sync/errgroup"
)

type BTType int

const (
	UnknownType BTType = iota
	Phone
	Tablet
	Computer
	OtherType
)

func (t BTType) String() string {
	switch t {
	case Phone:
		return "phone"
	case Tablet:
		return "tablet"
	case Computer:
		return "computer"
	case OtherType:
		return "other"
	default:
		return ""
	}
}

type BTManufacturer int

const (
	UnknownMfg BTManufacturer = iota
	Apple
	OtherMfg
)

func (t BTManufacturer) String() string {
	switch t {
	case Apple:
		return "apple"
	case OtherMfg:
		return "other"
	default:
		return ""
	}
}

type BTPeripheral struct {
	a            ble.Advertisement
	typ          BTType
	manufacturer BTManufacturer
	seen         time.Time
	attempts     int
}

type BTScannerConfig struct {
	MaxAge          time.Duration `default:"3m"`
	MinRSSI         int           `default:"-80"`
	DiscoveryBuffer int           `default:"100"`
	MaxConnAttempts int           `default:"5"`
}

type BTScanner struct {
	cfg         BTScannerConfig
	mu          sync.Mutex
	devices     map[string]*BTPeripheral
	r           *Reporter
	discoveries chan *BTPeripheral
}

func NewBTScanner(cfg BTScannerConfig, r *Reporter) *BTScanner {
	return &BTScanner{
		cfg:         cfg,
		devices:     map[string]*BTPeripheral{},
		r:           r,
		discoveries: make(chan *BTPeripheral, cfg.DiscoveryBuffer),
	}
}

func (s *BTScanner) Discover(a ble.Advertisement) {
	//fmt.Printf("Discovered %s (%s): rssi %d\n", a.Addr(), a.LocalName(), a.RSSI())
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, found := s.devices[a.Addr().String()]; !found {
		if a.RSSI() < s.cfg.MinRSSI {
			return
		}
		p := &BTPeripheral{a: a, seen: time.Now()}
		s.devices[a.Addr().String()] = p
		s.discoveries <- p
		return
	}
	s.devices[a.Addr().String()].seen = time.Now()
}

func (s *BTScanner) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return s.loopReport(ctx)
	})
	g.Go(func() error {
		return s.loopInfo(ctx)
	})
	return g.Wait()
}

func (s *BTScanner) loopReport(ctx context.Context) error {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			s.mu.Lock()
			cnt := 0.
			typs := map[BTType]float64{}
			for id, p := range s.devices {
				if time.Since(p.seen) > s.cfg.MaxAge {
					delete(s.devices, id)
					fmt.Printf("peripheral %s (type %s/mfg %s): left (last seen %s ago)\n", id, p.typ, p.manufacturer, time.Since(p.seen))
					continue
				}
				fmt.Printf("peripheral %s (type %s/mfg %s): rssi %d (last seen %s ago)\n", id, p.typ, p.manufacturer, p.a.RSSI(), time.Since(p.seen))
				cnt++
				typs[p.typ]++
			}
			s.mu.Unlock()
			s.r.Report("devices", "", cnt)
			for typ, cnt := range typs {
				if typ == UnknownType {
					continue
				}
				s.r.Report("devicetype", "type="+typ.String(), cnt)
			}
		}
	}
}

func (s *BTScanner) loopInfo(ctx context.Context) error {
	discoveries := map[string]*BTPeripheral{}
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case p := <-s.discoveries:
			if !p.a.Connectable() {
				continue
			}
			discoveries[p.a.Addr().String()] = p
		case <-t.C:
			for id, p := range discoveries {
				if s.discoverInfo(ctx, p) == nil || p.attempts > s.cfg.MaxConnAttempts {
					delete(discoveries, id)
				}
			}
		}
	}
}

func (s *BTScanner) discoverInfo(ctx context.Context, p *BTPeripheral) error {
	ctx, cancel := context.WithTimeout(ctx, 30 * time.Second)
	defer cancel()
	fmt.Println("discovering", p.a.Addr())
	p.attempts++
	c, err := ble.Dial(ctx, p.a.Addr())
	if err != nil {
		fmt.Println("unable to connect to", p.a.Addr(), ":", err)
		return err
	}
	defer c.CancelConnection()
	prof, err := c.DiscoverProfile(true)
	if err != nil {
		fmt.Println("unable to discover", p.a.Addr(), ":", err)
		return err
	}
	// Manufacturer
	char := prof.FindCharacteristic(&ble.Characteristic{UUID: ble.UUID16(0x2a29)})
	if char != nil {
		b, err := c.ReadCharacteristic(char)
		if err != nil {
			fmt.Println("unable to read", p.a.Addr(), ":", err)
			return err
		}
		fmt.Printf("Manufacturer: %x | %q\n", b, b)
		switch {
		case bytes.HasPrefix(b, []byte(`Apple `)):
			p.manufacturer = Apple
		default:
			p.manufacturer = OtherMfg
		}
	}
	// Model Number String
	char = prof.FindCharacteristic(&ble.Characteristic{UUID: ble.UUID16(0x2a24)})
	if char != nil {
		b, err := c.ReadCharacteristic(char)
		if err != nil {
			fmt.Println("unable to read", p.a.Addr(), ":", err)
			return err
		}
		fmt.Printf("Model: %x | %q\n", b, b)
		switch {
		case bytes.HasPrefix(b, []byte(`Mac`)):
			p.typ = Computer
		case bytes.HasPrefix(b, []byte(`iPad`)):
			p.typ = Tablet
		case bytes.HasPrefix(b, []byte(`iPhone`)):
			p.typ = Phone
		default:
			p.typ = OtherType
		}
	}
	// Device Information
    /*char = prof.FindCharacteristic(&ble.Characteristic{UUID: ble.UUID16(0x180a)})
    if char != nil {
        b, err := c.ReadCharacteristic(char)
        if err != nil {
            fmt.Println("unable to read", p.a.Addr(), ":", err)
            return err
        }
        fmt.Printf("Devince Information:%x | %q\n", b, b)
    }*/

	for _, s := range prof.Services {
		fmt.Println("  srv", s.UUID)
		for _, char := range s.Characteristics {
			fmt.Println("    char", char.UUID)
			if (char.Property & ble.CharRead) != 0 {
				b, err := c.ReadCharacteristic(char)
				if err != nil {
					fmt.Printf("Failed to read characteristic: %s\n", err)
					continue
				}
				fmt.Printf("        Value         %x | %q\n", b, b)
			}
		}
	}
	return nil
}
