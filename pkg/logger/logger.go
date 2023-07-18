package logger

import (
	"fmt"
	"os"
)

func Info(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}
