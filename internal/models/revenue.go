package models

import "time"

// RevenueReport is a user's revenue from done appointments over the last 12
// months, newest first, and the selected month's revenue by clinic, highest
// first. Amounts are in cents.
type RevenueReport struct {
	Months   []RevenueMonth
	Selected RevenueMonth
	Clinics  []RevenueClinic
	// MonthlyAverage is the average revenue of the 12 complete months before
	// the current one, e.g. 3000000 over 12 months -> 250000.
	MonthlyAverage int
}

// RevenueMonth is the revenue of done appointments booked in one month, in
// cents. Start is the first day of the month in the clinic timezone.
type RevenueMonth struct {
	Start        time.Time
	Total        int
	Appointments int
}

// RevenueClinic is one clinic's revenue in the selected month, in cents.
type RevenueClinic struct {
	Name         string
	Deleted      bool
	Total        int
	Appointments int
}

// SharePercent returns the clinic's revenue as a whole percent of total,
// rounded half up, e.g. 650000 of 1850000 -> 35.
func (c RevenueClinic) SharePercent(total int) int {
	if total == 0 {
		return 0
	}

	return (c.Total*100 + total/2) / total
}
