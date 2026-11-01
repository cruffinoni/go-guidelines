package sample

import (
	"context"
	"sync"
)

func pipe(ch chan int) { // want "provide cancellation handling"
	<-ch
}

func pipeOK(ch chan int, done chan struct{}) {
	select {
	case <-ch:
	case <-done:
	}
}

func waitGroupDone(ch chan int, wg *sync.WaitGroup) { // want "provide cancellation handling"
	defer wg.Done()
	ch <- 1
}

func contextDone(ctx context.Context, ch chan int) {
	<-ctx.Done()
	<-ch
}
