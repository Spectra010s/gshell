package cmd

import (
	"fmt"

	gshellSSH "gshell/internal/ssh"
	appui "gshell/internal/ui"
)

// reportTransfer prints per-file detail when requested and a one-line summary.
func reportTransfer(u *appui.UI, res *gshellSSH.Result, dryRun bool) {
	if dryRun {
		for _, a := range res.Actions {
			u.Printf("%s -> %s\n", a.Source, a.Dest)
		}
		u.Printf("%s would be copied (%s)\n", plural(res.Files, "file"), appui.Bytes(res.Bytes))
		return
	}

	for _, a := range res.Actions {
		if a.Skipped {
			u.Verbosef("  unchanged %s", a.Dest)
		} else {
			u.Verbosef("  %s -> %s", a.Source, a.Dest)
		}
	}

	switch {
	case res.Files == 0 && res.Skipped == 0:
		u.Successf("Nothing to copy")
	case res.Files == 0:
		u.Successf("%s unchanged", plural(res.Skipped, "file"))
	case res.Skipped > 0:
		u.Successf("Copied %s (%s), %d unchanged",
			plural(res.Files, "file"), appui.Bytes(res.Bytes), res.Skipped)
	default:
		u.Successf("Copied %s (%s)", plural(res.Files, "file"), appui.Bytes(res.Bytes))
	}
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
