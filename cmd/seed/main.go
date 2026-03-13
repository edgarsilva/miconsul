package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"miconsul/internal/database"
	"miconsul/internal/database/seeder"
	"miconsul/internal/lib/appenv"
	obslogging "miconsul/internal/observability/logging"

	"github.com/joho/godotenv"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiBlue   = "\033[34m"
	ansiCyan   = "\033[36m"
	ansiYellow = "\033[33m"
)

type seedRuntimeDeps struct {
	loadDotEnv      func(filenames ...string) error
	loadEnv         func() (*appenv.Env, error)
	openDB          func(env *appenv.Env) (*database.Database, error)
	applyMigrations func(db *database.Database) error
	runSeeder       func(ctx context.Context, db *gorm.DB, opts seeder.Options) (seeder.Result, error)
	logPrintf       func(format string, v ...any)
	logStep         func(icon string, color string, msg string)
}

var logStepWriter io.Writer = os.Stdout

var (
	runSeedForMain  = runSeed
	seedExitForMain = os.Exit
	seedLogForMain  = log.Printf
)

func main() {
	if err := runSeedForMain(context.Background(), os.Args[1:], defaultSeedRuntimeDeps()); err != nil {
		seedLogForMain("seed failed: %v", err)
		seedExitForMain(1)
	}
}

func defaultSeedRuntimeDeps() seedRuntimeDeps {
	return seedRuntimeDeps{
		loadDotEnv: godotenv.Load,
		loadEnv:    appenv.New,
		openDB: func(env *appenv.Env) (*database.Database, error) {
			return database.New(env, obslogging.Logger{}, nil)
		},
		applyMigrations: func(db *database.Database) error {
			return database.ApplyMigrationsSilent(db, obslogging.Logger{})
		},
		runSeeder: seeder.Run,
		logPrintf: log.Printf,
		logStep:   logStep,
	}
}

func runSeed(ctx context.Context, args []string, deps seedRuntimeDeps) error {
	if deps.loadDotEnv != nil {
		_ = deps.loadDotEnv(".env")
	}

	env, err := deps.loadEnv()
	if err != nil {
		return fmt.Errorf("load environment config: %w", err)
	}

	db, err := deps.openDB(env)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			deps.logPrintf("failed to close database: %v", err)
		}
	}()

	opts, skipMigrate, verboseSQL, err := parseSeedArgs(args)
	if err != nil {
		return fmt.Errorf("parse seed args: %w", err)
	}

	deps.logStep("🚀", ansiCyan, "Starting database seed bootstrap")
	deps.logStep("🧩", ansiCyan, fmt.Sprintf("Config: baseline=%t bulk=%t users=%d clinics=%d patients=%d appointments=%d",
		opts.Baseline,
		opts.RandomizedBulk,
		opts.BulkUsers,
		opts.BulkClinics,
		opts.BulkPatients,
		opts.BulkAppointments,
	))
	if opts.OwnerEmail != "" {
		deps.logStep("👤", ansiCyan, fmt.Sprintf("Owner: email=%s ensure_owner=%t", opts.OwnerEmail, opts.EnsureOwner))
	}

	if !skipMigrate {
		deps.logStep("🪿", ansiBlue, "Applying schema migrations")
		if err := deps.applyMigrations(db); err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
	} else {
		deps.logStep("⏭️", ansiYellow, "Skipping migrations (--skip-migrate)")
	}

	seedDB := db.GormDB()
	if !verboseSQL {
		seedDB = seedDB.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
		deps.logStep("🔇", ansiBlue, "SQL logs suppressed (use --verbose-sql to enable)")
	} else {
		deps.logStep("📣", ansiYellow, "SQL logs enabled (--verbose-sql)")
	}

	deps.logStep("🌱", ansiBlue, "Running seeders")
	result, err := deps.runSeeder(ctx, seedDB, opts)
	if err != nil {
		return fmt.Errorf("run seeders: %w", err)
	}

	deps.logStep("✅", ansiGreen, "Seed completed")
	deps.logStep("📊", ansiGreen, fmt.Sprintf("Totals: users=%d clinics=%d patients=%d appointments=%d total=%d",
		result.UsersCreated,
		result.ClinicsCreated,
		result.PatientsCreated,
		result.AppointmentsCreated,
		result.TotalCreated(),
	))

	return nil
}

func parseSeedArgs(args []string) (opts seeder.Options, skipMigrate bool, verboseSQL bool, err error) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.BoolVar(&opts.Baseline, "baseline", true, "create/update deterministic baseline seed data")
	fs.BoolVar(&opts.RandomizedBulk, "bulk", true, "create randomized bulk seed data")
	fs.IntVar(&opts.BulkUsers, "users", 5, "number of randomized users to create")
	fs.IntVar(&opts.BulkClinics, "clinics", 20, "number of randomized clinics to create")
	fs.IntVar(&opts.BulkPatients, "patients", 60, "number of randomized patients to create")
	fs.IntVar(&opts.BulkAppointments, "appointments", 120, "number of randomized appointments to create")
	fs.StringVar(&opts.OwnerEmail, "owner-email", "", "attach clinic/patient/appointment seed records to this existing user email")
	fs.BoolVar(&opts.EnsureOwner, "ensure-owner", false, "create owner-email user if missing (default password: SeedOwner123!)")
	fs.BoolVar(&skipMigrate, "skip-migrate", false, "skip running goose migrations before seeding")
	fs.BoolVar(&verboseSQL, "verbose-sql", false, "print SQL statements while seeding")

	if err := fs.Parse(args); err != nil {
		return seeder.Options{}, false, false, err
	}

	return opts, skipMigrate, verboseSQL, nil
}

func colorize(color string, text string) string {
	if os.Getenv("NO_COLOR") != "" {
		return text
	}

	return color + text + ansiReset
}

func logStep(icon string, color string, msg string) {
	fmt.Fprintf(logStepWriter, "%s %s\n", colorize(color, icon), msg)
}
