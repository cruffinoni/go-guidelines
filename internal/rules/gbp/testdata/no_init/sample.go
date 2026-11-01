package sample

func init() { // want "avoid init; prefer explicit setup"
}

func Init() {
}

type T struct{}

func (T) init() {
}
