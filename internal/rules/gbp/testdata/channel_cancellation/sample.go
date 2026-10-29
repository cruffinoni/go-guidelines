package sample

func pipe(ch chan int) { // want "provide cancellation handling"
	<-ch
}

func pipeOK(ch chan int, done chan struct{}) {
	select {
	case <-ch:
	case <-done:
	}
}
