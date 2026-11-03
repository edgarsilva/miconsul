package models

import "testing"

func TestRevenueClinicSharePercent(t *testing.T) {
	cases := []struct {
		total, of, want int
	}{
		{650000, 1850000, 35},
		{1200000, 1850000, 65},
		{1, 3, 33},
		{2, 3, 67},
		{100, 0, 0},
	}
	for _, tc := range cases {
		if got := (RevenueClinic{Total: tc.total}).SharePercent(tc.of); got != tc.want {
			t.Errorf("SharePercent(%d of %d) = %d, want %d", tc.total, tc.of, got, tc.want)
		}
	}
}

func TestRevenuePatientSharePercent(t *testing.T) {
	if got := (RevenuePatient{Total: 650000}).SharePercent(1850000); got != 35 {
		t.Errorf("SharePercent = %d, want 35", got)
	}
	if got := (RevenuePatient{Total: 100}).SharePercent(0); got != 0 {
		t.Errorf("SharePercent of nothing = %d, want 0", got)
	}
}
