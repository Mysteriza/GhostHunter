package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

func main() {
	// Ask the user for the domain to search
	var domain string
	color.Cyan("Enter the domain to search (e.g., target.com): ")
	_, err := fmt.Scanln(&domain)
	if err != nil {
		color.Red("Error reading domain input: %v\n", err)
		return
	}

	// Create a directory to store the results
	outputDir := filepath.Join("results", domain)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		color.Red("Failed to create directory '%s': %v\n", outputDir, err)
		return
	}
	color.Green("Directory '%s' created successfully.\n", outputDir)

	// Show a spinner while fetching data from Wayback Machine
	s := spinner.New(spinner.CharSets[36], 100*time.Millisecond)
	s.Prefix = "Fetching data from Wayback Machine "
	s.Start()

	// Prepare the Wayback Machine CDX API URL and parameters
	apiURL := "https://web.archive.org/cdx/search/cdx"
	params := url.Values{}
	params.Add("url", "*."+domain+"/*")
	params.Add("collapse", "urlkey")
	params.Add("output", "text")
	params.Add("fl", "original")

	// Make an HTTP GET request
	resp, err := http.Get(apiURL + "?" + params.Encode())
	if err != nil {
		s.Stop()
		color.Red("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Stop()
		color.Red("Error reading response: %v\n", err)
		return
	}
	s.Stop()

	// Display the total number of URLs found before filtering
	unfilteredURLs := strings.Split(string(body), "\n")
	color.Cyan("\nTotal URLs found (before filtering): %d\n", len(unfilteredURLs))

	// Process and filter URLs
	filteredURLs := filterURLs(string(body))

	// Save the results into separate files based on file extensions
	saveResultsByExtension(filteredURLs, domain, outputDir)

	color.Green("Process completed! Results saved in directory '%s'.\n", outputDir)
}

// Function to filter URLs based on desired file extensions
func filterURLs(data string) []string {
	// Regex to match desired file extensions
	regexPattern := `\.(config|yml|yaml|env|ini|properties|sql|db|backup|dump|log|cache|secret|pem|key|cer|pfx|php|js|py|java|rb|txt|csv|xml|json|pdf|doc|docx|xls|xlsx|zip|tar\.gz|7z|rar)(\?.*)?$`
	re := regexp.MustCompile(regexPattern)

	// Split the data into lines
	lines := strings.Split(data, "\n")

	// Store URLs that match the regex
	var filteredURLs []string
	for _, line := range lines {
		if re.MatchString(line) {
			filteredURLs = append(filteredURLs, line)
		}
	}

	return filteredURLs
}

// Function to save results into separate files based on file extensions
func saveResultsByExtension(urls []string, domain string, outputDir string) {
	// Map to group URLs by their file extensions
	extensionMap := make(map[string][]string)

	// Regex to extract file extensions
	re := regexp.MustCompile(`\.([a-zA-Z0-9]+)(\?.*)?$`)

	// Group URLs by their extensions
	for _, url := range urls {
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			ext := matches[1]
			extensionMap[ext] = append(extensionMap[ext], url)
		}
	}

	// Prepare a table to display the results
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

	// Save the results into separate files
	var wg sync.WaitGroup
	totalURLs := 0 // Variable to store the total number of URLs

	// Use a mutex to safely append rows to the table
	var mu sync.Mutex

	// Iterate over all extensions found
	for ext, urls := range extensionMap {
		wg.Add(1)
		go func(ext string, urls []string) {
			defer wg.Done()
			fileName := fmt.Sprintf("%s.%s.txt", domain, ext)
			filePath := filepath.Join(outputDir, fileName)
			content := strings.Join(urls, "\n")

			// Attempt to save the file
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				mu.Lock()
				table.Append([]string{ext, fileName, color.RedString("Failed"), fmt.Sprintf("%d URLs", len(urls))})
				mu.Unlock()
			} else {
				mu.Lock()
				table.Append([]string{ext, fileName, color.GreenString("Success"), fmt.Sprintf("%d URLs", len(urls))})
				mu.Unlock()
			}
			totalURLs += len(urls) // Add to the total count
		}(ext, urls)
	}
	wg.Wait()

	// Add a separator line before the TOTAL row
	table.Append([]string{"", "", "", ""}) // Empty row for spacing
	table.Append([]string{"", "", "-------------------", "-------------------"}) // Separator line

	// Add a row for the total number of URLs
	table.Append([]string{"", "", "TOTAL", fmt.Sprintf("%d URLs", totalURLs)})

	// Render the table
	fmt.Println("\nResults Summary:")
	table.Render()

	// Add a closing message
	color.Cyan("\nPlease manually check the Wayback Machine for available snapshots (archives) of URLs found by this tool.\n")
}