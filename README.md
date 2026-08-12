# Yet-Another-Pasting-Service

[![Release](https://img.shields.io/github/v/release/yaps-sh/yaps)](https://github.com/yaps-sh/yaps/releases)
[![License](https://img.shields.io/github/license/yaps-sh/yaps)](./LICENSE)
[![Website](https://img.shields.io/badge/website-yaps.sh-f0a83c)](https://yaps.sh)

A small, self-hostable pasting service. My version of a successor to hastebin before toptal killed it and made it worse
entirely.

Built to be minimal UI and easy to use. No frills. One binary, one SQLite file. No requirement on external dependencies
or services.

YAPS is under active development, there **will** be gaps and breaking changes. See [status](#status) for more info.

## Why?

Most pasting services are abandoned, bloated, or just a lot of UI fluff which gets in the way. I always loved hastebin
because of its simplicity and ease of use. I also wanted some more features on top of hastebin, so I built YAPS.

## Quickstart

1. Download the [latest release](https://github.com/yaps-sh/yaps/releases/latest)
2. Copy the [`config.toml`](./config.example.toml)
3. Run: `./yaps`

By default, YAPS listens on `:3000` and stores its database at `data/yaps.db`. See [config](./config.example.toml) for
more configuration options.

## Contributing

### Pre-Requisites

**Go 1.26.5 is required**

```sh
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
go install github.com/a-h/templ/cmd/templ@latest
```

### Commands

All commands have shortcuts in `Makefile`. Here are some common ones:

- `make generate`, generate templ and sqlc
- `make migrate/up`, run migrations
- `make migrate/create my_migration`, create a new migration

> [!IMPORTANT]
> `templ generate` and `sqlc generate` must run before `go build`. `make generate` will run both `templ generate` and
`sqlc generate`.

### Running

GoLand project files are provided in the repository. If you use GoLand you can use it's runner to automatically generate
and run YAPS.

## Status

This project is pre-1.0 and under active development. Core anonymous paste create/view flow works today.

End goal is:

- burn after read, self destructing pastes
- authentication
- password locked pastes
- private, unlisted, and public pastes