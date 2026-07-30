# Contributing

Participation in this project is governed by the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Development

```sh
go test ./...
WATSON_DIR=$(mktemp -d) go run ./cmd/watson-tui
```

`golangci-lint run` should be clean before you open a PR (CI enforces it,
`gofmt` included).

## Commit messages

Prefixes used in this repo's history: `feat:`, `fix:`, `refactor:`, `test:`,
`docs:`, `style:`, `ci:`, `chore:`.

## Pull requests

CI builds, vets, tests and lints every PR. Keep the change focused — if it
touches behavior described in the README, update the README in the same PR.
