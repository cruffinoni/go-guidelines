package util

type Dep struct{}

type Service struct {
	dep *Dep
}

func NewService(dep *Dep) *Service {
	if dep == nil {
		return nil
	}
	return &Service{dep: dep}
}
