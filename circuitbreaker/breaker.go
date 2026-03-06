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
		Interval:    time.Minute,
		Timeout:     15 * time.Second,

		ReadyToTrip: func(counts gobreaker.Counts) bool {

			return counts.ConsecutiveFailures >= 3
		},

		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {

			fmt.Printf(
				"Circuit breaker [%s] state change: %v -> %v\n",
				name,
				from,
				to,
			)
		},
	}

	return gobreaker.NewCircuitBreaker(settings)
}
