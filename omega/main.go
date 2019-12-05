package main

import (
    "context"
    "fmt"
    "log"

    "github.com/pkg/errors"
    "golang.org/x/sync/errgroup"
)

func main() {
    cfg := struct {
        Reporter ReporterConfig
    }{}
    
    r := NewReporter(cfg.Reporter)
    e, err := NewOmega(r)
    if err != nil {
        log.Fatalf("can't init omega : %s", err)
    }

    g, ctx := errgroup.WithContext(context.Background())
    g.Go(func() error { return r.Run(ctx) })
    g.Go(func() error { return e.Run(ctx) })
    chkErr(g.Wait())
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