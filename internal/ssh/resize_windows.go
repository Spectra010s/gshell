//go:build windows

package ssh

import "os"

// notifyResize is a no-op on Windows, which has no terminal resize signal.
func notifyResize(ch chan<- os.Signal) {}
