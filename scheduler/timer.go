package scheduler

import (
	"fmt"
	"time"
)

func RunTimer(timeout int, runCallback func()) {

	ticker := time.NewTicker(time.Millisecond * 10000)
	go func() {
	    for t := range ticker.C {
            fmt.Println("Start Parse ->", t)
            runCallback()
        }
    }()
}