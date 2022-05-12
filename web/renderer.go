package web

//from https://github.com/stnc/pongo4echo

import (
	"errors"
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	"io"
	"strings"
	"time"
)

type Pongo2Renderer struct {
	Debug bool
}

func (r Pongo2Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	var ctx pongo2.Context
	var ok bool
	if data != nil {
		ctx, ok = data.(pongo2.Context)
		if !ok {
			return errors.New("no pongo2.Context data was passed")
		}
	}

	var t *pongo2.Template
	var err error
	if r.Debug {
		t, err = pongo2.FromFile(name)
	} else {
		t, err = pongo2.FromCache(name)
	}
	if err != nil {
		return err
	}

	ctx["format_duration"] = formatDuration

	return t.ExecuteWriter(ctx, w)
}

///////////////////////////////////////////////////////////////////////////////

func formatDuration(d time.Duration) string {
	str := strings.Builder{}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		str.WriteString(fmt.Sprintf("%02dh ", hours))
	}
	if minutes > 0 || hours > 0 {
		str.WriteString(fmt.Sprintf("%02dm ", minutes))
	}
	str.WriteString(fmt.Sprintf("%02d.%03ds", seconds, d.Milliseconds()%1000))

	return str.String()
}
