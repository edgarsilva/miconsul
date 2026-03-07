package main

import (
	"context"
	"flag"
	"fmt"
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

func colorize(color string, text string) string {
	if os.Getenv("NO_COLOR") != "" {
		return text
	}

	return color + text + ansiReset
}

func logStep(icon string, color string, msg string) {
	fmt.Printf("%s %s\n", colorize(color, icon), msg)
}

func main() {
	_ = godotenv.Load(".env")

	env := appenv.New()
	db, err := database.New(env, obslogging.Logger{})
	if err != nil {
		log.Printf("failed to initialize database: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	var opts seeder.Options
	var skipMigrate bool
	var verboseSQL bool

	flag.BoolVar(&opts.Baseline, "baseline", true, "create/update deterministic baseline seed data")
	flag.BoolVar(&opts.RandomizedBulk, "bulk", true, "create randomized bulk seed data")
	flag.IntVar(&opts.BulkUsers, "users", 5, "number of randomized users to create")
	flag.IntVar(&opts.BulkClinics, "clinics", 20, "number of randomized clinics to create")
	flag.IntVar(&opts.BulkPatients, "patients", 60, "number of randomized patients to create")
	flag.IntVar(&opts.BulkAppointments, "appointments", 120, "number of randomized appointments to create")
	flag.StringVar(&opts.OwnerEmail, "owner-email", "", "attach clinic/patient/appointment seed records to this existing user email")
	flag.BoolVar(&opts.EnsureOwner, "ensure-owner", false, "create owner-email user if missing (default password: SeedOwner123!)")
	flag.BoolVar(&skipMigrate, "skip-migrate", false, "skip running goose migrations before seeding")
	flag.BoolVar(&verboseSQL, "verbose-sql", false, "print SQL statements while seeding")
	flag.Parse()

	logStep("🚀", ansiCyan, "Starting database seed bootstrap")
	logStep("🧩", ansiCyan, fmt.Sprintf("Config: baseline=%t bulk=%t users=%d clinics=%d patients=%d appointments=%d",
		opts.Baseline,
		opts.RandomizedBulk,
		opts.BulkUsers,
		opts.BulkClinics,
		opts.BulkPatients,
		opts.BulkAppointments,
	))
	if opts.OwnerEmail != "" {
		logStep("👤", ansiCyan, fmt.Sprintf("Owner: email=%s ensure_owner=%t", opts.OwnerEmail, opts.EnsureOwner))
	}

	if !skipMigrate {
		logStep("🪿", ansiBlue, "Applying schema migrations")
		if err := database.ApplyMigrationsSilent(db, env); err != nil {
			log.Printf("failed to apply migrations: %v", err)
			os.Exit(1)
		}
	} else {
		logStep("⏭️", ansiYellow, "Skipping migrations (--skip-migrate)")
	}

	seedDB := db.GormDB()
	if !verboseSQL {
		seedDB = seedDB.Session(&gorm.Session{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
		logStep("🔇", ansiBlue, "SQL logs suppressed (use --verbose-sql to enable)")
	} else {
		logStep("📣", ansiYellow, "SQL logs enabled (--verbose-sql)")
	}

	logStep("🌱", ansiBlue, "Running seeders")
	result, err := seeder.Run(context.Background(), seedDB, opts)
	if err != nil {
		log.Printf("failed to seed db: %v", err)
		os.Exit(1)
	}

	logStep("✅", ansiGreen, "Seed completed")
	logStep("📊", ansiGreen, fmt.Sprintf("Totals: users=%d clinics=%d patients=%d appointments=%d total=%d",
		result.UsersCreated,
		result.ClinicsCreated,
		result.PatientsCreated,
		result.AppointmentsCreated,
		result.TotalCreated(),
	))
}
