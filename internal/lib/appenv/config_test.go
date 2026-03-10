package appenv

import "testing"

func TestEnvHelpers(t *testing.T) {
	t.Run("nil env returns false for all helpers", func(t *testing.T) {
		var env *Env
		if env.IsDevelopment() {
			t.Fatalf("expected IsDevelopment false for nil env")
		}
		if env.IsTest() {
			t.Fatalf("expected IsTest false for nil env")
		}
		if env.IsDevOrTest() {
			t.Fatalf("expected IsDevOrTest false for nil env")
		}
		if env.IsProduction() {
			t.Fatalf("expected IsProduction false for nil env")
		}
		if env.IsValidEnvironment() {
			t.Fatalf("expected IsValidEnvironment false for nil env")
		}
	})

	t.Run("environment helpers map correctly", func(t *testing.T) {
		env := &Env{Environment: EnvironmentDevelopment}
		if !env.IsDevelopment() || !env.IsDevOrTest() {
			t.Fatalf("expected development helpers true")
		}
		if env.IsProduction() || env.IsTest() {
			t.Fatalf("expected non-production/test helpers false")
		}

		env.Environment = EnvironmentTest
		if !env.IsTest() || !env.IsDevOrTest() {
			t.Fatalf("expected test helpers true")
		}

		env.Environment = EnvironmentProduction
		if !env.IsProduction() {
			t.Fatalf("expected IsProduction true")
		}
		if env.IsDevOrTest() {
			t.Fatalf("expected IsDevOrTest false in production")
		}

		env.Environment = "invalid"
		if env.IsValidEnvironment() {
			t.Fatalf("expected invalid environment to fail validation")
		}
	})
}
