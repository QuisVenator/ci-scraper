package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	// Import local progressbar package
	"github.com/QuisVenator/ci-scraper/progressbar"
)

var (
	// This is an approximation of the range of ci available (starting point was manually tested)
	startingCI = 5708234
	endingCI   = 5708234
	// startingCI = 5708334
	// endingCI   = 5708234

	// Other constants for scraping
	targetURL    = "https://servicios.ips.gov.py/constancias_aop/controladores/funcionesConstanciasAsegurados.php?opcion=consultarAsegurado"
	notFoundText = `El Nro de CIC no existe en la base de datos local de la Policía`
	writer       *csv.Writer

	// Used for ui
	progress progressbar.Progress

	// Channel for stopping the scraper
	stopChan chan struct{}
)

type Data struct {
	CI          string
	Name        string
	Nationality string
	Status      string
}

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

	// Prepare stop channel
	stopChan = make(chan struct{})

	// Prepare tview app
	app := tview.NewApplication()
	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q':
				stopChan <- struct{}{}
				app.Stop()
			}
		}
		return event
	})

	textView.SetBorder(true)
	textView.SetBackgroundColor(tcell.ColorDefault)

	progress = progressbar.Progress{TextView: textView}
	progress.Init(startingCI-endingCI+1, 50, "Scraper Progress (press q to stop): ")

	// Scrape all
	go scrape()

	// Start UI
	if err := app.SetRoot(textView, true).Run(); err != nil {
		panic(err)
	}
}

func scrape() {
	for ci := startingCI; ci >= endingCI; ci-- {
		var body []byte
		for retries := 0; retries < 3; retries++ {
			resp, err := http.PostForm(targetURL, url.Values{"parmDocOrigen": {"226"}, "parmCedula": {fmt.Sprintf("%d", ci)}})
			if err != nil {
				progress.ErrorChan <- err
				time.Sleep(1 * time.Second)
				continue
			}
			body, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				progress.ErrorChan <- err
				time.Sleep(1 * time.Second)
				continue
			}
			resp.Body.Close()
			break
		}

		html := string(body)
		if strings.Contains(html, notFoundText) {
			// fmt.Printf("CI %d not found\n", ci)
		} else {
			// Use goquery to parse the html
			doc, err := goquery.NewDocumentFromReader(bytes.NewBufferString(html))
			if err != nil {
				progress.ErrorChan <- err
			} else {
				// Initialize a Data instance
				var data Data

				// Extract each field based on its ID
				data.CI = strings.TrimSpace(doc.Find("#varCedula").AttrOr("value", ""))
				data.Name = strings.TrimSpace(doc.Find("#varNombre").AttrOr("value", ""))
				data.Nationality = strings.TrimSpace(doc.Find("#varNacionalidad").AttrOr("value", ""))
				data.Status = strings.TrimSpace(doc.Find("#varEstado").AttrOr("value", ""))

				// Write to CSV
				writer.Write([]string{data.CI, data.Name, data.Nationality, data.Status})
			}
		}

		// Update UI
		progress.ProgChan <- 1

		// // DEBUG code
		// time.Sleep(10 * time.Millisecond)
		// progress.ProgChan <- 1

		// If stop signal is received, stop scraping
		select {
		case <-stopChan:
			return
		default:
			// Do nothing
		}
	}
	<-stopChan
}
