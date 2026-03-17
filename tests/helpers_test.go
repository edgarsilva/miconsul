package tests

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"miconsul/internal/database"
	"miconsul/internal/lib/appenv"
	"miconsul/internal/model"
	"miconsul/internal/observability/logging"
	"miconsul/internal/routes"
	"miconsul/internal/server"
	"miconsul/internal/services/auth"

	"go.opentelemetry.io/otel"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type testHarness struct {
	t      *testing.T
	server *server.Server
	db     *database.Database
	env    *appenv.Env
}

type requestOptions struct {
	method     string
	path       string
	body       url.Values
	accept     string
	htmx       bool
	authToken  string
	contentTyp string
}

func newTestHarness(t *testing.T) *testHarness {
	t.Helper()

	tmpDir := t.TempDir()
	useInMemoryDB := strings.EqualFold(strings.TrimSpace(os.Getenv("MICON_TEST_SQLITE_INMEMORY")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("MICON_TEST_SQLITE_INMEMORY")), "true")

	dbPath := filepath.Join(tmpDir, "app.db")
	dbOpts := database.SQLiteOptions(nil)
	if useInMemoryDB {
		dbPath = fmt.Sprintf("file:miconsul_test_%d", time.Now().UnixNano())
		dbOpts = database.SQLiteOptions{
			"mode":  "memory",
			"cache": "shared",
		}
	}

	env := &appenv.Env{
		Environment:        appenv.EnvironmentTest,
		AppName:            "miconsul-test",
		AppProtocol:        "http",
		AppDomain:          "localhost",
		AppPort:            3001,
		AppShutdownTimeout: 2 * time.Second,
		CookieSecret:       strings.Repeat("a", 32),
		JWTSecret:          strings.Repeat("b", 32),
		DBPath:             dbPath,
		SessionDBPath:      filepath.Join(tmpDir, "session.db"),
		AssetsDir:          tmpDir,
	}

	db, err := database.New(env, logging.Logger{}, dbOpts)
	if err != nil {
		t.Fatalf("create test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if useInMemoryDB {
		sqlDB, err := db.SQLDB()
		if err != nil {
			t.Fatalf("get sql db handle: %v", err)
		}
		if sqlDB != nil {
			sqlDB.SetMaxOpenConns(1)
			sqlDB.SetMaxIdleConns(1)
		}
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.Clinic{},
		&model.Patient{},
		&model.Appointment{},
		&model.Alert{},
		&model.FeedEvent{},
	); err != nil {
		t.Fatalf("auto migrate test schema: %v", err)
	}

	createdPublicDir := false
	if _, err := os.Stat("public"); os.IsNotExist(err) {
		createdPublicDir = true
	}
	if err := os.MkdirAll("public/.well-known", 0o755); err != nil {
		t.Fatalf("create test public dir: %v", err)
	}
	if err := os.WriteFile("public/favicon.ico", []byte("ico"), 0o644); err != nil {
		t.Fatalf("create test favicon: %v", err)
	}
	t.Cleanup(func() {
		if createdPublicDir {
			_ = os.RemoveAll("public")
			return
		}
		_ = os.Remove("public/favicon.ico")
	})

	s := server.New(
		server.WithEnv(env),
		server.WithDatabase(db),
		server.WithTracer(otel.Tracer("tests")),
	)

	if err := routes.RegisterServices(s); err != nil {
		t.Fatalf("register routes: %v", err)
	}

	return &testHarness{t: t, server: s, db: db, env: env}
}

func (h *testHarness) createUser(role model.UserRole) model.User {
	h.t.Helper()

	u := model.User{
		Name:  "Test User",
		Email: "test-" + time.Now().Format("150405.000000") + "@example.com",
		Role:  role,
	}
	if err := gorm.G[model.User](h.db.GormDB()).Create(h.t.Context(), &u); err != nil {
		h.t.Fatalf("create user fixture: %v", err)
	}

	return u
}

func (h *testHarness) createAuthUser(email, password string, role model.UserRole) model.User {
	h.t.Helper()

	if strings.TrimSpace(email) == "" {
		email = "auth-" + time.Now().Format("150405.000000") + "@example.com"
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		h.t.Fatalf("hash auth user password: %v", err)
	}

	u := model.User{
		Name:              "Auth User",
		Email:             email,
		Password:          string(hash),
		Role:              role,
		ConfirmEmailToken: "",
	}
	if err := gorm.G[model.User](h.db.GormDB()).Create(h.t.Context(), &u); err != nil {
		h.t.Fatalf("create auth user fixture: %v", err)
	}

	return u
}

func (h *testHarness) createPatient(userID string, name string) model.Patient {
	h.t.Helper()

	p := model.Patient{Name: name, Phone: "555-0100", Age: 30, UserID: userID}
	if err := gorm.G[model.Patient](h.db.GormDB()).Create(h.t.Context(), &p); err != nil {
		h.t.Fatalf("create patient fixture: %v", err)
	}

	return p
}

func (h *testHarness) createClinic(userID string, name string) model.Clinic {
	h.t.Helper()

	c := model.Clinic{Name: name, UserID: userID}
	if err := gorm.G[model.Clinic](h.db.GormDB()).Create(h.t.Context(), &c); err != nil {
		h.t.Fatalf("create clinic fixture: %v", err)
	}

	return c
}

func (h *testHarness) createAppointment(userID, patientID, clinicID string) model.Appointment {
	h.t.Helper()

	a := model.Appointment{
		UserID:      userID,
		PatientID:   patientID,
		ClinicID:    clinicID,
		Status:      model.ApntStatusPending,
		BookedAt:    time.Now().UTC().Add(2 * time.Hour),
		BookedYear:  time.Now().UTC().Year(),
		BookedMonth: int(time.Now().UTC().Month()),
		BookedDay:   time.Now().UTC().Day(),
		BookedHour:  time.Now().UTC().Hour(),
	}
	if err := gorm.G[model.Appointment](h.db.GormDB()).Create(h.t.Context(), &a); err != nil {
		h.t.Fatalf("create appointment fixture: %v", err)
	}

	return a
}

func (h *testHarness) authToken(user model.User) string {
	h.t.Helper()

	token, err := auth.JWTCreateToken(h.env, user.Email, user.ID)
	if err != nil {
		h.t.Fatalf("create auth token: %v", err)
	}

	return token
}

func (h *testHarness) doRequest(opt requestOptions) (*http.Response, string) {
	h.t.Helper()

	method := opt.method
	if method == "" {
		method = http.MethodGet
	}

	body := io.Reader(nil)
	if opt.body != nil {
		body = strings.NewReader(opt.body.Encode())
	}

	req, err := http.NewRequest(method, opt.path, body)
	if err != nil {
		h.t.Fatalf("create request: %v", err)
	}

	accept := opt.accept
	if accept == "" {
		accept = "text/html"
	}
	req.Header.Set("Accept", accept)

	contentType := opt.contentTyp
	if contentType == "" && opt.body != nil {
		contentType = "application/x-www-form-urlencoded"
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if opt.htmx {
		req.Header.Set("HX-Request", "true")
	}

	if opt.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+opt.authToken)
	}

	resp, err := h.server.Test(req)
	if err != nil {
		h.t.Fatalf("execute request: %v", err)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		h.t.Fatalf("read response body: %v", err)
	}

	return resp, string(bodyBytes)
}
