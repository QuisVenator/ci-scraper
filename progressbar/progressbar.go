package progressbar

import (
	"fmt"
	"strings"

	"github.com/rivo/tview"
)

type Progress struct {
	TextView     *tview.TextView
	leadText     string
	full         int
	limit        int
	ProgChan     chan int
	ErrorChan    chan error
	NotFoundChan chan struct{}
}

func (p *Progress) Init(full, limit int, prompt string) chan int {
	p.ProgChan = make(chan int)
	p.ErrorChan = make(chan error)
	p.NotFoundChan = make(chan struct{})
	p.full = full
	p.limit = limit

	if prompt == "" {
		prompt = "Progress: "
	}
	p.leadText = prompt

	go func() {
		progress := 0
		errors := 0
		notfound := 0
		for {
			select {
			case prog := <-p.ProgChan:
				progress += prog
			case <-p.ErrorChan:
				errors += 1
			case <-p.NotFoundChan:
				notfound += 1
				progress += 1
			}

			if progress > full {
				break
			}

			x := progress * limit / full
			p.TextView.Clear()
			fmt.Fprintf(p.TextView, "%s%s%s %d/%d",
				p.leadText,
				strings.Repeat("■", x),
				strings.Repeat("□", limit-x),
				progress, full)
			fmt.Fprintf(p.TextView, "\nNot found: [yellow]%d[white]\n", notfound)
			fmt.Fprintf(p.TextView, "\nError count: [red]%d[white]\n", errors)
		}
	}()

	return p.ProgChan
}
