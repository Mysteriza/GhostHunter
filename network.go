package main

import (
	"bufio"
	"crypto/tls"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// checkInternetConnection verifies if there’s an active internet connection
func checkInternetConnection() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	_, err := client.Get("https://www.google.com")
	return err == nil
}

// checkWaybackMachine checks if the Wayback Machine is available
func checkWaybackMachine() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://web.archive.org")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// checkDomainAvailability checks if the domain is reachable
func checkDomainAvailability(domain string) bool {
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Get("https://" + domain)
	if err != nil {
		resp, err = client.Get("http://" + domain)
		if err != nil {
			return false
		}
	}
	defer resp.Body.Close()

	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// fetchURLsConcurrently retrieves URLs from the Wayback Machine API concurrently
func fetchURLsConcurrently(apiURL string, params url.Values, numWorkers int) ([]string, error) {
	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Use bufio.Scanner for efficient streaming
	scanner := bufio.NewScanner(resp.Body)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Process lines in parallel
	chunkSize := (len(lines) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup
	results := make([][]string, numWorkers)

	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(lines) {
			end = len(lines)
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(i int, chunk []string) {
			defer wg.Done()
			var localURLs []string
			for _, line := range chunk {
				if line != "" {
					localURLs = append(localURLs, line)
				}
			}
			results[i] = localURLs
		}(i, lines[start:end])
	}

	wg.Wait()

	// Combine results
	var urls []string
	for _, result := range results {
		urls = append(urls, result...)
	}
	return urls, nil
}