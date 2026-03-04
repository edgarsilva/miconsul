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

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load(".env")

	env := appenv.New()
	db := database.New(env)
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	var opts seeder.Options
	var skipMigrate bool

	flag.BoolVar(&opts.Baseline, "baseline", true, "create/update deterministic baseline seed data")
	flag.BoolVar(&opts.RandomizedBulk, "bulk", true, "create randomized bulk seed data")
	flag.IntVar(&opts.BulkUsers, "users", 5, "number of randomized users to create")
	flag.IntVar(&opts.BulkClinics, "clinics", 20, "number of randomized clinics to create")
	flag.IntVar(&opts.BulkPatients, "patients", 60, "number of randomized patients to create")
	flag.IntVar(&opts.BulkAppointments, "appointments", 120, "number of randomized appointments to create")
	flag.StringVar(&opts.OwnerEmail, "owner-email", "", "attach clinic/patient/appointment seed records to this existing user email")
	flag.BoolVar(&opts.EnsureOwner, "ensure-owner", false, "create owner-email user if missing (default password: SeedOwner123!)")
	flag.BoolVar(&skipMigrate, "skip-migrate", false, "skip running goose migrations before seeding")
	flag.Parse()

	if !skipMigrate {
		if err := database.ApplyMigrations(db, env); err != nil {
			log.Printf("failed to apply migrations: %v", err)
			os.Exit(1)
		}
	}

	result, err := seeder.Run(context.Background(), db.GormDB(), opts)
	if err != nil {
		log.Printf("failed to seed db: %v", err)
		os.Exit(1)
	}

	fmt.Printf("seed completed: users=%d clinics=%d patients=%d appointments=%d total=%d\n",
		result.UsersCreated,
		result.ClinicsCreated,
		result.PatientsCreated,
		result.AppointmentsCreated,
		result.TotalCreated(),
	)
}
