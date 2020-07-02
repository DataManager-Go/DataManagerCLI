package commands

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

// BarTask for the bar to do
type BarTask uint8

// ...
const (
	UploadTask BarTask = iota
	DownloadTask
)

// Implement string
func (bt BarTask) String() string {
	switch bt {
	case UploadTask:
		return "Upload"
	case DownloadTask:
		return "Download"
	}

	return ""
}

// Verb return task as verb
func (bt BarTask) Verb() string {
	return bt.String() + "ing"
}

// Bar a porgressbar
type Bar struct {
	task    BarTask
	total   int64
	options []mpb.BarOption
	style   string

	bar *mpb.Bar

	doneTextChan chan string
	doneText     string
	done         bool

	outWriter io.Writer
}

// NewBar create a new bar
func NewBar(task BarTask, total int64, name string, singleMode bool) *Bar {
	// Create bar instance
	bar := &Bar{
		task:         task,
		total:        total,
		style:        "(=>_)",
		doneTextChan: make(chan string, 1),
	}

	// Trim text if its too long
	name = trimName(name, 40)

	bar.options = make([]mpb.BarOption, 1)

	// Singlemode true => Only one file was/is uploaded/uploading
	if singleMode {
		// Print fileinfo over multiple lines after uploading
		bar.options[0] = mpb.BarExtender(mpb.BarFillerFunc(func(w io.Writer, reqWidth int, stat decor.Statistics) {
			if stat.Completed {
				if bar.done {
					// Restore doneText to prevent blocking
					// from doneTextChan
					fmt.Fprint(w, bar.doneText)
					return
				}

				// Wait for file informations
				bar.doneText = <-bar.doneTextChan

				// Print file informations
				fmt.Fprint(w, bar.doneText)

				bar.done = true
			}
		}))
		bar.options = append(bar.options, mpb.BarFillerClearOnComplete())
	} else {
		// Middleware for printing singlelined file info
		bar.options[0] = mpb.BarFillerMiddleware(func(base mpb.BarFiller) mpb.BarFiller {
			return mpb.BarFillerFunc(func(w io.Writer, reqWidth int, st decor.Statistics) {
				if st.Completed {
					text := <-bar.doneTextChan
					bar.doneTextChan <- text
					io.WriteString(w, text)
					bar.done = true
				} else {
					base.Fill(w, reqWidth, st)
				}
			})
		})
	}

	// Decorate Bar
	bar.options = append(bar.options, []mpb.BarOption{
		mpb.PrependDecorators(
			decor.OnComplete(decor.Spinner([]string{" ⠋ ", " ⠙ ", " ⠹ ", " ⠸ ", " ⠼ ", " ⠴ ", " ⠦ ", " ⠧ ", " ⠇ ", " ⠏ "}), ""),
			decor.OnComplete(decor.Name(task.Verb()), "Done!"),
			decor.OnComplete(decor.Name(" '"+name+"'"), ""),
		),
		mpb.AppendDecorators(
			decor.OnComplete(decor.Percentage(decor.WCSyncWidth), ""),
			decor.OnComplete(decor.CountersKiloByte(" [%d / %d]", decor.WCSyncWidth), ""),
		),
	}...)

	return bar
}

// Stop the bar
func (bar *Bar) stop(text ...string) {
	if bar == nil {
		return
	}

	// Write into the textChan to prevent
	// it from blocking
	if bar.doneTextChan != nil {
		go func() {
			if len(text) > 0 {
				bar.doneTextChan <- text[0]
			} else {
				bar.doneTextChan <- "stopped"
			}
		}()
	}

	// Set the bar to a finished state
	// to ensure it won't block anything
	bar.bar.SetTotal(bar.total, true)
}

// ProgressView holds info for progress
type ProgressView struct {
	ProgressContainer *mpb.Progress
	Bars              []*mpb.Bar
	RawBars           []*Bar
}

// AddBar to ProgressView
func (pv *ProgressView) AddBar(bbar *Bar) *mpb.Bar {
	// Add bar to render queue
	bar := pv.ProgressContainer.Add(bbar.total, mpb.NewBarFiller(bbar.style, false), bbar.options...)

	// Set Bars mpb.Bar to allow it
	// to increase
	bbar.bar = bar

	// Append bar to pv bars
	pv.Bars = append(pv.Bars, bar)
	pv.RawBars = append(pv.RawBars, bbar)

	// Return prepared proxy func
	return bar
}

// NewProgressView create new progressview
func NewProgressView() *ProgressView {
	return &ProgressView{
		Bars: []*mpb.Bar{},
		ProgressContainer: mpb.New(
			mpb.WithWaitGroup(&sync.WaitGroup{}),
			mpb.WithRefreshRate(50*time.Millisecond),
			mpb.WithWidth(130),
		),
	}
}

type proxyWriter struct {
	bar *Bar
	w   io.Writer
}

func (proxy proxyWriter) Write(b []byte) (int, error) {
	n, err := proxy.w.Write(b)

	go proxy.bar.bar.IncrBy(n)

	return n, err
}
