package seeder

type Options struct {
	Baseline         bool
	RandomizedBulk   bool
	BulkUsers        int
	BulkClinics      int
	BulkPatients     int
	BulkAppointments int
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

	return o
}

type Result struct {
	UsersCreated        int
	ClinicsCreated      int
	PatientsCreated     int
	AppointmentsCreated int
}

func (r *Result) add(other Result) {
	r.UsersCreated += other.UsersCreated
	r.ClinicsCreated += other.ClinicsCreated
	r.PatientsCreated += other.PatientsCreated
	r.AppointmentsCreated += other.AppointmentsCreated
}

func (r Result) TotalCreated() int {
	return r.UsersCreated + r.ClinicsCreated + r.PatientsCreated + r.AppointmentsCreated
}
