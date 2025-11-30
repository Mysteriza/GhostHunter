package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
)

// filterURLs filters URLs based on specified extensions using a channel
func filterURLs(input <-chan string, extensions []string) []string {
	regexPattern := `\.(` + strings.Join(extensions, "|") + `)(\?.*)?$`
	re := regexp.MustCompile(regexPattern)

	var filteredURLs []string
	for line := range input {
		if line != "" && re.MatchString(line) {
			filteredURLs = append(filteredURLs, line)
		}
	}

	return filteredURLs
}

// saveResultsByExtension saves filtered URLs to files grouped by extension
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

			// Use buffered writing
			file, err := os.Create(filePath)
			if err != nil {
				mu.Lock()
				table.Append([]string{ext, fileName, color.RedString("Failed"), fmt.Sprintf("%d URLs", len(urls))})
				mu.Unlock()
				return
			}
			defer file.Close()

			writer := bufio.NewWriter(file)
			for _, url := range urls {
				fmt.Fprintln(writer, url)
			}

			if err := writer.Flush(); err != nil {
				mu.Lock()
				table.Append([]string{ext, fileName, color.RedString("Failed"), fmt.Sprintf("%d URLs", len(urls))})
				mu.Unlock()
			} else {
				mu.Lock()
				table.Append([]string{ext, fileName, color.GreenString("Success"), fmt.Sprintf("%d URLs", len(urls))})
				mu.Unlock()
			}

			mu.Lock()
			totalURLs += len(urls)
			mu.Unlock()
		}(ext, urls)
	}
	wg.Wait()

	table.Append([]string{"", "", "", ""})
	table.Append([]string{"", "", "-------------------", "-------------------"})
	table.Append([]string{"", "", "TOTAL", fmt.Sprintf("%d URLs", totalURLs)})

	fmt.Println("\nResults Summary:")
	table.Render()
}
