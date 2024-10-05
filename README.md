# Miconsul: patient appointment planner and notification center

Based on my GoScaffold project which allows you to quickly set-up Ready to
deploy Web application projects using Go, SQLite with GORM, Templ with HTMX
and DaisyUI/TailwindCSS:

- Go Web Server: [Fiber](https://docs.gofiber.io/)
- Database: [SQLite3 (or PostreSQL/MySQL)](https://sqlite.org/index.html)
- ORM/SQL Query Builder: [GORM](https://gorm.io/docs/)
- HTML/Templating: [Templ](https://templ.guide/)
- Client/Server interactions: [HTMX](https://htmx.org/)
- Client Interactivity: [AlpineJS](https://alpinejs.dev/start-here)
- UI/CSS: [DaisyUI](https://daisyui.com)/[TailwindCSS](https://tailwindcss.com/)

## The MVP App

[vokoscreenNG-2024-07-23_17-00-32.webm](https://github.com/user-attachments/assets/f6915e3a-bb64-4a34-8ccc-78bec186f4a3)

## Getting Started

These instructions will get you a copy of the project up and running
on your local machine for development, scaffolding and testing purposes.

Other sections still pending in the README:

Note: _working_ but pending readme section

1. Automated production deployments with Coolify
2. Sqlite Litestream backups
3. Object storage with Minio (used by Litestream)
4. Development feature brances.

## Justfile

I've added `just` recipes for the most common tasks, you can list them by
running `just`.

To install `justfile` support on your system run:

```bash
# you might need sudo or install to a diff directory that makes just available
# in your path.
$ curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin

# then run
$ just
just --list
Available recipes:
    build                     # Build the app
    clean                     # Clean builds
    default                   # Display this list of recipes
    dev                       # Start app in dev mode
    fmt                       # Run Go formatter/linter
    install                   # Install deps 🥐 Bun, 🪿 goose and 🛕 templ
    integration-test          # Run integration-test
    run                       # Run the app
    start                     # Start the app
    tailwind                  # Generate Tailwind styles
    templ                     # Generate templ files
    test                      # Run tests
    unit-test                 # Run unit-tests
    vet                       # Run Go vet to detect possible issues

    [db]
    db-create                 # Create Database
    db-delete                 # Deletes the DB giving you a choice.
    db-dump-schema            # Dumps the DB schema to ./database/schema.sql
    db-migrate                # Migrates the DB to latest migration
    db-setup                  # Set up the DB by running delete, create and migrate
    migration-create arg_name # Creates a new migration for the DB
    migration-redo            # Redo the last migration
    migration-rollback        # Rollbacks last migration
    migration-status          # Lists the DB migration status

    [docker]
    docker-down               # Terminates the docker services
    docker-logs               # Shows DB service logs
    docker-up                 # Starts the docker services
    docker-up-detached        # Starts the docker services detached

    [migration]
    migration-create arg_name # Creates a new migration for the DB
    migration-redo            # Redo the last migration
    migration-rollback        # Rollbacks last migration
    migration-status          # Lists the DB migration status
```

### Install dependencies and tooling (not Go itself)

Install dev environment tooling, you must have `go` correctly installed and in
your path.

It will install:

- bun: for TailwindCSS
- TailwindCSS: plugins
- goose: for migrations
- templ: to build/compile templ files into go code

```bash
just install
```

### DB - Create Database

Creates an Sqlite file at `./database/app.sqlite` to use for the app

```bash
just setup
```

### Development

To run the app in dev mode, will auto generate css, templ
files and translations and enable auto reloading of the server on file changes
(not browser, just refresh the page, hit [f5] and done).

```bash
just dev
```

## Overall architecture guidelines for new features (WIP)

![image](https://github.com/edgarsilva/miconsul/assets/518231/6c270679-a3dc-432b-9394-08c7857eb1ea)

## Overall Data Models and ERD (WIP needs updates)

![image](https://github.com/edgarsilva/miconsul/assets/518231/c37e3599-65d6-4e73-814b-54aa91576b3b)
