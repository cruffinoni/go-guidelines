//go:build windows

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportPathCrossVolume(t *testing.T) {
	assert.Equal(t, "D:/repo/a.go", reportPath(`C:\work`, `D:\repo\a.go`))
}
