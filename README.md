# gshell

> A lightweight bash wrapper around `gcloud alpha cloud-shell` that makes working with Google Cloud Shell from your terminal feel natural.

Instead of typing `gcloud alpha cloud-shell ssh --authorize-session` every time, you just type `gshell`.

**Author**: [Adeloye Adetayo](https://spectra010s.biuld.app)

---

## Features

- **Quick shell** — open an interactive Cloud Shell session (`gshell`)
- **Send / Get** — upload or download files and folders (`gshell send`, `gshell get`)
- **Sync** — upload only changed files, fast incremental sync (`gshell sync`)
- **Run** — run arbitrary commands remotely (`gshell run`)
- **Browser open** — open Cloud Shell in the browser (`gshell open`)
- **Status** — show current gcloud account and project info (`gshell status`)

---

## Requirements

- [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) (`gcloud`)
- `gcloud` alpha components — the installer will detect and offer to set this up for you

---

## Installation

```bash
git clone https://github.com/Spectra010s/gshell.git
cd gshell
chmod +x install.sh
./install.sh
```

Then reload your shell:

```bash
source ~/.bashrc
```

The installer will:
- Check if `gcloud` is installed and offer to open the install page if not
- Check if alpha components are available and offer to install them
- Copy `gshell` to `$PREFIX/bin` (works on Termux and standard Linux)
- Set up tab completion automatically

### Manual installation

- Copy `gshell` to a directory on your `PATH` and make it executable
- Copy `gshell-completion.bash` into your Bash completion directory and source it

---

## Files

- `gshell` — main script (CLI wrapper around Cloud Shell)
- `install.sh` — installer that copies files and sets up completion
- `gshell-completion.bash` — Bash tab completion for all subcommands

---

## Usage

```bash
gshell                           # Enter Cloud Shell (interactive SSH)
gshell send [-r] <file|folder>   # Upload file or folder to Cloud Shell
gshell get  [-r] <file|folder>   # Download file or folder from Cloud Shell
gshell sync [-r] <file|folder>   # Sync only changed files to Cloud Shell
gshell run  <command>            # Run a command on Cloud Shell
gshell open                      # Open Cloud Shell in browser
gshell status                    # Show current gcloud project & account info
gshell -v | --version            # Show version
gshell -h | --help               # Show help
```

### Flags

- `-r` — recursive, for folders on `send`, `get`, and `sync`
- `-n` / `--dry-run` — preview what would be transferred without uploading or downloading (available on `send` and `get`)

---

## Examples

```bash
# Enter Cloud Shell
gshell

# Upload a single file
gshell send index.html

# Upload a folder
gshell send -r myproject/

# Preview what would be uploaded without transferring
gshell send -n -r myproject/

# Download a file from Cloud Shell
gshell get config.json

# Download a folder
gshell get -r bd/

# Sync a single changed file
gshell sync bd/index.html

# Sync an entire folder (only uploads changed files)
gshell sync -r bd/

# Run a command remotely
gshell run ls ~
gshell run "cd myproject && git status"

# Check your active project and account
gshell status

# Open Cloud Shell in the browser
gshell open
```

---

## How sync works

`gshell sync` compares local and remote files using checksums (`md5sum` or `md5` depending on your system, with a size/mtime fallback). Only files that have changed are uploaded — similar to `rsync` but built entirely on top of `gcloud scp`.

---

## Tab completion

Tab completion is set up automatically by the installer. After reloading your shell, type `gshell` and press Tab to see all available commands, flags, and file suggestions.

---

## Roadmap & Development Phases (v2.0 Go Rewrite)

- [x] **Phase 1: CLI Skeleton & Flags (Cobra)**
  - Initialize Go module and Cobra CLI framework
  - Scaffold subcommands: `upload` (alias `send`), `download` (alias `get`), `sync`, `run`, `status`
  - Implement namespace conventions (`remote:`) and multi-file destination parsing
- [x] **Phase 2: Authentication & Token Management**
  - Implement OAuth 2.0 Device Flow (RFC 8628) for standalone terminal authentication
  - Local credential caching (`~/.config/gshell/credentials.json`) with auto-refresh
- [x] **Phase 3: Cloud Shell API Client**
  - Direct integration with `cloudshell.googleapis.com` (start environment, poll status)
  - Ephemeral SSH keypair generation and public key authorization
- [x] **Phase 4: Native SSH & SFTP Engine**
  - Interactive terminal PTY (`golang.org/x/term`)
  - Remote command execution (`gshell run`)
  - High-speed SFTP file transfers and checksum-based delta sync
- [ ] **Phase 5: Packaging & Release**
  - Multi-platform static binary builds (Linux, macOS, Termux)

---

## Contributing

PRs welcome. Keep changes small and focused. Update the README if you change CLI behavior.

---

## License

MIT - see the [LICENSE](LICENSE) file for details.

