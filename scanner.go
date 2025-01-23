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
	"log"
)

type Config struct {
	Extensions []string `json:"extensions"`
}

// Logging functions
func logInfo(message string) {
	log.Printf("[INFO] %s\n", message)
}

func logWarning(message string) {
	log.Printf("[WARNING] %s\n", message)
}

func logError(message string) {
	log.Printf("[ERROR] %s\n", message)
}

func setupLogging() *os.File {
	logFile, err := os.Create("ghosthunter.log")
	if err != nil {
		logError("Failed to create log file: " + err.Error())
		log.Fatal("Failed to create log file: ", err)
	}
	log.SetOutput(logFile)
	logInfo("Log file created successfully.")
	return logFile
}

func loadConfig() Config {
	file, err := os.ReadFile("config.json")
	if err != nil {
		logError("Failed to read config file: " + err.Error())
		log.Fatal("Failed to read config file: ", err)
	}

	var config Config
	if err := json.Unmarshal(file, &config); err != nil {
		logError("Failed to parse config file: " + err.Error())
		log.Fatal("Failed to parse config file: ", err)
	}

	logInfo("Config loaded successfully. Extensions: " + strings.Join(config.Extensions, ", "))
	return config
}

func checkInternetConnection() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	_, err := client.Get("https://www.google.com")
	if err != nil {
		logWarning("No internet connection: " + err.Error())
		return false
	}
	return true
}

func checkWaybackMachine() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://web.archive.org")
	if err != nil {
		logWarning("Wayback Machine is down: " + err.Error())
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
			logWarning("Domain is not reachable: " + err.Error())
			return false
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true
	}

	logWarning("Domain returned status code: " + fmt.Sprint(resp.StatusCode))
	return false
}

func fetchURLsConcurrently(apiURL string, params url.Values, numWorkers int) ([]string, error) {
	logInfo("Fetching URLs from API: " + apiURL)
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
		logError("Failed to fetch URLs: " + err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logError("Failed to read response body: " + err.Error())
		return nil, err
	}

	for _, line := range strings.Split(string(body), "\n") {
		if line != "" {
			urlChan <- line
		}
	}
	close(urlChan)

	wg.Wait()
	logInfo(fmt.Sprintf("Fetched %d URLs from API.", len(urls)))
	return urls, nil
}

func filterURLs(data string, extensions []string) []string {
	logInfo("Filtering URLs based on extensions: " + strings.Join(extensions, ", "))
	regexPattern := `\.(` + strings.Join(extensions, "|") + `)(\?.*)?$`
	re := regexp.MustCompile(regexPattern)

	lines := strings.Split(data, "\n")
	var filteredURLs []string

	for _, line := range lines {
		if line != "" && re.MatchString(line) {
			filteredURLs = append(filteredURLs, line)
		}
	}

	logInfo(fmt.Sprintf("Filtered %d URLs.", len(filteredURLs)))
	return filteredURLs
}

func saveResultsByExtension(urls []string, domain string, outputDir string) {
	logInfo("Saving results by extension to directory: " + outputDir)
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
				logError(fmt.Sprintf("Failed to save file %s: %s", fileName, err.Error()))
				table.Append([]string{ext, fileName, color.RedString("Failed"), fmt.Sprintf("%d URLs", len(urls))})
				mu.Unlock()
			} else {
				mu.Lock()
				logInfo(fmt.Sprintf("Successfully saved file %s with %d URLs.", fileName, len(urls)))
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

	logInfo(fmt.Sprintf("Total URLs saved: %d", totalURLs))
	color.Cyan("\nPlease manually check the Wayback Machine for available snapshots (archives) of URLs found by this tool.\n")
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
			// Extract the extension from the filename (e.g., "itenas.ac.id.pdf.txt" -> "pdf")
			fileName := entry.Name()
			parts := strings.Split(fileName, ".")
			if len(parts) >= 3 { // Ensure the filename has at least 3 parts (domain, extension, "txt")
				ext := parts[len(parts)-2] // The extension is the second-to-last part
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

func fetchSnapshots(urls []string) (map[string][]string, error) {
	snapshots := make(map[string][]string)
	client := &http.Client{Timeout: 30 * time.Second} // Increased timeout for Wayback Machine queries

	for _, url := range urls {
		if url == "" {
			continue // Skip empty lines
		}

		// Query Wayback Machine for snapshots of the URL
		apiURL := fmt.Sprintf("https://web.archive.org/cdx/search/cdx?url=%s&output=json", url)
		resp, err := client.Get(apiURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch snapshots for URL %s: %v", url, err)
		}
		defer resp.Body.Close()

		// Parse the JSON response
		var result [][]string
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode Wayback Machine response for URL %s: %v", url, err)
		}

		// Extract snapshot dates and links
		var snapshotLinks []string
		for _, entry := range result[1:] { // Skip the header row
			if len(entry) >= 3 { // Ensure the entry has at least 3 fields (timestamp, original URL, snapshot URL)
				timestamp := entry[0]
				originalURL := entry[1]

				// Ensure the original URL is properly formatted
				if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
					originalURL = "https://" + originalURL
				}

				// Construct the snapshot URL in the correct format
				snapshotURL := fmt.Sprintf("https://web.archive.org/web/%s/%s", timestamp, originalURL)
				snapshotLinks = append(snapshotLinks, snapshotURL)
			}
		}

		if len(snapshotLinks) > 0 {
			snapshots[url] = snapshotLinks
		}
	}

	return snapshots, nil
}


func saveSnapshotResults(snapshots map[string][]string, domain string, outputDir string) {
	logInfo("Saving snapshot results to directory: " + outputDir)
	for ext, urls := range snapshots {
		fileName := fmt.Sprintf("%s.%s.snapshots.txt", domain, ext)
		filePath := filepath.Join(outputDir, fileName)
		content := strings.Join(urls, "\n")

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			logError(fmt.Sprintf("Failed to save snapshot file %s: %s", fileName, err.Error()))
		} else {
			logInfo(fmt.Sprintf("Successfully saved snapshot file %s with %d URLs.", fileName, len(urls)))
		}
	}
}

func searchSnapshots() {
	// Ask user if they want to search for snapshots
	var choice string
	color.Cyan("\nDo you want to search for snapshots of the found URLs? (y/n): ")
	_, err := fmt.Scanln(&choice)
	if err != nil || strings.ToLower(choice) != "y" {
		color.Yellow("Snapshot search skipped.")
		return
	}

	// Let user select a domain
	domain, err := selectDomain()
	if err != nil {
		color.Red("Error selecting domain: %v\n", err)
		logError("Error selecting domain: " + err.Error())
		return
	}

	// Let user select extensions
	extensions, err := selectExtensions(domain)
	if err != nil {
		color.Red("Error selecting extensions: %v\n", err)
		logError("Error selecting extensions: " + err.Error())
		return
	}

	// Initialize spinner for snapshot fetching
	s := spinner.New(spinner.CharSets[36], 100*time.Millisecond)
	s.Prefix = "Fetching snapshots "
	s.Start()

	// Fetch snapshots for selected extensions
	snapshots := make(map[string][]string)
	for _, ext := range extensions {
		fileName := fmt.Sprintf("%s.%s.txt", domain, ext)
		filePath := filepath.Join("results", domain, fileName)

		content, err := os.ReadFile(filePath)
		if err != nil {
			color.Red("Failed to read file %s: %v\n", fileName, err)
			logError(fmt.Sprintf("Failed to read file %s: %v", fileName, err))
			continue
		}

		urls := strings.Split(string(content), "\n")
		snapshotResults, err := fetchSnapshots(urls)
		if err != nil {
			color.Red("Failed to fetch snapshots for extension %s: %v\n", ext, err)
			logError(fmt.Sprintf("Failed to fetch snapshots for extension %s: %v", ext, err))
			continue
		}

		// Save snapshot results for the extension
		for url, snapshotLinks := range snapshotResults {
			snapshots[ext] = append(snapshots[ext], fmt.Sprintf("URL: %s", url))
			for _, link := range snapshotLinks {
				snapshots[ext] = append(snapshots[ext], fmt.Sprintf("  - %s", link))
			}
		}
	}

	s.Stop()

	// Display results in a clean format
	color.Cyan("\nSnapshot Search Results:")
	for ext, results := range snapshots {
		color.Green("\nExtension: %s", ext)
		for _, line := range results {
			color.Yellow(line)
		}
	}

	// Save snapshot results
	saveSnapshotResults(snapshots, domain, filepath.Join("results", domain))
}

func main() {
	displayWelcomeMessage()

	startTime := time.Now()
	logFile := setupLogging()
	defer logFile.Close()

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
		logError("Error reading domain input: " + err.Error())
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
		logError("Failed to create directory '" + outputDir + "': " + err.Error())
		return
	}
	color.Green("\nDirectory '%s' created successfully.\n", outputDir)
	logInfo("Directory '" + outputDir + "' created successfully.")

	s := spinner.New(spinner.CharSets[36], 100*time.Millisecond)
	s.Prefix = "\nFetching data from Wayback Machine "
	s.Start()

	apiURL := "https://web.archive.org/cdx/search/cdx"
	params := url.Values{}
	params.Add("url", "*." + domain + "/*")
	params.Add("collapse", "urlkey")
	params.Add("output", "text")
	params.Add("fl", "original")

	urls, err := fetchURLsConcurrently(apiURL, params, 5) // 5 workers
	if err != nil {
		s.Stop()
		color.Red("Error fetching URLs: %v\n", err)
		logError("Error fetching URLs: " + err.Error())
		return
	}
	s.Stop()

	color.Cyan("\nTotal URLs found (before filtering): %d\n", len(urls))
	logInfo(fmt.Sprintf("Total URLs found (before filtering): %d", len(urls)))

	filteredURLs := filterURLs(strings.Join(urls, "\n"), config.Extensions)
	saveResultsByExtension(filteredURLs, domain, outputDir)

	color.Green("Process completed! Results saved in directory '%s'.\n", outputDir)
	logInfo("Process completed! Results saved in directory '" + outputDir + "'.")

	// Search for snapshots
	searchSnapshots()

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	formattedDuration := fmt.Sprintf("%.2f seconds", duration.Seconds())

	logInfo("Program ended at: " + endTime.Format("2006-01-02 15:04:05"))
	logInfo("Total duration: " + formattedDuration)

	color.Cyan("\nTotal duration: %s\n", formattedDuration)
}