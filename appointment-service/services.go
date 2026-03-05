package main

import (
	"encoding/json"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}
var userServiceURL string

// External Calls
func GetUser(id string) (*UserResponse, error) {
	url := userServiceURL + "/users/" + id

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var user UserResponse
	if err = json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}
