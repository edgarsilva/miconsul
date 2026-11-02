package seeder

type Options struct {
	Baseline         bool
	RandomizedBulk   bool
	BulkUsers        int
	BulkClinics      int
	BulkPatients     int
	BulkAppointments int
	// BulkPastAppointments are booked over the last BulkPastMonths months up to
	// now, priced, and mostly done, so revenue has data.
	BulkPastAppointments int
	// BulkPastMonths is how many months, the current one included, the past
	// appointments spread over; below 1 means 1.
	BulkPastMonths int
	OwnerEmail     string
	EnsureOwner    bool
}

func (o Options) withDefaults() Options {
	if !o.Baseline && !o.RandomizedBulk {
		o.Baseline = true
	}

	if o.BulkUsers < 0 {
		o.BulkUsers = 0
	}
	if o.BulkClinics < 0 {
		o.BulkClinics = 0
	}
	if o.BulkPatients < 0 {
		o.BulkPatients = 0
	}
	if o.BulkAppointments < 0 {
		o.BulkAppointments = 0
	}
	if o.BulkPastAppointments < 0 {
		o.BulkPastAppointments = 0
	}
	if o.BulkPastMonths < 1 {
		o.BulkPastMonths = 1
	}

	return o
}

type Result struct {
	UsersCreated        int
	ClinicsCreated      int
	PatientsCreated     int
	AppointmentsCreated int
	FeedEventsCreated   int
}

func (r *Result) add(other Result) {
	r.UsersCreated += other.UsersCreated
	r.ClinicsCreated += other.ClinicsCreated
	r.PatientsCreated += other.PatientsCreated
	r.AppointmentsCreated += other.AppointmentsCreated
	r.FeedEventsCreated += other.FeedEventsCreated
}

func (r Result) TotalCreated() int {
	return r.UsersCreated + r.ClinicsCreated + r.PatientsCreated + r.AppointmentsCreated + r.FeedEventsCreated
}
