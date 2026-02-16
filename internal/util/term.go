package util

import (
	"fmt"
	"os"
)

func ClearScreen() {
	_, _ = fmt.Fprint(os.Stdout, "\033c")
}
