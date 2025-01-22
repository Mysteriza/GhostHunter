package main

import (
	"encoding/json"
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
	"log"
)

type Config struct {
	Extensions []string `json:"extensions"`
}

// Fungsi untuk logging info
func logInfo(message string) {
	log.Printf("[INFO] %s\n", message)
}

// Fungsi untuk logging warning
func logWarning(message string) {
	log.Printf("[WARNING] %s\n", message)
}

// Fungsi untuk logging error
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
func main() {
	// Tampilkan pesan selamat datang
	displayWelcomeMessage()

	// Catat waktu mulai program (hanya di log)
	startTime := time.Now()
	logFile := setupLogging()
	defer logFile.Close()

	config := loadConfig()

	var domain string
	color.Cyan("Enter the domain to search (e.g., target.com): ")
	_, err := fmt.Scanln(&domain)
	if err != nil {
		color.Red("Error reading domain input: %v\n", err)
		logError("Error reading domain input: " + err.Error())
		return
	}

	outputDir := filepath.Join("results", domain)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		color.Red("Failed to create directory '%s': %v\n", outputDir, err)
		logError("Failed to create directory '" + outputDir + "': " + err.Error())
		return
	}
	color.Green("Directory '%s' created successfully.\n", outputDir)
	logInfo("Directory '" + outputDir + "' created successfully.")

	s := spinner.New(spinner.CharSets[36], 100*time.Millisecond)
	s.Prefix = "Fetching data from Wayback Machine "
	s.Start()

	apiURL := "https://web.archive.org/cdx/search/cdx"
	params := url.Values{}
	params.Add("url", "*." + domain + "/*")
	params.Add("collapse", "urlkey")
	params.Add("output", "text")
	params.Add("fl", "original")

	urls, err := fetchURLsConcurrently(apiURL, params, 4) // 4 workers
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

	// Catat waktu selesai program dan hitung durasi (hanya di log)
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Format durasi menjadi 2 angka di belakang koma
	formattedDuration := fmt.Sprintf("%.2f seconds", duration.Seconds())

	logInfo("Program ended at: " + endTime.Format("2006-01-02 15:04:05"))
	logInfo("Total duration: " + formattedDuration)

	// Opsional: Cetak durasi di konsol (jika diinginkan)
	color.Cyan("\nTotal duration: %s\n", formattedDuration)
}