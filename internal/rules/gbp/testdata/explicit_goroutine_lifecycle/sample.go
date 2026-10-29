package sample

func run() {
	go func() {}() // want "goroutine lifecycle"
}

func other() {
	_ = 1
}
