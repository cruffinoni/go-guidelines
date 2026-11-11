package sample

import "testing"

func TestProcessorHelpers(t *testing.T) {
	_ = validateConfig(Config{}, 0)
	_ = testHelperOnly(Config{}, 1)
}

// testHook would make testOnlyHelper look owned if test references counted.
func (p *Processor) testHook() {
	_ = testOnlyHelper(p.config, 1)
}

// testHelperOnly is declared in a test file and is never reported.
func testHelperOnly(c Config, n int) error {
	_, _ = c, n
	return nil
}
