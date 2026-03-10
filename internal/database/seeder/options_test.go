package seeder

import (
	"testing"
)

func TestOptionsWithDefaults(t *testing.T) {
	opts := Options{
		BulkUsers:        -1,
		BulkClinics:      -2,
		BulkPatients:     -3,
		BulkAppointments: -4,
	}

	got := opts.withDefaults()
	if !got.Baseline {
		t.Fatalf("expected baseline to default to true")
	}
	if got.BulkUsers != 0 || got.BulkClinics != 0 || got.BulkPatients != 0 || got.BulkAppointments != 0 {
		t.Fatalf("expected negative bulk values to clamp to zero, got %#v", got)
	}
}

func TestResultHelpers(t *testing.T) {
	r := Result{UsersCreated: 1, ClinicsCreated: 2, PatientsCreated: 3, AppointmentsCreated: 4}
	r.add(Result{UsersCreated: 2, ClinicsCreated: 3, PatientsCreated: 4, AppointmentsCreated: 5})

	if r.UsersCreated != 3 || r.ClinicsCreated != 5 || r.PatientsCreated != 7 || r.AppointmentsCreated != 9 {
		t.Fatalf("unexpected aggregated result %#v", r)
	}

	if total := r.TotalCreated(); total != 24 {
		t.Fatalf("expected total created 24, got %d", total)
	}
}

func TestRunRejectsNilDB(t *testing.T) {
	if _, err := Run(t.Context(), nil, Options{}); err == nil {
		t.Fatalf("expected nil db guard error")
	}
}
