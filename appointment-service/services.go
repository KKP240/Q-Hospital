package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/KKP240/Q-Hospital/circuitbreaker"
)

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

var userServiceURL string

// Circuit Breaker สำหรับเรียก user-service
var userBreaker = circuitbreaker.NewBreaker("user-service")

// Retry HTTP request 3 ครั้ง
func httpGetWithRetry(ctx context.Context, url string) (*http.Response, error) {

	var resp *http.Response
	var err error

	for i := 0; i < 3; i++ {

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err = httpClient.Do(req)

		if err == nil {
			return resp, nil
		}

		// wait ก่อน retry
		time.Sleep(500 * time.Millisecond)
	}

	return nil, err
}

func GetUser(ctx context.Context, id string) (*UserResponse, error) {

	url := userServiceURL + "/users/" + id

	result, err := userBreaker.Execute(func() (interface{}, error) {

		resp, err := httpGetWithRetry(ctx, url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, nil
		}

		var user UserResponse

		if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
			return nil, err
		}

		return &user, nil
	})

	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return result.(*UserResponse), nil
}
