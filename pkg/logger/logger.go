package logger

import (
	"fmt"
	"os"

	"github.com/busser/tfautomv/internal/format"
)

func Info(msg string) {
	fmt.Fprint(os.Stderr, format.Info(msg))
}
