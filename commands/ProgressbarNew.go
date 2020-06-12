package commands

import (
	"io"
	"sync"
	"time"

	"github.com/vbauerster/mpb/v5"
	"github.com/vbauerster/mpb/v5/decor"
)

// ProgressView holds info for progress
type ProgressView struct {
	ProgressContainer *mpb.Progress
	Bars              []*mpb.Bar
}

type barData struct {
	// Offset represents n bytes which
	// are written to the server but not
	// to the progressbar
	offset int
}

// Bar a porgressbar
type Bar struct {
	total   int64
	options []mpb.BarOption
	style   string

	bar *mpb.Bar

	// Original writer for the proxy
	ow io.Writer

	// Data required for the proxy
	barData *barData
}

func (bar Bar) Write(b []byte) (int, error) {
	n, err := bar.ow.Write(b)

	// if bar is set, write to it
	if bar.bar != nil {
		// If cached writtenBytes are
		// not restored yet, restore them
		if bar.barData.offset > 0 {
			bar.bar.IncrBy(bar.barData.offset)
			bar.barData.offset = 0
		}

		bar.bar.IncrBy(n)
	} else {
		// If bar is not visible yet,
		// cache written bytes
		bar.barData.offset += n
	}

	return n, err
}

// NewProgressView create new progressview
func NewProgressView() *ProgressView {
	return &ProgressView{
		Bars: []*mpb.Bar{},
		ProgressContainer: mpb.New(
			mpb.WithWaitGroup(&sync.WaitGroup{}),
			mpb.WithRefreshRate(50*time.Millisecond),
			mpb.WithWidth(100),
		),
	}
}

// NewBar create a new bar
func NewBar(total int64, name string) *Bar {
	// Create bar instance
	bar := &Bar{
		total:   total,
		style:   mpb.DefaultBarStyle,
		barData: &barData{},
	}

	// Add Bar options
	bar.options = []mpb.BarOption{
		mpb.PrependDecorators(
			decor.Name(name),
			decor.Elapsed(decor.ET_STYLE_GO, decor.WCSyncSpace),
		),
	}

	return bar
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

	// Return prepared proxy func
	return bar
}

// ProgressBarProxy a proxywriter for progressbars
type ProgressBarProxy struct {
	bar *mpb.Bar
	w   io.Writer
}
