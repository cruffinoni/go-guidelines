package sample

// Documented explains itself.
type Documented struct{}

type Undocumented struct{} // want "exported type is missing a documentation comment"

// Grouped covers the doc-on-group case.
type (
	Grouped struct{}
)

type unexported struct{}

// ServerConfig is left to GCM004.
type ServerConfig struct{}

type UndocumentedConfig struct{}
