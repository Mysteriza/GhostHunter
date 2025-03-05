package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

	urls, err := fetchURLsConcurrently(apiURL, params, config.NumWorkers)
	if err != nil {
		s.Stop()
		return fmt.Errorf("error fetching URLs: %v", err)
	}
	s.Stop()

	color.Cyan("\nTotal URLs found (before filtering): %d\n", len(urls))

	filteredURLs := filterURLs(strings.Join(urls, "\n"), config.Extensions)
	saveResultsByExtension(filteredURLs, domain, outputDir)

	color.Green("Process completed! Results saved in directory '%s'.\n", outputDir)
	return nil
}