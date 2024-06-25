# Miconsul: patient appointment planner and notification center

Base on my GoScaffold project which allows you to quickly set-up Ready to deploy Web application projects
using Go, SQLite with GORM, Templ with HTMX and DaisyUI/TailwindCSS:

- Go Web Server: Fiber
- Database: SQLite3 (PostreSQL/MySQL)
- ORM/SQL Query Builder: GORM
- HTML/Templating: Templ and HTMX
- UI/CSS: DaisyUI/TailwindCSS

## Release Milestones

### Ongoing

- [ ] README and Go Env Setup/Updates

### V0 (7day proof of concept)

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

**note**: v0 took more than 7ds narrow scope

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

- [x] Running Somewhere other than the dev machine
- [x] Initial Deployment using coolify
- [x] Make updates and Docker file

[10/Jun/24 - 11/Jun/24]

- [x] Setup servers DNS and Coolify container/service manager 
- [x] Hostinger backup for Sqlite DB backup streams with Litestream/Minio
- [x] Sep up Goose migrations
- [x] Minio install for Litestream

### V1 (We can consider v1 Alpha started here)

[12/Jun/24 - 17/Jun/24]

- [x] Appointment select day filtering by default
- [x] Appointment filtering by clinic
- [x] Clinic index page search
- [x] Patient index page search
- [x] Appointment add clinic icon
- [ ] Show favorite clinic in Dashboard
- [ ] Show appointment clinic and profesional in emails and messaging
- [x] Overall architecture design
- [x] Optional authentication integration with Logto identity manage/provider

[01/Jun/24 - 30/Jun/24] 

- [] Reaease to beta testers

[13/May/24] - [18/May/24]

## Overall architecture guidelines for new features (WIP)

![image](https://github.com/edgarsilva/miconsul/assets/518231/6c270679-a3dc-432b-9394-08c7857eb1ea)

## Overall Data Models and ERD (WIP needs updates)

![image](https://github.com/edgarsilva/miconsul/assets/518231/c37e3599-65d6-4e73-814b-54aa91576b3b)

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
