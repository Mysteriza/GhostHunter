package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

func main() {
	displayWelcomeMessage()

	startTime := time.Now()

	config := loadConfig()

	var domain string
	color.Cyan("\nEnter the domain to search (e.g., target.com): ")
	_, err := fmt.Scanln(&domain)
	if err != nil {
		color.Red("Error reading domain input: %v\n", err)
		return
	}

	if err := runGhostHunter(config, domain); err != nil {
		color.Red("Error: %v\n", err)
		return
	}

	searchSnapshots()

	duration := time.Since(startTime)
	color.Cyan("\nTOTAL duration: %.2f seconds\n", duration.Seconds())
}

// runGhostHunter contains the core logic of the program
func runGhostHunter(config Config, domain string) error {
	if !checkInternetConnection() {
		return fmt.Errorf("no internet or slow connection")
	}
	color.Green("Connected to the Internet!")

	if !checkWaybackMachine() {
		return fmt.Errorf("Wayback Machine is currently DOWN")
	}
	color.Green("Wayback Machine is UP and running.")

	if !checkDomainAvailability(domain) {
		return fmt.Errorf("domain is not reachable")
	}
	color.Green("Domain is active!")

	outputDir := filepath.Join("results", domain)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory '%s': %v", outputDir, err)
	}
	color.Green("\nDirectory '%s' created successfully.\n", outputDir)

	s := spinner.New(spinner.CharSets[36], 100*time.Millisecond)
	s.Prefix = "\nFetching data from Wayback Machine "
	s.Start()

	apiURL := "https://web.archive.org/cdx/search/cdx"
	params := url.Values{}
	params.Add("url", "*."+domain+"/*")
	params.Add("collapse", "urlkey")
	params.Add("output", "text")
	params.Add("fl", "original")

	urlsChan, errChan := fetchURLsConcurrently(apiURL, params)

	// We need to wait for fetching to complete or at least handle errors
	// But filterURLs consumes the channel, so we can run it directly.
	// However, we also want to know the total count of found URLs which we can't know until we consume the channel.
	// filterURLs returns a slice, so it consumes the whole channel.

	filteredURLs := filterURLs(urlsChan, config.Extensions)

	// Check for errors from fetching
	select {
	case err := <-errChan:
		if err != nil {
			s.Stop()
			return fmt.Errorf("error fetching URLs: %v", err)
		}
	default:
	}
	s.Stop()

	color.Cyan("\nTotal URLs found (filtered): %d\n", len(filteredURLs))

	saveResultsByExtension(filteredURLs, domain, outputDir)

	color.Green("Process completed! Results saved in directory '%s'.\n", outputDir)
	return nil
}
