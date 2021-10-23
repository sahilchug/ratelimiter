package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

const (
	MaxIterations = 100
)

type RateLimitedEntity string

type RateLimiter interface {
	Limit(entity RateLimitedEntity) (bool, error)
}

type FixedRateLimiter struct {
	maxLimit     int
	currentLimit int
	fillInterval time.Duration

	closeCh chan bool
	sync.Mutex
}

func NewFixedRateLimiter(maxLimit int, fillInterval time.Duration) *FixedRateLimiter {
	closeCh := make(chan bool, 1)

	return &FixedRateLimiter{
		maxLimit:     maxLimit,
		fillInterval: fillInterval,
		closeCh:      closeCh,
	}
}

func (f *FixedRateLimiter) Limit(entity RateLimitedEntity) (bool, error) {
	f.Lock()
	defer f.Unlock()
	if f.currentLimit <= f.maxLimit {
		f.currentLimit += 1
		return true, nil
	}
	return false, nil
}

func (f *FixedRateLimiter) Start() error {
	log.Println(f.fillInterval)
	ticker := time.NewTicker(time.Duration(f.fillInterval) * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			log.Println("filling tokens")
			f.Lock()
			log.Println("filling tokens: aquired lock")
			f.currentLimit = 0
			log.Println("filled tokens")
			f.Unlock()
		}
	}()

	<-f.closeCh
	return nil
}

func (f *FixedRateLimiter) Stop() error {
	close(f.closeCh)
	return nil
}

func main() {

	rateLimiter := NewFixedRateLimiter(10, 10)
	go func() {
		rateLimiter.Start()
	}()

	for i := 0; i <= MaxIterations; i++ {
		time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

		ok, err := rateLimiter.Limit("127.0.0.1")
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
