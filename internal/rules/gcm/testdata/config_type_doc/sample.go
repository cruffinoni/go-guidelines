package sample

// ServerConfig documents itself.
type ServerConfig struct{}

type ClientConfig struct{} // want "exported type is missing a documentation comment"

// Grouped covers the doc-on-group case.
type (
	PoolConfig struct{}
)

type plainConfig struct{}

// Widget is left to GCM001.
type Widget struct{}

type Undocumented struct{}
