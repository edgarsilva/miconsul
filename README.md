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

[2024/Apr/21]

- [x] Basic project setup

[2024/Apr/25] - [2024/May/01]

- [x] Login page
- [x] Email/Password endpoints
- [x] Reset Password Functionality

[2024/May/02] - [2024/May/07]

- [x] Working Login Page
- [x] Reset Password Functionality
- [x] Signup page

[2024/May/08]

- [x] Confirm email Functionality

[2024/May/09]

- [x] Fix Schema Ids to use XID

**note**: v0 took more than 7ds narrow scope

[2024/May/10 - 2024/May/17]

- [x] Add ERD for Miconsul sample DB
- [x] Patients index page
- [x] Patients Create/Update Form
- [x] HTMX integration

[2024/May/18 - 2024/May/22]

- [x] Clinics views and endpoints
- [x] Appointments views and endpoints
- [x] Basic Toast and notifications

[2024/May/23 - 2024/May/31]

- [x] Added patient actions and emails

[2024/Jun/01 - 2024/Jun/06]

- [x] Running Somewhere other than the dev machine
- [x] Initial Deployment using coolify
- [x] Make updates and Docker file

[2024/Jun/10 - 2024/Jun/11]

- [x] Setup servers DNS and Coolify container/service manager
- [x] Hostinger backup for Sqlite DB backup streams with Litestream/Minio
- [x] Sep up Goose migrations
- [x] Minio install for Litestream

### V1 (We can consider v1 Alpha started here)

[2024/Jun/12 - 2024/Jun/17]

- [x] Appointment select day filtering by default
- [x] Appointment filtering by clinic
- [x] Clinic index page search
- [x] Patient index page search

[2024/Jun/18 - 2024/Jun/24]

- [x] Appointment add clinic icon
- [x] Overall architecture design
- [x] Optional authentication integration with Logto identity manage/provider

[2024/Jun/25 - 2024/Jun/30]

- [ ] Fix navigation buttons on mobile
- [ ] Accept profile pic from Logto (This is partially done, need to check the identity of social media to pull from there)
- [ ] Accept file uploads for avatars
- [ ] Upload images to object storage

- [ ] Dashboard updates

  - [ ] Show favorite clinic in Dashboard

- [ ] Email/Messages updates

  - [ ] Show appointment clinic and professional profile in emails and messages
  - [ ] Update actions in emails/messages to use pro info

- [ ] Release to beta testers

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
