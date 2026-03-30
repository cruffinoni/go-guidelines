package pipeline

func Drain(ch <-chan string, done <-chan bool) {
	for v := range ch { _ = <-done; _ = v }
}

func Produce(out chan<- string) {
	out <- "item"
}
