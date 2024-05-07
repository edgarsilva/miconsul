# Go Scaffold

A GoScaffold allows you to quickly set-up Ready to deploy Web application projects
using Go, SQLite with GORM, Templ with HTMX and DaisyUI/TailwindCSS:

- Go Web Server: Fiber
- Database: SQLite3 (PostreSQL/MySQL)
- ORM/SQL Query Builder: GORM
- HTML/Templating: Templ and HTMX
- UI/CSS: DaisyUI/TailwindCSS

## Release Milestones

### V0 (1day)

- [ ] README and Go Env Setup
- [ ] Simple Login page
- [ ] Simple Email/Password endpoints

- [ ] Running Somewhere other than the dev machine

### V1 (7 days)

- [ ] Working Login Page
- [ ] Reset Password Functionality

## Getting Started

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes. See deployment for notes on how to deploy the project on a live system.

## MakeFile

run all make commands with clean tests

```bash
make all build
```

build the application

```bash
make build
```

run the application

```bash
make run
```

Create DB container

```bash
make docker-run
```

Shutdown DB container

```bash
make docker-down
```

live reload the application

```bash
make watch
```

run the test suite

```bash
make test
```

clean up binary from the last build

```bash
make clean
```
