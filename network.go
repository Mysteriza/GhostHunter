package main

import (
	"bufio"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// Shared HTTP client with connection pooling
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	},
}

// checkInternetConnection verifies if there’s an active internet connection
func checkInternetConnection() bool {
	_, err := httpClient.Get("https://www.google.com")
	return err == nil
}

// checkWaybackMachine checks if the Wayback Machine is available
func checkWaybackMachine() bool {
	resp, err := httpClient.Get("https://web.archive.org")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// checkDomainAvailability checks if the domain is reachable
func checkDomainAvailability(domain string) bool {
	resp, err := httpClient.Get("https://" + domain)
	if err != nil {
		resp, err = httpClient.Get("http://" + domain)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// fetchURLsConcurrently retrieves URLs from the Wayback Machine API using streaming
func fetchURLsConcurrently(apiURL string, params url.Values) (<-chan string, <-chan error) {
	out := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		defer close(out)
		defer close(errChan)

		resp, err := httpClient.Get(apiURL + "?" + params.Encode())
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				out <- line
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return out, errChan
}
