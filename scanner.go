package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

type Config struct {
	Extensions []string `json:"extensions"`
}

func loadConfig() Config {
	file, err := os.ReadFile("config.json")
	if err != nil {
		fmt.Println("Failed to read config file:", err)
		os.Exit(1)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		fmt.Println("Failed to parse config file:", err)
		os.Exit(1)
	}

	return config
}

func checkInternetConnection() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	_, err := client.Get("https://www.google.com")
	return err == nil
}

func checkWaybackMachine() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://web.archive.org")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

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

func fetchURLsConcurrently(apiURL string, params url.Values, numWorkers int) ([]string, error) {
	var wg sync.WaitGroup
	urlChan := make(chan string, numWorkers)
	var urls []string
	var mu sync.Mutex

	worker := func() {
		defer wg.Done()
		for url := range urlChan {
			mu.Lock()
			urls = append(urls, url)
			mu.Unlock()
		}
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker()
	}

	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(body), "\n") {
		if line != "" {
			urlChan <- line
		}
	}
	close(urlChan)

	wg.Wait()
	return urls, nil
}

func filterURLs(data string, extensions []string) []string {
	regexPattern := `\.(` + strings.Join(extensions, "|") + `)(\?.*)?$`
	re := regexp.MustCompile(regexPattern)

	lines := strings.Split(data, "\n")
	var filteredURLs []string

	for _, line := range lines {
		if line != "" && re.MatchString(line) {
			filteredURLs = append(filteredURLs, line)
		}
	}

	return filteredURLs
}

func saveResultsByExtension(urls []string, domain string, outputDir string) {
	extensionMap := make(map[string][]string)
	re := regexp.MustCompile(`\.([a-zA-Z0-9]+)(\?.*)?$`)

	for _, url := range urls {
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			ext := matches[1]
			extensionMap[ext] = append(extensionMap[ext], url)
		}
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"File Found", "File Name", "Status", "URL Count"})
	table.SetBorder(false)
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
	)
	table.SetColumnColor(
		tablewriter.Colors{tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.FgHiWhiteColor},
		tablewriter.Colors{tablewriter.FgHiYellowColor},
		tablewriter.Colors{tablewriter.FgHiMagentaColor},
	)

	var wg sync.WaitGroup
	var mu sync.Mutex
	totalURLs := 0

	for ext, urls := range extensionMap {
		wg.Add(1)
		go func(ext string, urls []string) {
			defer wg.Done()
			fileName := fmt.Sprintf("%s.%s.txt", domain, ext)
			filePath := filepath.Join(outputDir, fileName)
			content := strings.Join(urls, "\n")

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				mu.Lock()
				table.Append([]string{ext, fileName, color.RedString("Failed"), fmt.Sprintf("%d URLs", len(urls))})
				mu.Unlock()
			} else {
				mu.Lock()
				table.Append([]string{ext, fileName, color.GreenString("Success"), fmt.Sprintf("%d URLs", len(urls))})
				mu.Unlock()
			}
			totalURLs += len(urls)
		}(ext, urls)
	}
	wg.Wait()

	table.Append([]string{"", "", "", ""})
	table.Append([]string{"", "", "-------------------", "-------------------"})
	table.Append([]string{"", "", "TOTAL", fmt.Sprintf("%d URLs", totalURLs)})

	fmt.Println("\nResults Summary:")
	table.Render()
}

func displayWelcomeMessage() {
	color.Cyan(`
  ____ _               _   _   _             _            
 / ___| |__   ___  ___| |_| | | |_   _ _ __ | |_ ___ _ __ 
| |  _| '_ \ / _ \/ __| __| |_| | | | | '_ \| __/ _ \ '__|
| |_| | | | | (_) \__ \ |_|  _  | |_| | | | | ||  __/ |   
 \____|_| |_|\___/|___/\__|_| |_|\__,_|_| |_|\__\___|_|      
	`)
	color.Green("\nWelcome to GhostHunter!")
	color.Yellow("Unearth hidden treasures from the Wayback Machine!")
	color.Yellow("Let's hunt some ghosts! 👻")
	color.Cyan("\nCreated by: Mysteriza & Deepseek AI")
	color.Cyan("-------------------------------------------------------------")
}

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

func fetchSnapshots(urls []string, domain string) {
	client := &http.Client{Timeout: 120 * time.Second}
	numWorkers := 5
	var wg sync.WaitGroup
	urlChan := make(chan string, numWorkers)

	outputDir := filepath.Join("results", domain)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Println("Failed to create directory:", err)
		return
	}

	outputFile := filepath.Join(outputDir, domain+".snapshots.txt")
	file, err := os.Create(outputFile)
	if err != nil {
		fmt.Println("Failed to create file:", err)
		return
	}
	defer file.Close()

	worker := func() {
		defer wg.Done()
		for url := range urlChan {
			if url == "" {
				continue
			}

			apiURL := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%s&output=text&fl=timestamp,original", url)
			resp, err := client.Get(apiURL)
			if err != nil {
				fmt.Println("Failed to fetch snapshots for URL:", url, err)
				continue
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Println("Failed to read response body for URL:", url, err)
				continue
			}

			lines := strings.Split(string(body), "\n")
			if len(lines) > 1 {
				fmt.Printf("\nSnapshots for URL: %s\n", url)
				fmt.Fprintf(file, "Snapshots for URL: %s\n", url)

				for _, line := range lines {
					if line != "" {
						parts := strings.Fields(line)
						if len(parts) >= 2 {
							timestamp := parts[0]
							originalURL := parts[1]
							snapshotURL := fmt.Sprintf("https://web.archive.org/web/%s/%s", timestamp, originalURL)
							fmt.Println("  -", snapshotURL)
							fmt.Fprintln(file, "  -", snapshotURL)
						}
					}
				}
				fmt.Fprintln(file)
			} else {
				fmt.Printf("\nNo snapshots found for URL: %s\n", url)
				fmt.Fprintf(file, "No snapshots found for URL: %s\n\n", url)
			}

			time.Sleep(500 * time.Millisecond)
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

	fmt.Printf("\nAll snapshots saved to: %s\n", outputFile)
}

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
	fetchSnapshots(urls, domain)

	duration := time.Since(startTime)
	fmt.Printf("\nTotal duration for Snapshots Scan: %.2f seconds\n", duration.Seconds())
}

func main() {
	displayWelcomeMessage()

	startTime := time.Now()

	color.Cyan("\nChecking internet connection...")
	if !checkInternetConnection() {
		color.Red("No internet or slow connection. Please check your network and try again.")
		return
	}
	color.Green("Connected to the Internet!")

	color.Cyan("Checking Wayback Machine availability...")
	if !checkWaybackMachine() {
		color.Red("Wayback Machine is currently DOWN. Please try again later.")
		return
	}
	color.Green("Wayback Machine is UP and running.")

	config := loadConfig()

	var domain string
	color.Cyan("\nEnter the domain to search (e.g., target.com): ")
	_, err := fmt.Scanln(&domain)
	if err != nil {
		color.Red("Error reading domain input: %v\n", err)
		return
	}

	color.Cyan("Checking domain availability...")
	if !checkDomainAvailability(domain) {
		color.Red("Domain is not reachable. Please check the domain and try again.")
		return
	}
	color.Green("Domain is active!")

	outputDir := filepath.Join("results", domain)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		color.Red("Failed to create directory '%s': %v\n", outputDir, err)
		return
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

	urls, err := fetchURLsConcurrently(apiURL, params, 5)
	if err != nil {
		s.Stop()
		color.Red("Error fetching URLs: %v\n", err)
		return
	}
	s.Stop()

	color.Cyan("\nTotal URLs found (before filtering): %d\n", len(urls))

	filteredURLs := filterURLs(strings.Join(urls, "\n"), config.Extensions)
	saveResultsByExtension(filteredURLs, domain, outputDir)

	color.Green("Process completed! Results saved in directory '%s'.\n", outputDir)

	searchSnapshots()

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	formattedDuration := fmt.Sprintf("%.2f seconds", duration.Seconds())

	color.Cyan("\nTOTAL duration: %s\n", formattedDuration)
}