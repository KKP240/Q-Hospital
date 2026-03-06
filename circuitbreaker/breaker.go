package circuitbreaker

import (
	"fmt"
	"time"

	"github.com/sony/gobreaker"
)

func NewBreaker(name string) *gobreaker.CircuitBreaker {

	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     15 * time.Second,
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			fmt.Println("Circuit breaker state change:", from, "->", to)
			fmt.Printf("Circuit breaker [%s] state change: %v -> %v\n", name, from, to)
		},
	}

	return gobreaker.NewCircuitBreaker(settings)
}
