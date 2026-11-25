# gshell

> A terminal client for Google Cloud Shell.

`gshell` talks to Google Cloud Shell directly, without `gcloud`, Python, or a bundled SDK. It signs in with your Google account, manages the environment, and gives you an interactive session, one-shot commands, and file transfers over SSH/SFTP.

---

## Features

- **Interactive shell** — full PTY session with terminal resize
- **Run** — execute a command on Cloud Shell and stream its output
- **Copy** — `cp` in either direction over SFTP, with dry-run and checksum sync
- **Project** — activate a project inside the session, the same way `gcloud` does
- **Auth** — PKCE sign-in, account display, logout with server-side revocation
- **Mount command** — print the `sshfs` command for your home directory

---

## Requirements

- A Google account with Cloud Shell access
- Linux, macOS, or Windows

Building from source needs Go 1.27 or newer. Installing a release does not.

---

## Installation

Linux and macOS:

```sh
curl -fsSL https://raw.githubusercontent.com/Spectra010s/gshell/main/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://raw.githubusercontent.com/Spectra010s/gshell/main/install.ps1 | iex
```

Both installers resolve the latest release, download the archive for your
platform, verify its SHA-256 against `checksums.txt`, and install the binary
(`~/.local/bin` on Unix, `%LOCALAPPDATA%\Programs\gshell` on Windows), adding it
to `PATH` when needed.

| Override | Effect |
|---|---|
| `GSHELL_VERSION=2.0.0` / `-Version 2.0.0` | Install a specific tag instead of the latest |
| `GSHELL_INSTALL_DIR=~/bin` / `-InstallDir C:\Tools\gshell` | Choose the destination |
| `PREFIX=/usr/local` | Unix destination becomes `$PREFIX/bin` |

To install by hand, download the archive for your platform from the
[releases page](https://github.com/Spectra010s/gshell/releases), verify it
against `checksums.txt`, and put the binary on your `PATH`.

### Termux

```sh
pkg install tur-repo
pkg install gshell
```

`gshell` is packaged in the [Termux User Repository](https://github.com/termux-user-repository/tur)
(TUR), which is why `tur-repo` comes first.

The release archive is not usable on Termux: a prebuilt Linux binary cannot
resolve DNS there, because Termux has no `/etc/resolv.conf`. The TUR package is
built for Termux from source instead.

To build it yourself, install `golang` and `git`, clone the repository, and run
`go build .`.

---

## Quick start

```sh
gshell auth login    # sign in once, in the browser
gshell               # open a Cloud Shell session
```

---

## Commands

### `gshell`

Opens an interactive session. If a project is configured, it is passed to the
session as `DEVSHELL_PROJECT_ID`, which Cloud Shell turns into `gcloud config`.

### `gshell auth`

```sh
gshell auth login    # sign in with the OAuth PKCE flow
gshell auth status   # show the account and credentials path
gshell auth logout   # revoke the token with Google, then delete it locally
```

```
$ gshell auth status
✓ Authenticated to Google Cloud Shell
  Account: you@example.com
  Credentials: ~/.config/gshell/credentials.json
```

### `gshell status`

Shows the environment without opening a shell.

```
$ gshell status
  Account      you@example.com
  Project      my-project
  State        Running
  SSH          user@35.189.228.144:6000
  Public keys  10 registered
```

If the environment is suspended, the next `gshell` or `gshell run` starts it.

### `gshell run`

Runs a command on Cloud Shell and streams stdout/stderr.

```sh
gshell run ls -la
gshell run "cd project && git status"
```

### `gshell cp`

Copies files between your machine and Cloud Shell. Direction is determined
by the `remote:` prefix; paths after it are relative to the remote home.

```sh
gshell cp notes.txt remote:~/            # upload
gshell cp -r ./project remote:~/project  # upload a directory
gshell cp remote:~/app.log ./logs/       # download
```

| Flag | Meaning |
|---|---|
| `-r, --recursive` | Required when any source is a directory |
| `-e, --expand` | Evaluate remote names as a bash expression (globs, `~`, `$VAR`) |
| `-n, --dry-run` | Print planned transfers without copying |
| `--sync` | Transfer only files whose checksums differ |

```
$ gshell cp -r --sync ./project remote:~/project
✓ Copied 2 files (12.4 KB), 5 unchanged

$ gshell cp -n notes.txt remote:~/
notes.txt -> remote:~/notes.txt
1 file would be copied (128 B)
```

`--dry-run` output goes to stdout; progress goes to stderr, so
`gshell cp ... 2>/dev/null` shows only results. Per-file lines for real
transfers require `--verbose`.

### `gshell config`

```sh
gshell config set project my-project   # project used for new sessions
gshell config show
```

### `gshell get-mount-command`

Prints the `sshfs` command that would mount the Cloud Shell home directory at a
local path. gshell does not run it.

Mounting attaches your Cloud Shell home directory to a local path over sshfs,
so you can access Cloud Shell files from your local directory tree.

```
$ gshell get-mount-command ~/cloudshell
sshfs user@35.189.228.144: ~/cloudshell -p 6000 -oIdentityFile=... -oStrictHostKeyChecking=no
```

Requires `sshfs` and FUSE, plus permission to mount. Not available on Android
without root.

### `gshell completion`

Generates completion scripts for `bash`, `zsh`, `fish`, and `powershell`:

```sh
gshell completion bash
```

---

## Projects

The project is the Google Cloud project your Cloud Shell session works in.
Inside the session it becomes the active project, so `gcloud` and other tools
use it without extra flags, and `gshell status` reports it.

Set a default once:

```sh
gshell config set project my-project
```

Override it for a single run with `--project` or `GSHELL_PROJECT`.

Resolution order:

1. `--project <id>`
2. `GSHELL_PROJECT` environment variable
3. `gshell config set project <id>`

If none is set, the session starts without a project (`gshell status` shows
`Project  not set`).

---

## Config files

gshell stores its credentials, settings, and SSH key under `~/.config/gshell`.

| Path | Contents |
|---|---|
| `~/.config/gshell/credentials.json` | OAuth token and account email (0600) |
| `~/.config/gshell/config.json` | Saved project |
| `~/.config/gshell/id_rsa` | SSH private key (0600), generated on first connect |

## Environment variables

| Variable | Effect |
|---|---|
| `GSHELL_PROJECT` | Project to activate, overrides saved config |
| `NO_COLOR` | Disable colored output |
| `TERM=dumb` | Disable color and spinner animation |
| `GSHELL_CLIENT_ID` / `GSHELL_CLIENT_SECRET` | Use a custom OAuth client |

### Global flags

| Flag | Effect |
|---|---|
| `--project <id>` | Project to activate for this run |
| `--verbose` | Show detailed output, including per-file transfers |
| `--no-color` | Disable colored output |

---

## Output and exit codes

Results go to stdout, progress and diagnostics to stderr. `gshell` exits `0` on
success and `1` on failure. A remote command that fails is reported as an error
with its exit status; gshell does not exit with the remote code.

---

## Contributing

Bug reports and pull requests are welcome. Read
[CONTRIBUTING.md](CONTRIBUTING.md) first — it covers the build, commit message,
and testing conventions.

### Project structure

| Directory | Purpose |
|---|---|
| `cmd/` | Cobra commands and flag wiring |
| `internal/auth/` | OAuth PKCE flow, token storage, revocation |
| `internal/cloudshell/` | Cloud Shell REST client, keys, environment lifecycle |
| `internal/ssh/` | SSH, PTY, command execution, SFTP transfers |
| `internal/config/` | Saved settings (`config.json`) |
| `internal/ui/` | stdout/stderr split, progress, color |

---

## Maintainer

[Adeloye Adetayo](https://spectra010s.com)

---

## License

MIT — see [LICENSE](LICENSE).
