package commands

import (
	"fmt"
	"io"

	"github.com/gosuri/uiprogress"
	"github.com/gosuri/uiprogress/util/strutil"
	"github.com/sbani/go-humanizer/units"
)

type barProxyData struct {
	bytesWritten int
}

// Bar proxy a proxywriter for progressbars
type barProxy struct {
	data *barProxyData
	bar  *uiprogress.Bar
	w    io.Writer
	r    io.Reader
}

// Implement io.Writer to barProxy
func (p barProxy) Write(b []byte) (int, error) {
	size := len(b)
	p.bar.Incr(size)
	p.data.bytesWritten += size
	return p.w.Write(b)
}

// Create a proxy for bar
func barProxyFromBar(bar *uiprogress.Bar, w io.Writer) *barProxy {
	proxy := &barProxy{
		data: &barProxyData{},
		bar:  bar,
		w:    w,
	}

	bar.Data = proxy
	return proxy
}

// Build a progressbar and a proxy for it
func buildProgressbar(prefix string, len uint) (*uiprogress.Bar, func(io.Writer) io.Writer) {
	// Create bar
	bar := uiprogress.NewBar(0).PrependCompleted()

	// Prepend prefix
	if prefix != "" && len > 0 {
		bar.PrependFunc(func(b *uiprogress.Bar) string {
			return strutil.Resize(prefix, len)
		})
	}

	// Append amount
	bar.AppendFunc(func(b *uiprogress.Bar) string {
		if proxy, ok := (b.Data).(*barProxy); ok {
			_ = proxy
			return fmt.Sprintf("[%s/%s]", units.BinarySuffix(float64(proxy.data.bytesWritten)), units.BinarySuffix(float64(b.Total)))
		}

		return ""
	})

	// Set custom bar style
	bar.LeftEnd = '('
	bar.RightEnd = ')'
	bar.Empty = '_'

	// Create proxy
	proxy := func(w io.Writer) io.Writer {
		return barProxyFromBar(bar, w)
	}

	return bar, proxy
}
