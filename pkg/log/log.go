package log

import (
    "fmt"
)

func Info(format string, args ...any) {
    fmt.Printf(format+"\n", args...)
}

