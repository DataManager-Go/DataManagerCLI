package commands

import (
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

// AddBar to ProgressView
func (pv *ProgressView) AddBar(total int64, name string) *mpb.Bar {
	// Create nev bar
	bar := pv.ProgressContainer.AddBar(int64(total),
		mpb.PrependDecorators(
			decor.Name(name),
			decor.Elapsed(decor.ET_STYLE_GO, decor.WCSyncSpace),
		),
		mpb.AppendDecorators(
			decor.OnComplete(
				decor.Percentage(decor.WC{W: 5}), "done",
			),
		),
	)

	// Append bar to pv bars
	pv.Bars = append(pv.Bars, bar)

	return bar
}
