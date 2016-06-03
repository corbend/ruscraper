package scheduler

import (
	"fmt"
	"time"
)

func RunTimer(timeout int, runCallback func()) {

	ticker := time.NewTicker(time.Duration(timeout) * 1000 * time.Millisecond)
	go func() {
	    for t := range ticker.C {
            fmt.Println("Start Parse ->", t)
            runCallback()
        }
    }()
}