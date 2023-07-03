package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	// Import local progressbar package
	"github.com/QuisVenator/ci-scraper/progressbar"
)

var (
	// This is an approximation of the range of ci available (starting point was manually tested)
	// startingCI = 6956795
	// endingCI   = 300000
	startingCI = 5708234
	endingCI   = 5708334

	// Other constants for scraping
	targetURL    = "https://santipresidente.com/padron-nacional.php"
	notFoundText = `<p class="fs-lg text-primary pb-lg-1 mb-4">El Número de CI <strong>9999999</strong> no fue encontrado</p>`
	re           = regexp.MustCompile(`<span class="text-dark fw-semibold me-1">(.*?)</span>`)
	writer       *csv.Writer

	// Used for ui
	progress progressbar.Progress
)

func main() {
	// Prepare CSV writer
	file, err := os.Create("results.csv")
	if err != nil {
		fmt.Printf("Error creating CSV file: %v\n", err)
		return
	}
	defer file.Close()

	writer = csv.NewWriter(file)
	defer writer.Flush()

	// Prepare tview app
	app := tview.NewApplication()
	textView := tview.NewTextView().
		SetChangedFunc(func() {
			app.Draw()
		})

	textView.SetBorder(true)
	textView.SetBackgroundColor(tcell.ColorDefault)

	progress = progressbar.Progress{TextView: textView}
	progress.Init(startingCI-endingCI, 50, "Scraper Progress: ")

	// Scrape all
	go scrape()

	// Start UI
	if err := app.SetRoot(textView, true).Run(); err != nil {
		panic(err)
	}
}

func scrape() {
	fmt.Println("Starting scrape...")
	for ci := startingCI; ci >= endingCI; ci-- {
		var body []byte
		for retries := 0; retries < 3; retries++ {
			resp, err := http.PostForm(targetURL, url.Values{"ci": {fmt.Sprint(ci)}})
			if err != nil {
				fmt.Printf("Error making POST request: %v\n", err)
				fmt.Println("Retrying in 1 second...")
				time.Sleep(1 * time.Second)
				continue
			}
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Printf("Error reading response body: %v\n", err)
				resp.Body.Close()
				fmt.Println("Retrying in 1 second...")
				time.Sleep(1 * time.Second)
				continue
			}
			resp.Body.Close()
			break
		}

		html := string(body)
		if !strings.Contains(html, notFoundText) {
			matches := re.FindAllStringSubmatch(html, -1)
			if matches != nil {
				data := []string{}
				for _, match := range matches {
					data = append(data, strings.TrimSuffix(match[1], "\n"))
				}
				// Writing the matched data to the CSV file
				err := writer.Write(data)
				if err != nil {
					progress.ErrorChan <- err
				}
			}
		}

		// Update UI
		progress.ProgChan <- 1
	}
}
