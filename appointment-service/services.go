package main

import (
	"encoding/json"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}
var userServiceURL string

// External Calls
func GetPatient(id string) (*Patient, error) {
	url := userServiceURL + "/patients/" + id

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var patient Patient
	if err = json.NewDecoder(resp.Body).Decode(&patient); err != nil {
		return nil, err
	}

	return &patient, nil
}

func GetDoctor(id string) (*Doctor, error) {
	url := userServiceURL + "/doctors/" + id

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	var doctor Doctor
	if err = json.NewDecoder(resp.Body).Decode(&doctor); err != nil {
		return nil, err
	}

	return &doctor, nil
}
