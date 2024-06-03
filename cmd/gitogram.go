package main

import (
	"flag"

	"github.com/IlorDash/gitogram/internal/tui"
)

func main() {
	flag.Parse()
	tui.Run()
}
