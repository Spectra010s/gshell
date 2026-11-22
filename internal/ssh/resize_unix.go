//go:build !windows

package ssh

import (
	"os"
	"os/signal"
	"syscall"
)

// notifyResize subscribes ch to terminal resize signals.
func notifyResize(ch chan<- os.Signal) {
	signal.Notify(ch, syscall.SIGWINCH)
}
