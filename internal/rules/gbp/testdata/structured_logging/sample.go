package sample

import (
	"log"
	"log/slog"
)

func f() {
	log.Println("x") // want "structured logging"
	log.Printf("x")  // want "structured logging"
	log.Fatal("x")   // want "structured logging"
	slog.Info("x")
}
