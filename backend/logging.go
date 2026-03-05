package main

import (
	"fmt"
	"os"
	"strings"
)

var verboseLogs bool

func init() {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("MBTA_VERBOSE_LOGS")))
	verboseLogs = v == "1" || v == "true" || v == "yes" || v == "debug"
}

func debugf(format string, args ...interface{}) {
	if verboseLogs {
		fmt.Printf(format, args...)
	}
}

func debugln(args ...interface{}) {
	if verboseLogs {
		fmt.Println(args...)
	}
}
