package sysutil

import (
	"testing"
	"time"
)

// TestBootTime tests the boot time.
func TestBootTime(t *testing.T) {
	t.Logf("boot time: %s", BootTime().Format(time.RFC3339Nano))
	t.Logf("now:       %s", time.Now().Format(time.RFC3339Nano))
}
