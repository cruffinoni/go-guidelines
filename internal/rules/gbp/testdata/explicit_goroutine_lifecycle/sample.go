package sample

import "sync"

func run() {
	go func() {}() // want "goroutine lifecycle"
}

func managed(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() { // want "goroutine lifecycle"
		defer wg.Done()
	}()
}

func other() {
	_ = 1
}
