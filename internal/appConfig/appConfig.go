package appConfig

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
)

var Debug bool

func init() {
	flag.BoolVar(&Debug, "debug", false, "Enable debug mode")

}

func LogErr(err error, format string, a ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	prefix := fmt.Sprintf("%s:%d ERROR: ", file, line)
	log.Println(prefix+fmt.Sprintf(format, a...), err)
}

func LogDebug(format string, a ...interface{}) {
	if !Debug {
		return
	}

	_, file, line, ok := runtime.Caller(1)
	if !ok {
		file = "unknown"
		line = 0
	} else {
		file = filepath.Base(file)
	}

	prefix := fmt.Sprintf("%s:%d DEBUG: ", file, line)
	log.Println(prefix + fmt.Sprintf(format, a...))
}
