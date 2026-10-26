package sample

type T struct {
	Items   []string `json:"items"` // want "consider omitempty"
	Good    []string `json:"good,omitempty"`
	Ignored []string `json:"-"`
	Name    string   `json:"name"`
}
