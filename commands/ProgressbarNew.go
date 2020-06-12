package commands

import (
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
}

// NewBar create a new bar
func NewBar(task BarTask, total int64, name string) *Bar {
	// Create bar instance
	bar := &Bar{
		task:         task,
		total:        total,
		style:        "(=>_)",
		doneTextChan: make(chan string, 1),
	}

	// Trim name
	if len(name) > 40 {
		name = name[:20] + "..." + name[len(name)-20:]
	}

	// Add Bar options
	bar.options = []mpb.BarOption{
		mpb.BarFillerMiddleware(func(base mpb.BarFiller) mpb.BarFiller {
			return mpb.BarFillerFunc(func(w io.Writer, reqWidth int, st decor.Statistics) {
				if bar.done {
					io.WriteString(w, bar.doneText)
					return
				}

				// Check if there is text in the doneText channel
				select {
				case text := <-bar.doneTextChan:
					bar.doneText = text
					bar.done = true
					io.WriteString(w, text)
					return
				default:
				}

				base.Fill(w, reqWidth, st)
			})
		}),
	}

	bar.options = append(bar.options, []mpb.BarOption{
		mpb.PrependDecorators(
			decor.OnComplete(decor.Spinner(nil, decor.WCSyncSpace), "done"),
			decor.Name(task.Verb(), decor.WCSyncSpace),
			decor.Name(" '"+name+"'", decor.WCSyncSpaceR),
			decor.Percentage(decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			decor.CountersKiloByte("[%d / %d]", decor.WCSyncWidth),
		),
	}...)

	return bar
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
