package sample

// GetName returns name.
func GetName() string { return "" } // want "documentation comment appears tautological and low-value"

// FetchName fetches the name from storage.
func FetchName() string { return "" }

// Count returns the count.
func Count() int { return 0 } // want "documentation comment appears tautological and low-value"

// T is a receiver type.
type T struct{}

// Value returns value.
func (T) Value() string { return "" } // want "documentation comment appears tautological and low-value"

// Compute runs a non-trivial calculation.
func (T) Compute() int { return 0 }

func (T) Undocumented() int { return 0 }
