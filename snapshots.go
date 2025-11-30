package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// listAvailableDomains lists domains in the results directory
func listAvailableDomains() ([]string, error) {
	entries, err := os.ReadDir("results")
	if err != nil {
		return nil, err
	}

	var domains []string
	for _, entry := range entries {
		if entry.IsDir() {
			domains = append(domains, entry.Name())
		}
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("no domains found in the results directory")
	}

	return domains, nil
}

// listAvailableExtensions lists available extensions for a domain
func listAvailableExtensions(domain string) ([]string, error) {
	domainDir := filepath.Join("results", domain)
	entries, err := os.ReadDir(domainDir)
	if err != nil {
		return nil, err
	}

	var extensions []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			fileName := entry.Name()
			parts := strings.Split(fileName, ".")
			if len(parts) >= 3 {
				ext := parts[len(parts)-2]
				extensions = append(extensions, ext)
			}
		}
	}

	if len(extensions) == 0 {
		return nil, fmt.Errorf("no valid extensions found in the domain directory")
	}

	return extensions, nil
}

// selectDomain prompts the user to select a domain
func selectDomain() (string, error) {
	domains, err := listAvailableDomains()
	if err != nil {
		return "", err
	}

	color.Cyan("\nAvailable domains:")
	for i, domain := range domains {
		color.Cyan("%d. %s", i+1, domain)
	}

	var choice int
	color.Cyan("\nSelect a domain by entering its number: ")
	_, err = fmt.Scanln(&choice)
	if err != nil || choice < 1 || choice > len(domains) {
		return "", fmt.Errorf("invalid selection")
	}

	return domains[choice-1], nil
}

// selectExtensions prompts the user to select extensions for a domain
func selectExtensions(domain string) ([]string, error) {
	extensions, err := listAvailableExtensions(domain)
	if err != nil {
		return nil, err
	}

	color.Cyan("\nAvailable extensions for domain %s:", domain)
	for i, ext := range extensions {
		color.Cyan("%d. %s", i+1, ext)
	}

	color.Cyan("\nSelect extensions by entering their numbers (comma-separated, e.g., 1,2,3): ")
	var input string
	_, err = fmt.Scanln(&input)
	if err != nil {
		return nil, err
	}

	choices := strings.Split(input, ",")
	var selectedExtensions []string
	for _, c := range choices {
		index, err := strconv.Atoi(strings.TrimSpace(c))
		if err != nil || index < 1 || index > len(extensions) {
			return nil, fmt.Errorf("invalid selection")
		}
		selectedExtensions = append(selectedExtensions, extensions[index-1])
	}

	return selectedExtensions, nil
}

// fetchSnapshots retrieves and saves snapshots for the given URLs
func fetchSnapshots(ctx context.Context, urls []string, domain string) {
	numWorkers := DefaultNumWorkers
	var wg sync.WaitGroup
	urlChan := make(chan string, numWorkers)

	outputDir := filepath.Join("results", domain)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		color.Red("Failed to create directory: %v\n", err)
		return
	}

	outputFile := filepath.Join(outputDir, domain+".snapshots.txt")
	file, err := os.Create(outputFile)
	if err != nil {
		color.Red("Failed to create file: %v\n", err)
		return
	}
	defer file.Close()

	summaryTable := tablewriter.NewWriter(os.Stdout)
	summaryTable.SetHeader([]string{"URL", "Snapshot Count"})
	summaryTable.SetBorder(false)
	summaryTable.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
	)
	summaryTable.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor},
	)

	var mu sync.Mutex

	worker := func() {
		defer wg.Done()
		for url := range urlChan {
			if url == "" {
				continue
			}

			apiURL := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%s&output=text&fl=timestamp,original", url)
			req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
			if err != nil {
				// color.Red("Failed to create request for URL: %s\nError: %v\n", url, err)
				continue
			}
			resp, err := httpClient.Do(req)
			if err != nil {
				// color.Red("Failed to fetch snapshots for URL: %s\nError: %v\n", url, err)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusTooManyRequests {
				color.Yellow("Rate limit exceeded for URL: %s. Waiting before retrying...\n", url)
				time.Sleep(10 * time.Second)
				// Ideally retry, but for now just skip or simple retry logic could be added
				continue
			}
			if resp.StatusCode != http.StatusOK {
				color.Red("Failed to fetch snapshots for URL: %s\nStatus Code: %d\n", url, resp.StatusCode)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				color.Red("Failed to read response body for URL: %s\nError: %v\n", url, err)
				continue
			}

			if len(body) == 0 || strings.Contains(string(body), "<html>") {
				color.Yellow("Invalid response for URL: %s\nResponse: %s\n", url, string(body))
				continue
			}

			lines := strings.Split(string(body), "\n")
			if len(lines) > 1 {
				color.Cyan("\n────────────────────────────────────────────────────────────────────────")
				color.Cyan("Snapshots for URL: %s", url)
				color.Cyan("────────────────────────────────────────────────────────────────────────")

				fmt.Fprintf(file, "Snapshots for URL: %s\n", url)

				for _, line := range lines {
					if line != "" {
						parts := strings.Fields(line)
						if len(parts) >= 2 {
							timestamp := parts[0]
							originalURL := parts[1]
							snapshotURL := fmt.Sprintf("https://web.archive.org/web/%s/%s", timestamp, originalURL)

							parsedTime, err := time.Parse("20060102150405", timestamp)
							if err != nil {
								color.Red("Failed to parse timestamp: %s\nError: %v\n", timestamp, err)
								continue
							}
							formattedTime := parsedTime.Format("02 January 2006, 15:04:05")

							color.Green("  - Timestamp: %s", color.YellowString(formattedTime))
							color.Green("    URL: %s", color.BlueString(snapshotURL))
							fmt.Fprintf(file, "  - Timestamp: %s\n    URL: %s\n", formattedTime, snapshotURL)
						}
					}
				}
				mu.Lock()
				summaryTable.Append([]string{url, fmt.Sprintf("%d snapshots", len(lines)-1)})
				mu.Unlock()
			} else {
				color.Yellow("\nNo snapshots found for URL: %s\n", url)
				fmt.Fprintf(file, "No snapshots found for URL: %s\n\n", url)
			}

			// mu.Lock()
			// processedCount++
			// fmt.Printf("\rProgress: %d/%d URLs processed", processedCount, totalURLs)
			// mu.Unlock()

			// time.Sleep(DefaultWorkerDelay) // Rate limiting delay - maybe reduce if using shared client with pooling?
			// Keep delay to be safe with Wayback Machine
		}
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	for _, url := range urls {
		urlChan <- url
	}
	close(urlChan)
	wg.Wait()
	fmt.Println() // Newline after progress bar

	color.Cyan("\n────────────────────────────────────────────────────────────────────────")
	color.Cyan("Summary of Snapshots")
	color.Cyan("────────────────────────────────────────────────────────────────────────")
	summaryTable.Render()

	color.Green("\nAll snapshots saved to: %s\n", outputFile)
}

// searchSnapshots prompts the user to search for snapshots
func searchSnapshots() {
	var choice string
	color.Cyan("\nDo you want to search for snapshots of the found URLs? (Y/n): ")
	_, err := fmt.Scanln(&choice)
	if err != nil || strings.ToLower(choice) != "y" {
		color.Yellow("Snapshot search skipped.")
		return
	}

	domain, err := selectDomain()
	if err != nil {
		color.Red("Error selecting domain: %v\n", err)
		return
	}

	extensions, err := selectExtensions(domain)
	if err != nil {
		color.Red("Error selecting extensions: %v\n", err)
		return
	}

	s := spinner.New(spinner.CharSets[36], 100*time.Millisecond)
	s.Prefix = "Fetching snapshots "
	s.Start()

	var urls []string
	for _, ext := range extensions {
		fileName := fmt.Sprintf("%s.%s.txt", domain, ext)
		filePath := filepath.Join("results", domain, fileName)

		content, err := os.ReadFile(filePath)
		if err != nil {
			color.Red("Failed to read file %s: %v\n", fileName, err)
			continue
		}

		urls = append(urls, strings.Split(string(content), "\n")...)
	}

	s.Stop()

	startTime := time.Now()
	ctx := context.Background() // Could be enhanced with timeout
	fetchSnapshots(ctx, urls, domain)

	duration := time.Since(startTime)
	fmt.Printf("\nTotal duration for Snapshots Scan: %.2f seconds\n", duration.Seconds())
}
