package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"miconsul/internal/database"
	"miconsul/internal/database/seeder"
	"miconsul/internal/lib/appenv"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestColorize(t *testing.T) {
	t.Run("returns raw text when NO_COLOR is set", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		got := colorize(ansiGreen, "ok")
		if got != "ok" {
			t.Fatalf("colorize() = %q, want %q", got, "ok")
		}
	})

	t.Run("returns colored text when NO_COLOR is not set", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		got := colorize(ansiGreen, "ok")
		want := ansiGreen + "ok" + ansiReset
		if got != want {
			t.Fatalf("colorize() = %q, want %q", got, want)
		}
	})
}

func TestParseSeedArgs(t *testing.T) {
	t.Run("uses defaults", func(t *testing.T) {
		opts, skipMigrate, verboseSQL, err := parseSeedArgs(nil)
		if err != nil {
			t.Fatalf("parseSeedArgs(nil) unexpected error: %v", err)
		}
		if !opts.Baseline || !opts.RandomizedBulk {
			t.Fatalf("parseSeedArgs(nil) baseline/bulk = %v/%v, want true/true", opts.Baseline, opts.RandomizedBulk)
		}
		if opts.BulkUsers != 5 || opts.BulkClinics != 20 || opts.BulkPatients != 60 || opts.BulkAppointments != 120 {
			t.Fatalf("parseSeedArgs(nil) got bulk values users=%d clinics=%d patients=%d appointments=%d", opts.BulkUsers, opts.BulkClinics, opts.BulkPatients, opts.BulkAppointments)
		}
		if skipMigrate || verboseSQL {
			t.Fatalf("parseSeedArgs(nil) skipMigrate/verboseSQL = %v/%v, want false/false", skipMigrate, verboseSQL)
		}
	})

	t.Run("parses overrides", func(t *testing.T) {
		opts, skipMigrate, verboseSQL, err := parseSeedArgs([]string{
			"--baseline=false",
			"--bulk=false",
			"--users=2",
			"--clinics=3",
			"--patients=4",
			"--appointments=5",
			"--owner-email=owner@example.com",
			"--ensure-owner",
			"--skip-migrate",
			"--verbose-sql",
		})
		if err != nil {
			t.Fatalf("parseSeedArgs(overrides) unexpected error: %v", err)
		}
		if opts.Baseline || opts.RandomizedBulk {
			t.Fatalf("parseSeedArgs(overrides) baseline/bulk = %v/%v, want false/false", opts.Baseline, opts.RandomizedBulk)
		}
		if opts.BulkUsers != 2 || opts.BulkClinics != 3 || opts.BulkPatients != 4 || opts.BulkAppointments != 5 {
			t.Fatalf("parseSeedArgs(overrides) got bulk values users=%d clinics=%d patients=%d appointments=%d", opts.BulkUsers, opts.BulkClinics, opts.BulkPatients, opts.BulkAppointments)
		}
		if opts.OwnerEmail != "owner@example.com" || !opts.EnsureOwner {
			t.Fatalf("parseSeedArgs(overrides) owner values = %q/%v", opts.OwnerEmail, opts.EnsureOwner)
		}
		if !skipMigrate || !verboseSQL {
			t.Fatalf("parseSeedArgs(overrides) skipMigrate/verboseSQL = %v/%v, want true/true", skipMigrate, verboseSQL)
		}
	})

	t.Run("returns error for invalid args", func(t *testing.T) {
		_, _, _, err := parseSeedArgs([]string{"--users=not-a-number"})
		if err == nil {
			t.Fatal("parseSeedArgs(invalid) expected error")
		}
	})
}

func TestRunSeed(t *testing.T) {
	t.Run("success path with migrations and quiet sql", func(t *testing.T) {
		testDB := newTestSeedDB(t)

		migrationsCalled := false
		seedCalled := false
		steps := []string{}

		err := runSeed(context.Background(), []string{"--owner-email=owner@example.com", "--ensure-owner"}, seedRuntimeDeps{
			loadDotEnv: func(...string) error { return nil },
			loadEnv: func() (*appenv.Env, error) {
				return &appenv.Env{}, nil
			},
			openDB: func(*appenv.Env) (*database.Database, error) {
				return testDB, nil
			},
			applyMigrations: func(*database.Database, *appenv.Env) error {
				migrationsCalled = true
				return nil
			},
			runSeeder: func(_ context.Context, db *gorm.DB, opts seeder.Options) (seeder.Result, error) {
				seedCalled = true
				if db == nil {
					t.Fatal("runSeed() passed nil db to seeder")
				}
				if opts.OwnerEmail != "owner@example.com" || !opts.EnsureOwner {
					t.Fatalf("runSeed() owner options = %q/%v", opts.OwnerEmail, opts.EnsureOwner)
				}
				return seeder.Result{}, nil
			},
			logPrintf: func(string, ...any) {},
			logStep: func(_ string, _ string, msg string) {
				steps = append(steps, msg)
			},
		})
		if err != nil {
			t.Fatalf("runSeed() unexpected error: %v", err)
		}
		if !migrationsCalled {
			t.Fatal("runSeed() expected migrations to run")
		}
		if !seedCalled {
			t.Fatal("runSeed() expected seeder to run")
		}
		if !containsStep(steps, "Applying schema migrations") {
			t.Fatalf("runSeed() expected apply-migrations step, got %v", steps)
		}
		if !containsStep(steps, "SQL logs suppressed (use --verbose-sql to enable)") {
			t.Fatalf("runSeed() expected quiet SQL step, got %v", steps)
		}
	})

	t.Run("success path with skip migrate", func(t *testing.T) {
		testDB := newTestSeedDB(t)

		migrationsCalled := false
		seedCalled := false
		steps := []string{}

		err := runSeed(context.Background(), []string{"--skip-migrate", "--verbose-sql", "--owner-email=owner@example.com", "--ensure-owner"}, seedRuntimeDeps{
			loadDotEnv: func(...string) error { return nil },
			loadEnv: func() (*appenv.Env, error) {
				return &appenv.Env{}, nil
			},
			openDB: func(*appenv.Env) (*database.Database, error) {
				return testDB, nil
			},
			applyMigrations: func(*database.Database, *appenv.Env) error {
				migrationsCalled = true
				return nil
			},
			runSeeder: func(_ context.Context, db *gorm.DB, opts seeder.Options) (seeder.Result, error) {
				seedCalled = true
				if db == nil {
					t.Fatal("runSeed() passed nil db to seeder")
				}
				if opts.OwnerEmail != "owner@example.com" || !opts.EnsureOwner {
					t.Fatalf("runSeed() owner options = %q/%v", opts.OwnerEmail, opts.EnsureOwner)
				}
				return seeder.Result{UsersCreated: 1, ClinicsCreated: 2, PatientsCreated: 3, AppointmentsCreated: 4}, nil
			},
			logPrintf: func(string, ...any) {},
			logStep: func(_ string, _ string, msg string) {
				steps = append(steps, msg)
			},
		})
		if err != nil {
			t.Fatalf("runSeed() unexpected error: %v", err)
		}
		if migrationsCalled {
			t.Fatal("runSeed() expected migrations to be skipped")
		}
		if !seedCalled {
			t.Fatal("runSeed() expected seeder to run")
		}
		if !containsStep(steps, "Skipping migrations (--skip-migrate)") {
			t.Fatalf("runSeed() expected skip-migrate step, got %v", steps)
		}
		if !containsStep(steps, "SQL logs enabled (--verbose-sql)") {
			t.Fatalf("runSeed() expected verbose SQL step, got %v", steps)
		}
	})

	t.Run("returns migration error", func(t *testing.T) {
		testDB := newTestSeedDB(t)
		expectedErr := errors.New("migrate failed")

		err := runSeed(context.Background(), nil, seedRuntimeDeps{
			loadDotEnv: func(...string) error { return nil },
			loadEnv: func() (*appenv.Env, error) {
				return &appenv.Env{}, nil
			},
			openDB: func(*appenv.Env) (*database.Database, error) {
				return testDB, nil
			},
			applyMigrations: func(*database.Database, *appenv.Env) error {
				return expectedErr
			},
			runSeeder: func(context.Context, *gorm.DB, seeder.Options) (seeder.Result, error) {
				return seeder.Result{}, nil
			},
			logPrintf: func(string, ...any) {},
			logStep:   func(string, string, string) {},
		})
		if err == nil {
			t.Fatal("runSeed() expected error")
		}
		if !errors.Is(err, expectedErr) {
			t.Fatalf("runSeed() error = %v, want wrapped %v", err, expectedErr)
		}
	})

	t.Run("returns seeder error", func(t *testing.T) {
		testDB := newTestSeedDB(t)
		expectedErr := errors.New("seed failed")

		err := runSeed(context.Background(), []string{"--skip-migrate"}, seedRuntimeDeps{
			loadDotEnv: func(...string) error { return nil },
			loadEnv: func() (*appenv.Env, error) {
				return &appenv.Env{}, nil
			},
			openDB: func(*appenv.Env) (*database.Database, error) {
				return testDB, nil
			},
			applyMigrations: func(*database.Database, *appenv.Env) error {
				return nil
			},
			runSeeder: func(context.Context, *gorm.DB, seeder.Options) (seeder.Result, error) {
				return seeder.Result{}, expectedErr
			},
			logPrintf: func(string, ...any) {},
			logStep:   func(string, string, string) {},
		})
		if err == nil {
			t.Fatal("runSeed() expected error")
		}
		if !errors.Is(err, expectedErr) {
			t.Fatalf("runSeed() error = %v, want wrapped %v", err, expectedErr)
		}
	})

	t.Run("returns env load error", func(t *testing.T) {
		expectedErr := errors.New("env failed")

		err := runSeed(context.Background(), nil, seedRuntimeDeps{
			loadDotEnv: func(...string) error { return nil },
			loadEnv: func() (*appenv.Env, error) {
				return nil, expectedErr
			},
			openDB: func(*appenv.Env) (*database.Database, error) {
				return nil, nil
			},
			applyMigrations: func(*database.Database, *appenv.Env) error { return nil },
			runSeeder: func(context.Context, *gorm.DB, seeder.Options) (seeder.Result, error) {
				return seeder.Result{}, nil
			},
			logPrintf: func(string, ...any) {},
			logStep:   func(string, string, string) {},
		})
		if err == nil {
			t.Fatal("runSeed() expected error")
		}
		if !errors.Is(err, expectedErr) {
			t.Fatalf("runSeed() error = %v, want wrapped %v", err, expectedErr)
		}
	})

	t.Run("returns db open error", func(t *testing.T) {
		expectedErr := errors.New("db failed")

		err := runSeed(context.Background(), nil, seedRuntimeDeps{
			loadDotEnv: func(...string) error { return nil },
			loadEnv: func() (*appenv.Env, error) {
				return &appenv.Env{}, nil
			},
			openDB: func(*appenv.Env) (*database.Database, error) {
				return nil, expectedErr
			},
			applyMigrations: func(*database.Database, *appenv.Env) error { return nil },
			runSeeder: func(context.Context, *gorm.DB, seeder.Options) (seeder.Result, error) {
				return seeder.Result{}, nil
			},
			logPrintf: func(string, ...any) {},
			logStep:   func(string, string, string) {},
		})
		if err == nil {
			t.Fatal("runSeed() expected error")
		}
		if !errors.Is(err, expectedErr) {
			t.Fatalf("runSeed() error = %v, want wrapped %v", err, expectedErr)
		}
	})
}

func TestDefaultSeedRuntimeDeps(t *testing.T) {
	deps := defaultSeedRuntimeDeps()

	if deps.loadDotEnv == nil || deps.loadEnv == nil || deps.openDB == nil || deps.applyMigrations == nil || deps.runSeeder == nil || deps.logPrintf == nil || deps.logStep == nil {
		t.Fatal("defaultSeedRuntimeDeps() expected all dependency functions to be non-nil")
	}

	env := &appenv.Env{Environment: appenv.EnvironmentDevelopment, DBPath: filepath.Join(t.TempDir(), "seed-default.sqlite")}
	db, err := deps.openDB(env)
	if err != nil {
		t.Fatalf("defaultSeedRuntimeDeps().openDB() unexpected error: %v", err)
	}
	if db == nil {
		t.Fatal("defaultSeedRuntimeDeps().openDB() expected non-nil db")
	}
	if err := db.Close(); err != nil {
		t.Fatalf("defaultSeedRuntimeDeps().openDB() close error: %v", err)
	}
}

func newTestSeedDB(t *testing.T) *database.Database {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:seed_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	seedDB := &database.Database{DB: db}
	t.Cleanup(func() {
		_ = seedDB.Close()
	})

	return seedDB
}

func containsStep(steps []string, want string) bool {
	for _, step := range steps {
		if step == want {
			return true
		}
	}

	return false
}

func TestLogStep(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	original := logStepWriter
	defer func() { logStepWriter = original }()

	b := &stringsBuilder{}
	logStepWriter = b
	logStep("i", ansiBlue, "hello")

	if got, want := b.String(), "i hello\n"; got != want {
		t.Fatalf("logStep output = %q, want %q", got, want)
	}
}

func TestMain(t *testing.T) {
	t.Run("does not exit on successful run", func(t *testing.T) {
		restore := installSeedMainTestHooks()
		defer restore()

		exited := false
		runSeedForMain = func(context.Context, []string, seedRuntimeDeps) error { return nil }
		seedExitForMain = func(int) { exited = true }

		main()

		if exited {
			t.Fatal("main() should not call exit on success")
		}
	})

	t.Run("logs and exits on failure", func(t *testing.T) {
		restore := installSeedMainTestHooks()
		defer restore()

		exitCode := 0
		logCalled := false
		runSeedForMain = func(context.Context, []string, seedRuntimeDeps) error { return errors.New("boom") }
		seedExitForMain = func(code int) { exitCode = code }
		seedLogForMain = func(string, ...any) { logCalled = true }

		main()

		if !logCalled {
			t.Fatal("main() expected log call on failure")
		}
		if exitCode != 1 {
			t.Fatalf("main() exit code = %d, want 1", exitCode)
		}
	})
}

type stringsBuilder struct {
	b []byte
}

func (s *stringsBuilder) Write(p []byte) (int, error) {
	s.b = append(s.b, p...)
	return len(p), nil
}

func (s *stringsBuilder) String() string {
	return string(s.b)
}

func installSeedMainTestHooks() func() {
	oldRunSeed := runSeedForMain
	oldExit := seedExitForMain
	oldLog := seedLogForMain

	return func() {
		runSeedForMain = oldRunSeed
		seedExitForMain = oldExit
		seedLogForMain = oldLog
	}
}
