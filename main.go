package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const (
	MaxIterations = 100
)

type RateLimitedEntity string

type RateLimiter interface {
	Limit(ctx context.Context, entity RateLimitedEntity) (bool, error)
}

type FixedRateLimiter struct {
	maxLimit     int
	currentLimit int
	fillInterval time.Duration

	doneCh chan bool
	sync.Mutex
	ctx context.Context
}

func NewFixedRateLimiter(ctx context.Context, maxLimit int, fillInterval time.Duration) *FixedRateLimiter {
	doneCh := make(chan bool, 1)

	return &FixedRateLimiter{
		maxLimit:     maxLimit,
		fillInterval: fillInterval,
		doneCh:       doneCh,
		ctx:          ctx,
	}
}

func (f *FixedRateLimiter) Limit(ctx context.Context, entity RateLimitedEntity) (bool, error) {
	f.Lock()
	defer f.Unlock()
	if f.currentLimit <= f.maxLimit {
		f.currentLimit += 1
		return true, nil
	}
	return false, nil
}

func (f *FixedRateLimiter) Start(ctx context.Context) error {
	log.Printf("fill interval set to %d seconds", f.fillInterval)
	ticker := time.NewTicker(f.fillInterval)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("filling tokens")
				f.Lock()
				log.Println("filling tokens: aquired lock")
				f.currentLimit = 0
				log.Println("filled tokens")
				f.Unlock()
			case <-ctx.Done():
				log.Println("context is done")
				f.doneCh <- true
			}
		}
	}()

	<-f.doneCh
	return nil
}

func (f *FixedRateLimiter) Stop(ctx context.Context) error {
	close(f.doneCh)
	return nil
}

func main() {

	// signal channel to listen for SIGINT and SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// context
	ctx := context.Background()
	ctx, cancelFunc := context.WithCancel(ctx)

	rateLimiter := NewFixedRateLimiter(ctx, 10, time.Duration(10)*time.Second)
	go func() {
		rateLimiter.Start(ctx)
	}()

	go func() {
		<-sigs
		log.Println("calling cancelFunc")
		cancelFunc()
		os.Exit(1)
	}()

	defer rateLimiter.Stop(ctx)

	for i := 0; i <= MaxIterations; i++ {
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

		ok, err := rateLimiter.Limit(ctx, "127.0.0.1")
		if err != nil {
			log.Fatal(err)
		}

		if !ok {
			log.Printf("thottled at %d\n", i)
		} else {
			log.Printf("request no %d allowed\n", i)
		}
	}
}
