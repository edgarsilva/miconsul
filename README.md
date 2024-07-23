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
[vokoscreenNG-2024-07-23_14-35-58.webm](https://github.com/user-attachments/assets/ed603525-3912-4c27-8d2c-5896b3d21a7c)

## Getting Started

These instructions will get you a copy of the project up and running
on your local machine for development, scaffolding and testing purposes.

Other sections still pending in the README:

Note: _working_ but pending readme section

1. Automated production deployments with Coolify
2. Sqlite Litestream backups
3. Object storage with Minio (used by Litestream)
4. Development feature brances.

## MakeFile

### Install dependencies and tooling (not Go itself)

Install dev environment tooling, you must have go correctly installed and in
your path.

It will install:

- bun: for TailwindCSS
- TailwindCSS: plugins
- goose: for migrations
- templ: to build/compile templ files into go code

```bash
make install
```

### Build the application

Generate TailwindCSS styles, templ files and go executable (go build)

```bash
make build
```

### Start the app (for docker and prod deployments)

Apply migrations to the DB and start the application (used in prod docker deployment)

```bash
make start
```

### Development

This is what you should be using for development, will auto generate css, templ
files and translations and enable auto reloading of the server on file changes
(not browser, just refresh the page, hit [f5] and done).

```bash
make dev
```

### Run the app manually in dev mode (requires manual restart of the app)

If for whatever reason you don't want to use Air and have auto reloading enabled
(use `make dev` above for that), this will generate the css and templ files and `go run`.

```bash
make run
```

### DB - Create Database

Creates an Sqlite file at `./database/app.sqlite` to use for the app

```bash
make db/create
```

### DB - Delete Database

It deletes the database file

```bash
make db/delete
```

### DB - Setup the database

It deletes, then re-creates the Database and runs all migrations to start with a
clean slate.

```bash
make db/setup
```

### DB - Dump schema

It dumps the DB schema into a file at `./database/schema.sql`

```bash
make db/dump-schema
```

### DB - Create migration

Creates a new migration file for the DB.

```bash
make db/migration name=add_column_price_to_payments
```

### DB - Check migrations status

Lists Goose migration status

```bash
make db/status
```

### DB - Run migrations

Runs pending migrations

```bash
make db/migrate
```

### DB - Rollback last migrations

Rollbacks the last migration

```bash
make db/rollback
```

### DB - Redo last migration

Rollbacks then re-runs last migration

```bash
make db/redo
```

## Overall architecture guidelines for new features (WIP)

![image](https://github.com/edgarsilva/miconsul/assets/518231/6c270679-a3dc-432b-9394-08c7857eb1ea)

## Overall Data Models and ERD (WIP needs updates)

![image](https://github.com/edgarsilva/miconsul/assets/518231/c37e3599-65d6-4e73-814b-54aa91576b3b)
