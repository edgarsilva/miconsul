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
