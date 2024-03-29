package timer

import (
	"log"
	"time"
)

// var l = log.New(os.Stdout, "timer ", log.LstdFlags)

func StartTimer(name string, logger *log.Logger) func() {
	t := time.Now()
	logger.Println(name, "started")

	return func() {
		// d := time.Now().Sub(t)
		d := time.Since(t)
		logger.Println(name, "took", d)
	}

}
