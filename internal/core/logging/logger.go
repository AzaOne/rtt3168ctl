package logging

import (
	"io"
	"log"
)

func New(output io.Writer) *log.Logger {
	return log.New(output, "rtt3168ctl: ", log.LstdFlags)
}
