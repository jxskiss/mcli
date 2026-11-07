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
	app := NewApp()
	app.getFlagSet().SetOutput(terminal)
	for _, tt := range []struct {
		columns uint16
		want    int
	}{{0, 80}, {10, 30}, {20, 30}, {29, 30}, {30, 30}, {40, 40}, {52, 52}} {
		if err := unix.IoctlSetWinsize(int(terminal.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 24, Col: tt.columns}); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, tt.want, newUsagePrinter(app).width)
	}
	app.HelpWidth = 20
	assert.Equal(t, 30, newUsagePrinter(app).width)
	app.HelpWidth = 100
	assert.Equal(t, 100, newUsagePrinter(app).width)
}
