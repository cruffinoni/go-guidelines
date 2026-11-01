package sample

import "fmt"

type Dep struct{}
type Thing struct{}

type Service struct {
	dep *Dep
}

func NewThing(dep *Dep) *Thing {
	if dep == nil { // want "required dependency"
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

func NewLocal() *Dep {
	var dep *Dep
	if dep == nil {
		return nil
	}
	return dep
}

func NewErrorName() error {
	var parseErr error
	if parseErr == nil {
		return nil
	}
	return parseErr
}

func NewReversed(dep *Dep) *Dep {
	if nil == dep { // want "required dependency"
		return nil
	}
	return dep
}

func NewErrorParam(parseErr error) *Thing {
	if parseErr == nil {
		return nil
	}
	return &Thing{}
}

func NewCompound(dep *Dep) *Thing {
	if dep == nil || false { // want "required dependency"
		return nil
	}
	return &Thing{}
}

func NewService(s *Service) *Service {
	if s.dep == nil { // want "required dependency"
		return nil
	}
	return s
}

func OpenService(dep *Dep) *Thing {
	if dep == nil { // want "required dependency"
		return nil
	}
	return &Thing{}
}

func BuildService(dep *Dep) *Thing {
	if dep == nil { // want "required dependency"
		return nil
	}
	return &Thing{}
}

func InitService(dep *Dep) *Thing {
	if dep == nil { // want "required dependency"
		return nil
	}
	return &Thing{}
}

func NewFailFast(dep *Dep) (*Thing, error) {
	if dep == nil {
		return nil, fmt.Errorf("dep is required")
	}
	return &Thing{}, nil
}

func NewNilError(dep *Dep) (*Thing, error) {
	if dep == nil { // want "required dependency"
		return nil, nil
	}
	return &Thing{}, nil
}

func NewPanic(dep *Dep) *Thing {
	if dep == nil {
		panic("dep is required")
	}
	return &Thing{}
}

func NewOptional(dep *Dep) *Thing {
	if dep != nil { // want "required dependency"
		return &Thing{}
	}
	return nil
}

func NewElseFail(dep *Dep) (*Thing, error) {
	if dep != nil {
		return &Thing{}, nil
	} else {
		return nil, fmt.Errorf("dep is required")
	}
}
