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
- [x] Signup page

[08/May/24]

- [x] Confirm email Functionality

[09/May/24]

- [x] Fix Schema Ids to use XID

[10/May/24 - 17/May/24]

- [x] Add ERD for Miconsul sample DB
- [x] Patients index page
- [x] Patients Create/Update Form
- [x] HTMX integration

[18/May/24 - 22/May/24]

- [x] Clinics views and endpoints
- [x] Appointments views and endpoints
- [x] Basic Toast and notifications

[23/May/24 - 31/May/24]

- [x] Added patient actions and emails

[01/Jun/24 - 06/Jun/24]

- [x] Initial Deployment using coolify
- [x] Make updates and Docker file

[10/Jun/24 - 11/Jun/24]

- [x] Hostinger backup for coolify
- [x] Sep up Goose migrations
- [x] Minio install for Litestream

[12/Jun/24 - 17/Jun/24]

- [x] Appointment select day filtering by default
- [x] Appointment filtering by clinic
- [ ] Clinic index page search
- [ ] Patient index page search
- [ ] Appointment add clinic icon
- [ ] Show favorite clinic in Dashboard
- [ ] Getting ready for V1

### V1

[11/May/24]

- Add Models for Miconsul V1

[13/May/24] - [18/May/24]

- [ ] Running Somewhere other than the dev machine

## Getting Started

These instructions will get you a copy of the project up and running on your
local machine for development and testing purposes. See deployment for notes on
how to deploy the project on a live system.

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
