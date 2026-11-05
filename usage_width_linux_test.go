//go:build linux

package mcli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/unix"
)

func TestHelpWidthOutputTerminal(t *testing.T) {
	terminal, err := os.OpenFile("/dev/ptmx", os.O_RDWR|unix.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("pseudo-terminal unavailable: %v", err)
	}
	defer terminal.Close()
	if err := unix.IoctlSetWinsize(int(terminal.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: 52}); err != nil {
		t.Fatal(err)
	}
	app := NewApp()
	app.getFlagSet().SetOutput(terminal)
	assert.Equal(t, 52, newUsagePrinter(app).width)
	app.HelpWidth = 100
	assert.Equal(t, 100, newUsagePrinter(app).width)
}
