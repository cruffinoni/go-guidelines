package sample

import "context"

type Service struct {
	ctx context.Context // want "do not store context.Context"
}

func Good(ctx context.Context, x int) {}

func Bad(x int, ctx context.Context) {} // want "should be the first parameter"
