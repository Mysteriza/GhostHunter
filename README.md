# GhostHunter
GhostHunter is a powerful web scraping tool designed to discover archived files from a target domain using the Wayback Machine CDX API. It is particularly useful for penetration testers and security researchers to identify potentially sensitive files exposed on a website.

# Features
- Domain Search: Search for archived files from a specific domain.
- File Type Filtering: Automatically filters and categorizes files by their extensions (e.g., .sql, .pdf, .log).
- Efficient Storage: Saves results in a structured directory named after the target domain.
- User-Friendly Interface: Includes a spinner and colored output for better user experience.

# Installation
Prerequisites
Go: Make sure you have Go installed on your system.

## Steps
1. Clone the repository:
   ```
   git clone https://github.com/mysteriza/GhostHunter.git
   ```
   ```
   cd GhostHunter
   ```
2. Install Dependencies:
   
   GhostHunter uses the following Go packages:
   - github.com/briandowns/spinner
   - github.com/fatih/color
   - github.com/olekukonko/tablewriter

   Install them using:
    ```
    go get github.com/briandowns/spinner
    ```
    ```
    go get github.com/fatih/color
    ```
    ```
    go get github.com/olekukonko/tablewriter
    ```

    Or install them globally:
    ```
    go install github.com/briandowns/spinner@latest
    ```
    ```
    go install github.com/fatih/color@latest
    ```
    ```
    go install github.com/olekukonko/tablewriter@latest
    ```

3. Usage:   
   Run the Tool (Faster Way):
   ```
   ./GhostHunter
   ```
   
   Or run directly without using the build file:
   ```
   go run scanner.go
   ```

   You want to rebuild the file? Use:
   ```
   go build -o GhostHunter
   ```

# Contributing
Contributions are welcome! If you have any suggestions, bug reports, or feature requests, please open an issue or submit a pull request.

# Acknowledgments
Thanks to the [Wayback Machine](https://web.archive.org/) for providing the CDX API.

Special thanks to the creators of the Go libraries used in this project:
- [briandowns/spinner](https://github.com/briandowns/spinner)
- [fatih/color](https://github.com/fatih/color)
- [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter)
