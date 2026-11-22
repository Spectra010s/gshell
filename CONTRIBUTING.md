# Contributing

Thanks for taking the time to improve gshell.

## Before you start

Open an issue for anything larger than a bug fix or a small documentation
change. It is easier to agree on the shape of a change before it is written.

## Development

You need Go 1.27 or newer. Everything else is in the repository.

```sh
go build ./...
go vet ./...
gofmt -l .
```

`gofmt -l .` should print nothing. Run `gofmt -w .` to fix formatting.

Running the CLI needs a Google account with Cloud Shell access:

```sh
go run . auth login
go run . status
```

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
type(scope): summary
```

- `type` is one of `feat`, `fix`, `docs`, `chore`, `build`, `ci`, `refactor`,
  `test`, `perf`.
- `scope` names the area: `cli`, `auth`, `cloudshell`, `cp`, `keys`, `release`,
  `deps`.
- `summary` is lowercase, imperative, and has no trailing period.

```
fix(cloudshell): send credentials to the session on every connect
feat(cp): make cp the single transfer command
build(release): add goreleaser configuration
```

## Code style

- Errors are lowercase and single-line: `failed to read config: ...`.
- Results go to stdout. Progress, spinners, and diagnostics go to stderr.
- Green output is reserved for a meaningful success, not for routine steps.
- Keep state under `~/.config/gshell`; never write credentials anywhere else.

## Testing

There is no test suite yet. Verify the commands you touched by hand:

- `auth login`, `auth status`, `auth logout`
- `status` with a running and a suspended environment
- `run` with a command that succeeds and one that fails
- `cp` upload, download, `-r`, `--sync`, and `-n`
- `get-mount-command` output

State the platform you tested on (Linux, macOS, Windows, Termux) in the pull
request.

## Pull requests

- Keep one concern per pull request.
- Explain what changed and which commands you ran.
- Update the README when behavior or commands change.
- Do not bump the version or create tags; releases are cut by the maintainer.

## License

By contributing, you agree that your contributions are licensed under the MIT
License in [LICENSE](LICENSE).
