# GhostHunter
GhostHunter is a powerful and user-friendly tool designed to uncover hidden treasures from the Wayback Machine. It allows you to search for archived URLs (snapshots) of a specific domain, filter them by file extensions, and save the results in an organized manner. Whether you're a researcher, developer, or cybersecurity enthusiast, GhostHunter makes it easy to explore historical web data.

# Features
1. Domain Search
   - Search for all archived URLs of a specific domain from the Wayback Machine.
   - Automatically checks domain availability before starting the search.

2. File Extension Filtering
   - Filter URLs by specific file extensions (e.g., pdf, docx, xlsx, jpg).
   - Customize the list of extensions in the config.json file.

3. Concurrent URL Fetching
   - Fetch URLs concurrently using multiple workers for faster results.
   - Configurable number of workers for optimal performance.

4. Snapshot Finder
   - Find and display snapshots (archived versions) of the discovered URLs.
   - Timestamps are displayed in a human-readable format (e.g., 23 January 2025, 15:46:09).

5. Organized Results
    - Save filtered URLs into separate files based on their extensions (e.g., example.com.pdf.txt, example.com.docx.txt).
    - Save snapshot results into a single file for easy reference.

6. Colorful and User-Friendly Interface
    - Use of colors and tables for a visually appealing and easy-to-read output.
    - Summary tables provide a quick overview of the results.

7. Internet and Wayback Machine Status Check
   - Automatically checks for an active internet connection and Wayback Machine availability before proceeding.

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

# User Interface
<img src="https://github.com/user-attachments/assets/dae9e3ac-9948-4895-bd32-75ecc0145101" alt="GhostHunter Logo" width="600">
<img src="https://github.com/user-attachments/assets/6302d388-f745-4eda-8fc9-dcad9d02d974" alt="GhostHunter Logo" width="600">
<img src="https://github.com/user-attachments/assets/87a60b32-19bc-45ba-a6fd-09a9ab7df5a5" alt="GhostHunter Logo" width="600">

# Contributing
Contributions are welcome! If you have any suggestions, bug reports, or feature requests, please open an issue or submit a pull request.

# Acknowledgments
Thanks to the [Wayback Machine](https://web.archive.org/) for providing the CDX API.

Special thanks to the creators of the Go libraries used in this project:
- [briandowns/spinner](https://github.com/briandowns/spinner)
- [fatih/color](https://github.com/fatih/color)
- [olekukonko/tablewriter](https://github.com/olekukonko/tablewriter)
