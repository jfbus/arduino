package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type ReporterConfig struct {
	URL  string `default:"http://35.198.66.194:2015/influxdb/write?db=demo"`
	Tags string `default:"room=raspi-grove"`
}

type Data struct {
	TS    time.Time
	Name  string
	Value float64
	Tags  string
}

type Reporter struct {
	cfg  ReporterConfig
	mu   sync.Mutex
	data []Data
}

func NewReporter(cfg ReporterConfig) *Reporter {
	return &Reporter{cfg: cfg}
}

func (r *Reporter) Run(ctx context.Context) error {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			r.mu.Lock()
			msg := fmt.Sprintf("alive,%s value=1\n", r.cfg.Tags)
			for _, d := range r.data {
				tags := d.Tags
				if tags != "" {
					tags = "," + tags
				}
				msg += fmt.Sprintf("%s,%s%s value=%f\n", d.Name, r.cfg.Tags, tags, d.Value)
			}
			r.data = r.data[:0]
			r.mu.Unlock()
			fmt.Println("sending", msg)
			resp, err := http.Post(r.cfg.URL, "application/x-www-form-urlencoded", bytes.NewBufferString(msg))
			if err != nil {
				fmt.Println("err", err)
			} else {
				defer resp.Body.Close()
				buf, _ := ioutil.ReadAll(resp.Body)
				fmt.Println("response", resp.StatusCode, "-", string(buf))
			}
		}
	}
}

func (r *Reporter) Report(ts time.Time, name, tags string, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = append(r.data, Data{TS: ts, Name: name, Tags: tags, Value: value})
}
