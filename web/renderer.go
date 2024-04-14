package web

//from https://github.com/stnc/pongo4echo

import (
	"errors"
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"io"
	"strings"
	"time"
)

const MaxOutputLength = 120

type Pongo2Renderer struct {
	Debug bool
}

func init() {
	if err := pongo2.RegisterFilter("format_duration", formatDuration); err != nil {
		log.Errorf("unable to add filter format_duration: %s", err)
	}
	if err := pongo2.RegisterFilter("split_output", splitOutput); err != nil {
		log.Errorf("unable to add filter split_output: %s", err)
	}
}

func (r Pongo2Renderer) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
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

	return t.ExecuteWriter(ctx, w)
}

///////////////////////////////////////////////////////////////////////////////

func formatDuration(in *pongo2.Value, _ *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if !in.IsInteger() {
		return pongo2.AsValue("ERR DURATION"), nil
	}

	d := time.Duration(in.Integer())

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

	return pongo2.AsValue(str.String()), nil
}

func splitOutput(in *pongo2.Value, _ *pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	if !in.IsString() {
		return pongo2.AsValue("ERR STR"), nil
	}

	lines := strings.Split(in.String(), "\n")
	sb := strings.Builder{}

	for lineIdx, line := range lines {
		if lineIdx > 0 {
			sb.WriteRune('\n')
		}

		r := []rune(line)
		rlen := len(r)

		for left := 0; left < rlen; left += MaxOutputLength {
			if left > 0 {
				sb.WriteString("\\\n")
			}
			right := left + MaxOutputLength
			if right > rlen {
				right = rlen
			}
			sb.WriteString(string(r[left:right]))
		}
	}

	return pongo2.AsValue(sb.String()), nil
}
