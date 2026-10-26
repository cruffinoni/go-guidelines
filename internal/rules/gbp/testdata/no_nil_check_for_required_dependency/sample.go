package sample

type Dep struct{}
type Thing struct{}

func NewThing(dep *Dep) *Thing {
	if dep == nil { // want "nil-checks a required dependency"
		return nil
	}
	return &Thing{}
}

func NewOK(err error) *Thing {
	if err == nil {
		return &Thing{}
	}
	return nil
}

func notNew(dep *Dep) {
	if dep == nil {
	}
}
