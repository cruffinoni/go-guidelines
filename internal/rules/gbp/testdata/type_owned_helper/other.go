package sample

// Other adds a second call site from another file.
func (p *Processor) Other() {
	_ = validateConfig(p.config, 2)
}
