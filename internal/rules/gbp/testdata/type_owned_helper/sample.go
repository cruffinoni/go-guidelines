package sample

type Config struct{ limit int }

type Mode int

type Processor struct {
	config Config
	mode   Mode
}

type Worker struct{ config Config }

// Process delegates to helpers that are owned by Processor.
func (p *Processor) Process(data []byte) error {
	if err := validateConfig(p.config, 0); err != nil {
		return err
	}
	_ = data
	return resetState(p, true)
}

// Validate uses a helper whose only parameter is a local struct (GBP018).
func (p *Processor) Validate() error {
	return checkLimits(p.config)
}

// Normalize calls an independent transformation.
func (p *Processor) Normalize(data []byte) []byte {
	return normalizeWhitespace(data)
}

// Parse calls a helper whose ownership is ambiguous.
func (p *Processor) Parse(raw []byte) {
	_, _ = parseConfig(raw)
}

// ApplyMode passes a local defined scalar type.
func (p *Processor) ApplyMode() {
	_ = applyMode(p.mode, true)
}

// UseScalar passes a scalar receiver field to a utility.
func (p *Processor) UseScalar() {
	_ = scalarLimit(p.config.limit)
}

// UseMixed mixes receiver data with a method parameter.
func (p *Processor) UseMixed(raw []byte) {
	_ = mixedHelper(p.config, raw)
}

// Reload exercises a transitive helper chain.
func (p *Processor) Reload() {
	reload(p, true)
}

// Register passes a helper as a function value.
func (p *Processor) Register() {
	applyHelper(runReset)
}

// UseExported calls an exported helper.
func (p *Processor) UseExported() {
	ExportedHelper(p, 1)
}

// UseGeneric calls a generic helper.
func (p *Processor) UseGeneric() {
	genericHelper(p, 1)
}

// UseConstructor calls a constructor-shaped helper.
func (p *Processor) UseConstructor() {
	_ = newProcessor(p.config, 1)
}

// Share calls a helper that also serves another type.
func (p *Processor) Share() {
	shared(p, &Worker{})
}

// Share calls a helper that also serves Processor.
func (w *Worker) Share() {
	shared(&Processor{}, w)
}

// touch calls a helper that generated code also calls.
func (w *Worker) touch() {
	_ = runGeneratedTarget(w, 1)
}

// UseGeneratedCandidate calls a helper declared in a generated file.
func (p *Processor) UseGeneratedCandidate() {
	_ = generatedCandidate(&Worker{}, 1)
}

// validateConfig reads Config data passed from the receiver.
func validateConfig(c Config, minimum int) error { // want "only reachable from methods of Processor"
	if c.limit < minimum {
		return nil
	}
	return nil
}

// resetState reads and writes the receiver's state.
func resetState(p *Processor, force bool) error { // want "only reachable from methods of Processor"
	if force {
		p.config = Config{}
	}
	return nil
}

// checkLimits is GBP018's shape and is left to GBP018.
func checkLimits(c Config) error {
	return nil
}

// applyMode consumes a receiver-owned package type.
func applyMode(m Mode, force bool) error { // want "only reachable from methods of Processor"
	_ = force
	_ = m
	return nil
}

// scalarLimit is an independent transformation over a plain number.
func scalarLimit(limit int) int {
	return limit
}

// mixedHelper mixes receiver data with caller-owned data.
func mixedHelper(c Config, raw []byte) error {
	_, _ = c, raw
	return nil
}

// normalizeWhitespace is an independent transformation.
func normalizeWhitespace(data []byte) []byte {
	return data
}

// parseConfig has one owner path but no ownership evidence.
func parseConfig(raw []byte) (Config, error) {
	return Config{}, nil
}

// reload passes the owner type on to another helper.
func reload(p *Processor, eager bool) { // want "only reachable from methods of Processor"
	_ = eager
	loadConfig(p, 1)
}

// loadConfig consumes the owner state.
func loadConfig(p *Processor, n int) error { // want "only reachable from methods of Processor"
	_ = n
	_ = p.config
	return nil
}

// applyHelper stores a function value, so ownership is unknown.
func applyHelper(fn func(*Processor, bool) error) {
	_ = fn
}

// runReset is referenced only as a function value.
func runReset(p *Processor, force bool) error {
	_, _ = p, force
	return nil
}

// ExportedHelper is exported and therefore not a candidate.
func ExportedHelper(p *Processor, n int) {
	_, _ = p.config, n
}

// genericHelper is generic and therefore not a candidate.
func genericHelper[T any](p *Processor, v T) {
	_, _ = p.config, v
}

// newProcessor is a constructor and therefore not a candidate.
func newProcessor(c Config, n int) *Processor {
	_ = n
	return &Processor{config: c}
}

// shared serves two owner types.
func shared(p *Processor, w *Worker) {
	_, _ = p, w
}

// startWorker is referenced from a package-level variable.
func startWorker(w *Worker, n int) error {
	_, _ = w, n
	return nil
}

// runInit is referenced from an init function.
func runInit(p *Processor, n int) {
	_, _ = p, n
}

// runGeneratedTarget is referenced from production and generated code.
func runGeneratedTarget(w *Worker, n int) error {
	_, _ = w.config, n
	return nil
}

// testOnlyHelper is referenced only from tests.
func testOnlyHelper(c Config, n int) error {
	_, _ = c, n
	return nil
}

func init() {
	runInit(&Processor{}, 0)
}

var boot = startWorker(&Worker{}, 0)
