package seeder

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"miconsul/internal/lib/avatar"
	"miconsul/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	seedAdminEmail    = "admin@seed.local"
	seedAdminPassword = "Admin123!"
	seedAdminName     = "Seed Admin"
	seedOwnerPassword = "SeedOwner123!"

	baselineClinicExtID = "seed-baseline-clinic-main"
)

var baselinePatientExtIDs = []string{
	"seed-baseline-patient-alma",
	"seed-baseline-patient-bruno",
	"seed-baseline-patient-carla",
}

var baselineAppointmentExtIDs = []string{
	"seed-baseline-appt-1",
	"seed-baseline-appt-2",
	"seed-baseline-appt-3",
}

func Run(ctx context.Context, db *gorm.DB, opts Options) (Result, error) {
	if db == nil {
		return Result{}, errors.New("db is nil")
	}

	opts = opts.withDefaults()
	result := Result{}

	ownerUser, baselineResult, err := seedBaseline(ctx, db, opts)
	if err != nil {
		return Result{}, err
	}
	result.add(baselineResult)

	if opts.RandomizedBulk {
		bulkResult, err := seedBulk(ctx, db, ownerUser, opts)
		if err != nil {
			return Result{}, err
		}
		result.add(bulkResult)
	}

	return result, nil
}

func seedBaseline(ctx context.Context, db *gorm.DB, opts Options) (model.User, Result, error) {
	ownerUser, createdOwner, err := resolveOwnerUser(ctx, db, opts)
	if err != nil {
		return model.User{}, Result{}, err
	}

	result := Result{}
	if createdOwner {
		result.UsersCreated++
	}

	if !opts.Baseline {
		return ownerUser, result, nil
	}

	clinic, createdClinic, err := ensureBaselineClinic(ctx, db, ownerUser)
	if err != nil {
		return model.User{}, Result{}, err
	}
	if createdClinic {
		result.ClinicsCreated++
	}

	patients, createdPatients, err := ensureBaselinePatients(ctx, db, ownerUser)
	if err != nil {
		return model.User{}, Result{}, err
	}
	result.PatientsCreated += createdPatients

	createdAppointments, err := ensureBaselineAppointments(ctx, db, ownerUser, clinic, patients)
	if err != nil {
		return model.User{}, Result{}, err
	}
	result.AppointmentsCreated += createdAppointments

	return ownerUser, result, nil
}

func resolveOwnerUser(ctx context.Context, db *gorm.DB, opts Options) (model.User, bool, error) {
	ownerEmail := strings.TrimSpace(opts.OwnerEmail)
	if ownerEmail == "" {
		return ensureAdminUser(ctx, db)
	}

	user := model.User{}
	err := db.WithContext(ctx).Where("email = ?", ownerEmail).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if !opts.EnsureOwner {
			return model.User{}, false, fmt.Errorf("owner user not found: %s", ownerEmail)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(seedOwnerPassword), 12)
		if err != nil {
			return model.User{}, false, fmt.Errorf("hash seed owner password: %w", err)
		}

		user = model.User{
			Name:              "Seed Owner",
			Email:             ownerEmail,
			Password:          string(hashedPassword),
			ProfilePic:        avatar.DicebearAvatarURL(ownerEmail),
			Role:              model.UserRoleUser,
			ConfirmEmailToken: "",
		}

		if err := db.WithContext(ctx).Create(&user).Error; err != nil {
			return model.User{}, false, fmt.Errorf("create seed owner user: %w", err)
		}

		return user, true, nil
	}
	if err != nil {
		return model.User{}, false, fmt.Errorf("find seed owner user: %w", err)
	}

	return user, false, nil
}

func ensureAdminUser(ctx context.Context, db *gorm.DB) (model.User, bool, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(seedAdminPassword), 12)
	if err != nil {
		return model.User{}, false, fmt.Errorf("hash seed admin password: %w", err)
	}

	user := model.User{}
	err = db.WithContext(ctx).Where("email = ?", seedAdminEmail).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user = model.User{
			Name:              seedAdminName,
			Email:             seedAdminEmail,
			Password:          string(hashedPassword),
			ProfilePic:        avatar.DicebearAvatarURL(seedAdminEmail),
			Role:              model.UserRoleAdmin,
			ConfirmEmailToken: "",
		}
		if err := db.WithContext(ctx).Create(&user).Error; err != nil {
			return model.User{}, false, fmt.Errorf("create seed admin user: %w", err)
		}

		return user, true, nil
	}
	if err != nil {
		return model.User{}, false, fmt.Errorf("find seed admin user: %w", err)
	}

	updates := map[string]any{
		"name":                     seedAdminName,
		"role":                     model.UserRoleAdmin,
		"password":                 string(hashedPassword),
		"profile_pic":              avatar.DicebearAvatarURL(seedAdminEmail),
		"confirm_email_token":      "",
		"confirm_email_expires_at": time.Time{},
	}

	if err := db.WithContext(ctx).Model(&user).Updates(updates).Error; err != nil {
		return model.User{}, false, fmt.Errorf("update seed admin user: %w", err)
	}

	if err := db.WithContext(ctx).Where("id = ?", user.ID).Take(&user).Error; err != nil {
		return model.User{}, false, fmt.Errorf("reload seed admin user: %w", err)
	}

	return user, false, nil
}

func ensureBaselineClinic(ctx context.Context, db *gorm.DB, owner model.User) (model.Clinic, bool, error) {
	clinic := model.Clinic{}
	err := db.WithContext(ctx).Where("ext_id = ?", baselineClinicExtID).Take(&clinic).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		clinic = model.Clinic{
			ExtID:      baselineClinicExtID,
			UserID:     owner.ID,
			Name:       "Seed Family Clinic",
			Email:      "clinic-main@seed.local",
			Phone:      "+52-55-1111-2222",
			ProfilePic: avatar.DicebearShapeAvatarURL(baselineClinicExtID),
			Address:    model.Address{City: "Monterrey", State: "NL", Country: "MX"},
		}
		if err := db.WithContext(ctx).Create(&clinic).Error; err != nil {
			return model.Clinic{}, false, fmt.Errorf("create baseline clinic: %w", err)
		}

		return clinic, true, nil
	}
	if err != nil {
		return model.Clinic{}, false, fmt.Errorf("find baseline clinic: %w", err)
	}

	updates := map[string]any{
		"user_id":     owner.ID,
		"name":        "Seed Family Clinic",
		"email":       "clinic-main@seed.local",
		"phone":       "+52-55-1111-2222",
		"profile_pic": avatar.DicebearShapeAvatarURL(baselineClinicExtID),
		"city":        "Monterrey",
		"state":       "NL",
		"country":     "MX",
	}
	if err := db.WithContext(ctx).Model(&clinic).Updates(updates).Error; err != nil {
		return model.Clinic{}, false, fmt.Errorf("update baseline clinic: %w", err)
	}

	if err := db.WithContext(ctx).Where("id = ?", clinic.ID).Take(&clinic).Error; err != nil {
		return model.Clinic{}, false, fmt.Errorf("reload baseline clinic: %w", err)
	}

	return clinic, false, nil
}

func ensureBaselinePatients(ctx context.Context, db *gorm.DB, owner model.User) ([]model.Patient, int, error) {
	basePatients := []model.Patient{
		{
			ExtID:      baselinePatientExtIDs[0],
			UserID:     owner.ID,
			Name:       "Alma Rivera",
			Email:      "alma@seed.local",
			Phone:      "+52-55-2000-1001",
			ProfilePic: avatar.PravatarURL(baselinePatientExtIDs[0]),
			Age:        31,
		},
		{
			ExtID:      baselinePatientExtIDs[1],
			UserID:     owner.ID,
			Name:       "Bruno Chavez",
			Email:      "bruno@seed.local",
			Phone:      "+52-55-2000-1002",
			ProfilePic: avatar.PravatarURL(baselinePatientExtIDs[1]),
			Age:        42,
		},
		{
			ExtID:      baselinePatientExtIDs[2],
			UserID:     owner.ID,
			Name:       "Carla Medina",
			Email:      "carla@seed.local",
			Phone:      "+52-55-2000-1003",
			ProfilePic: avatar.PravatarURL(baselinePatientExtIDs[2]),
			Age:        28,
		},
	}

	patients := make([]model.Patient, 0, len(basePatients))
	created := 0

	for _, desired := range basePatients {
		patient := model.Patient{}
		err := db.WithContext(ctx).Where("ext_id = ?", desired.ExtID).Take(&patient).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := db.WithContext(ctx).Create(&desired).Error; err != nil {
				return nil, 0, fmt.Errorf("create baseline patient %s: %w", desired.ExtID, err)
			}
			patients = append(patients, desired)
			created++
			continue
		}
		if err != nil {
			return nil, 0, fmt.Errorf("find baseline patient %s: %w", desired.ExtID, err)
		}

		updates := map[string]any{
			"user_id":     owner.ID,
			"name":        desired.Name,
			"email":       desired.Email,
			"phone":       desired.Phone,
			"profile_pic": desired.ProfilePic,
			"age":         desired.Age,
		}
		if err := db.WithContext(ctx).Model(&patient).Updates(updates).Error; err != nil {
			return nil, 0, fmt.Errorf("update baseline patient %s: %w", desired.ExtID, err)
		}
		if err := db.WithContext(ctx).Where("id = ?", patient.ID).Take(&patient).Error; err != nil {
			return nil, 0, fmt.Errorf("reload baseline patient %s: %w", desired.ExtID, err)
		}

		patients = append(patients, patient)
	}

	return patients, created, nil
}

func ensureBaselineAppointments(ctx context.Context, db *gorm.DB, owner model.User, clinic model.Clinic, patients []model.Patient) (int, error) {
	if len(patients) == 0 {
		return 0, nil
	}

	bookedAt := []time.Time{
		time.Date(2026, 1, 10, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 11, 11, 30, 0, 0, time.UTC),
		time.Date(2026, 1, 12, 16, 15, 0, 0, time.UTC),
	}

	created := 0
	for i, extID := range baselineAppointmentExtIDs {
		apnt := model.Appointment{}
		err := db.WithContext(ctx).Where("ext_id = ?", extID).Take(&apnt).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			candidate := model.Appointment{
				ExtID:     extID,
				UserID:    owner.ID,
				ClinicID:  clinic.ID,
				PatientID: patients[i%len(patients)].ID,
				BookedAt:  bookedAt[i%len(bookedAt)],
				Timezone:  model.DefaultTimezone,
				Status:    model.ApntStatusConfirmed,
				Summary:   fmt.Sprintf("Baseline follow-up #%d", i+1),
				Token:     fmt.Sprintf("seed-baseline-token-%d", i+1),
			}
			if err := db.WithContext(ctx).Create(&candidate).Error; err != nil {
				return 0, fmt.Errorf("create baseline appointment %s: %w", extID, err)
			}
			created++
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("find baseline appointment %s: %w", extID, err)
		}

		updates := map[string]any{
			"user_id":    owner.ID,
			"clinic_id":  clinic.ID,
			"patient_id": patients[i%len(patients)].ID,
			"booked_at":  bookedAt[i%len(bookedAt)],
			"timezone":   model.DefaultTimezone,
			"status":     model.ApntStatusConfirmed,
			"summary":    fmt.Sprintf("Baseline follow-up #%d", i+1),
		}
		if err := db.WithContext(ctx).Model(&apnt).Updates(updates).Error; err != nil {
			return 0, fmt.Errorf("update baseline appointment %s: %w", extID, err)
		}
	}

	return created, nil
}

func seedBulk(ctx context.Context, db *gorm.DB, owner model.User, opts Options) (Result, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	runID := time.Now().Unix()

	result := Result{}

	createdUsers, err := createBulkUsers(ctx, db, rng, runID, opts.BulkUsers)
	if err != nil {
		return Result{}, err
	}
	result.UsersCreated += createdUsers

	clinics, createdClinics, err := createBulkClinics(ctx, db, owner, rng, runID, opts.BulkClinics)
	if err != nil {
		return Result{}, err
	}
	result.ClinicsCreated += createdClinics

	patients, createdPatients, err := createBulkPatients(ctx, db, owner, rng, runID, opts.BulkPatients)
	if err != nil {
		return Result{}, err
	}
	result.PatientsCreated += createdPatients

	createdAppointments, err := createBulkAppointments(ctx, db, owner, rng, runID, clinics, patients, opts.BulkAppointments)
	if err != nil {
		return Result{}, err
	}
	result.AppointmentsCreated += createdAppointments

	return result, nil
}

func createBulkUsers(ctx context.Context, db *gorm.DB, rng *rand.Rand, runID int64, count int) (int, error) {
	if count <= 0 {
		return 0, nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("SeedUser123!"), 12)
	if err != nil {
		return 0, fmt.Errorf("hash bulk user password: %w", err)
	}

	users := make([]model.User, 0, count)
	for i := 0; i < count; i++ {
		users = append(users, model.User{
			Name:              fmt.Sprintf("Seed User %d", i+1),
			Email:             fmt.Sprintf("seed.user.%d.%d@seed.local", runID, i+1),
			Password:          string(hashedPassword),
			ProfilePic:        avatar.DicebearFunEmojiAvatarURL(fmt.Sprintf("seed-user-%d-%d", runID, i+1)),
			Role:              model.UserRoleUser,
			Timezone:          "America/Mexico_City",
			Phone:             fmt.Sprintf("+52-55-%04d-%04d", rng.Intn(10000), rng.Intn(10000)),
			ConfirmEmailToken: "",
		})
	}

	if err := db.WithContext(ctx).Create(&users).Error; err != nil {
		return 0, fmt.Errorf("create bulk users: %w", err)
	}

	return len(users), nil
}

func createBulkClinics(ctx context.Context, db *gorm.DB, owner model.User, rng *rand.Rand, runID int64, count int) ([]model.Clinic, int, error) {
	if count <= 0 {
		return nil, 0, nil
	}

	clinics := make([]model.Clinic, 0, count)
	for i := 0; i < count; i++ {
		extID := fmt.Sprintf("seed-bulk-clinic-%d-%d", runID, i+1)
		clinics = append(clinics, model.Clinic{
			ExtID:      extID,
			UserID:     owner.ID,
			Name:       fmt.Sprintf("Seed Clinic %d", i+1),
			Email:      fmt.Sprintf("clinic.%d.%d@seed.local", runID, i+1),
			Phone:      fmt.Sprintf("+52-81-%04d-%04d", rng.Intn(10000), rng.Intn(10000)),
			ProfilePic: avatar.DicebearShapeAvatarURL(extID),
			Address:    model.Address{City: "Monterrey", State: "NL", Country: "MX"},
		})
	}

	if err := db.WithContext(ctx).Create(&clinics).Error; err != nil {
		return nil, 0, fmt.Errorf("create bulk clinics: %w", err)
	}

	return clinics, len(clinics), nil
}

func createBulkPatients(ctx context.Context, db *gorm.DB, owner model.User, rng *rand.Rand, runID int64, count int) ([]model.Patient, int, error) {
	if count <= 0 {
		return nil, 0, nil
	}

	patients := make([]model.Patient, 0, count)
	for i := 0; i < count; i++ {
		patients = append(patients, model.Patient{
			ExtID:      fmt.Sprintf("seed-bulk-patient-%d-%d", runID, i+1),
			UserID:     owner.ID,
			Name:       fmt.Sprintf("Seed Patient %d", i+1),
			Email:      fmt.Sprintf("patient.%d.%d@seed.local", runID, i+1),
			Phone:      fmt.Sprintf("+52-55-%04d-%04d", rng.Intn(10000), rng.Intn(10000)),
			ProfilePic: avatar.PravatarURL(fmt.Sprintf("seed-bulk-patient-%d-%d", runID, i+1)),
			Age:        18 + rng.Intn(63),
		})
	}

	if err := db.WithContext(ctx).Create(&patients).Error; err != nil {
		return nil, 0, fmt.Errorf("create bulk patients: %w", err)
	}

	return patients, len(patients), nil
}

func createBulkAppointments(ctx context.Context, db *gorm.DB, owner model.User, rng *rand.Rand, runID int64, clinics []model.Clinic, patients []model.Patient, count int) (int, error) {
	if count <= 0 || len(clinics) == 0 || len(patients) == 0 {
		return 0, nil
	}

	appointments := make([]model.Appointment, 0, count)
	baseTime := time.Now().UTC().Add(24 * time.Hour)

	for i := 0; i < count; i++ {
		clinic := clinics[rng.Intn(len(clinics))]
		patient := patients[rng.Intn(len(patients))]

		appointments = append(appointments, model.Appointment{
			ExtID:     fmt.Sprintf("seed-bulk-appointment-%d-%d", runID, i+1),
			UserID:    owner.ID,
			ClinicID:  clinic.ID,
			PatientID: patient.ID,
			BookedAt:  baseTime.Add(time.Duration(i) * 45 * time.Minute),
			Timezone:  model.DefaultTimezone,
			Duration:  45,
			Status:    model.ApntStatusPending,
			Summary:   fmt.Sprintf("Seed appointment #%d", i+1),
			Token:     fmt.Sprintf("seed-bulk-token-%d-%d", runID, i+1),
		})
	}

	if err := db.WithContext(ctx).Create(&appointments).Error; err != nil {
		return 0, fmt.Errorf("create bulk appointments: %w", err)
	}

	return len(appointments), nil
}
