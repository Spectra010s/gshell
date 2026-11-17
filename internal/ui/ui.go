// Package ui provides consistent terminal output for gshell.
//
// Command results are written to Out so stdout stays clean when piped.
// Progress and diagnostics are written to Err.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// Options configures a UI.
type Options struct {
	Out     io.Writer
	Err     io.Writer
	Verbose bool
	NoColor bool
}

// UI writes human-facing output.
type UI struct {
	out, err       io.Writer
	verbose        bool
	outTTY, errTTY bool
	color          bool
	mu             sync.Mutex
}

// New returns a UI writing results to Out and progress to Err.
func New(opts Options) *UI {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}

	color := !opts.NoColor &&
		os.Getenv("NO_COLOR") == "" &&
		os.Getenv("TERM") != "dumb"

	return &UI{
		out:     opts.Out,
		err:     opts.Err,
		verbose: opts.Verbose,
		outTTY:  isTerminal(opts.Out),
		errTTY:  isTerminal(opts.Err),
		color:   color,
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

// Printf writes a command result to stdout.
func (u *UI) Printf(format string, a ...any) {
	u.write(u.out, format, a...)
}

// Successf writes a success message to stdout.
func (u *UI) Successf(format string, a ...any) {
	u.write(u.out, "%s %s\n", u.paint("32", u.outTTY, "✓"), fmt.Sprintf(format, a...))
}

// Infof writes informational output to stderr.
func (u *UI) Infof(format string, a ...any) {
	u.write(u.err, format+"\n", a...)
}

// Verbosef writes extra detail to stderr when --verbose is set.
func (u *UI) Verbosef(format string, a ...any) {
	if !u.verbose {
		return
	}
	u.write(u.err, format+"\n", a...)
}

// Warnf writes a warning to stderr.
func (u *UI) Warnf(format string, a ...any) {
	u.write(u.err, "%s %s\n", u.paint("33", u.errTTY, "!"), fmt.Sprintf(format, a...))
}

// Errorf writes an error message to stderr.
func (u *UI) Errorf(format string, a ...any) {
	u.write(u.err, "%s %s\n", u.paint("31", u.errTTY, "✗"), fmt.Sprintf(format, a...))
}

// Prompt writes a label to stderr and reads one line from stdin.
func (u *UI) Prompt(label string) (string, error) {
	u.write(u.err, "%s", label)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (u *UI) write(w io.Writer, format string, a ...any) {
	u.mu.Lock()
	defer u.mu.Unlock()
	fmt.Fprintf(w, format, a...)
}

func (u *UI) paint(code string, enabled bool, s string) string {
	if !u.color || !enabled {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// Step tracks a long-running operation and renders it on stderr.
type Step struct {
	ui    *UI
	msg   string
	start time.Time
	stop  chan struct{}
	done  chan struct{}
	once  sync.Once
}

var spinFrames = []string{"⠏", "⠛", "⠹", "⠼", "⠶", "⠧"}

// Step starts a progress indicator. On a terminal it animates in place; on a
// non-interactive stream it prints the message once and a result when finished.
func (u *UI) Step(msg string) *Step {
	s := &Step{
		ui:    u,
		msg:   msg,
		start: time.Now(),
		stop:  make(chan struct{}),
		done:  make(chan struct{}),
	}

	if !u.errTTY {
		u.write(u.err, "%s...\n", msg)
		close(s.done)
		return s
	}

	u.write(u.err, "  %s %s...", spinFrames[0], msg)
	go s.spin()
	return s
}

func (s *Step) spin() {
	defer close(s.done)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for i := 1; ; i++ {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.ui.write(s.ui.err, "\r\x1b[K  %s %s...", spinFrames[i%len(spinFrames)], s.msg)
		}
	}
}

// Done marks the step as successfully completed.
func (s *Step) Done() { s.finish(true) }

// Fail marks the step as failed. The caller is responsible for reporting the
// underlying error.
func (s *Step) Fail() { s.finish(false) }

func (s *Step) finish(ok bool) {
	s.once.Do(func() {
		close(s.stop)
		<-s.done

		elapsed := ShortDuration(time.Since(s.start))

		if !s.ui.errTTY {
			if ok {
				s.ui.write(s.ui.err, "done.\n")
			} else {
				s.ui.write(s.ui.err, "failed.\n")
			}
			return
		}

		s.ui.write(s.ui.err, "\r\x1b[K")
		if ok {
			s.ui.write(s.ui.err, "  ✓ %s (%s)\n", s.msg, elapsed)
			return
		}
		s.ui.write(s.ui.err, "  %s %s (%s)\n", s.ui.paint("31", true, "✗"), s.msg, elapsed)
	})
}

// ShortDuration formats a duration for progress output, e.g. "0.4s", "1m02s".
func ShortDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}

	minutes := int(d / time.Minute)
	seconds := int((d % time.Minute) / time.Second)
	if minutes >= 60 {
		return fmt.Sprintf("%dh%02dm", minutes/60, minutes%60)
	}
	return fmt.Sprintf("%dm%02ds", minutes, seconds)
}

// HumanDuration formats a duration for status output, e.g. "58 minutes".
func HumanDuration(d time.Duration) string {
	switch {
	case d <= 0:
		return "expired"
	case d < time.Minute:
		return "less than a minute"
	case d < time.Hour:
		minutes := int(d.Minutes())
		if minutes == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", minutes)
	case d < 48*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%d days", days)
	}
}
