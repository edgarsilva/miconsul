# Go Scaffold

A GoScaffold allows you to quickly set-up Ready to deploy Web application projects
using Go, SQLite with GORM, Templ with HTMX and DaisyUI/TailwindCSS:

- Go Web Server: Fiber
- Database: SQLite3 (PostreSQL/MySQL)
- ORM/SQL Query Builder: GORM
- HTML/Templating: Templ and HTMX
- UI/CSS: DaisyUI/TailwindCSS

## Release Milestones

### Ongoing
- [ ] README and Go Env Setup/Updates

### V0 (7day)

[21/Apr/24]
- [x] Basic project setup

[24/Apr/24] - [01/May/24]
- [x] Login page
- [x] Email/Password endpoints
- [x] Reset Password Functionality

[02/May/24] - [07/May/24]
- [x] Working Login Page
- [x] Reset Password Functionality
- [X] Signup page

[08/May/24]
- [X] Confirm email Functionality

[09/May/24]
- [X] Fix Schema Ids to use XID

[10/May/24]
- [ ] Add ERD for Miconsul sample DB 


### V1 (7day)

[11/May/24]
- Add Models for Miconsul V1

[13/May/24] - [18/May/24]
- [ ] Running Somewhere other than the dev machine

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
