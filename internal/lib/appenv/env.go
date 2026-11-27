// Package appenv provides Runtime configuration for the App
package appenv

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/edgarsilva/simpleenv"
)

type Env struct {
	Environment Environment `env:"APP_ENV;oneof=development,test,staging,production"`

	AppName     string `env:"APP_NAME"`
	AppProtocol string `env:"APP_PROTOCOL"`
	AppDomain   string `env:"APP_DOMAIN"`
	AppVersion  string `env:"APP_VERSION"`
	AppPort     int    `env:"APP_PORT;min=1;max=65535"`

	AppShutdownTimeout time.Duration `env:"APP_SHUTDOWN_TIMEOUT;optional;min=1s"`
	RateLimiterEnabled bool          `env:"RATE_LIMITER_ENABLED;optional"`

	CookieSecret string `env:"COOKIE_SECRET;regex=^(.{16}|.{24}|.{32})$"`
	JWTSecret    string `env:"JWT_SECRET"`

	DBPath        string `env:"DB_PATH"`
	SessionDBPath string `env:"SESSION_DB_PATH"`
	CacheDBPath   string `env:"CACHE_DB_PATH;optional"`

	DBMaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS;optional;min=1"`
	DBMaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS;optional;min=1"`
	DBConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME;optional;min=1s"`
	DBConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME;optional;min=1s"`

	RuntimeSampleInterval time.Duration `env:"RUNTIME_SAMPLE_INTERVAL;optional;min=1s"`
	RuntimeSampleSize     int           `env:"RUNTIME_SAMPLE_SIZE;optional;min=1;max=100000"`

	EmailSender              string `env:"EMAIL_SENDER"`
	EmailSecret              string `env:"EMAIL_SECRET"`
	EmailFromAddress         string `env:"EMAIL_FROM_ADDRESS"`
	EmailSMTPURL             string `env:"EMAIL_SMTP_URL"`
	WhatsAppProvider         string `env:"WHATSAPP_PROVIDER;optional;trimspace"`
	TwilioAccountSID         string `env:"TWILIO_ACCOUNT_SID;optional;trimspace"`
	TwilioAuthToken          string `env:"TWILIO_AUTH_TOKEN;optional;trimspace"`
	TwilioSMSFrom            string `env:"TWILIO_SMS_FROM;optional;trimspace"`
	TwilioWhatsAppFrom       string `env:"TWILIO_WHATSAPP_FROM;optional;trimspace"`
	TwilioWhatsAppContentSID string `env:"TWILIO_WHATSAPP_CONTENT_SID;optional;trimspace"`
	TwilioAPIBaseURL         string `env:"TWILIO_API_BASE_URL;optional;trimspace;format=URL"`
	AdminUser                string `env:"ADMIN_USER;optional;trimspace"`
	AdminPassword            string `env:"ADMIN_PASSWORD;optional"`

	GooseDriver       string `env:"GOOSE_DRIVER"`
	GooseDBString     string `env:"GOOSE_DBSTRING"`
	GooseMigrationDir string `env:"GOOSE_MIGRATION_DIR"`

	LogtoResource  string `env:"LOGTO_RESOURCE;optional;trimspace;format=URL;regex=^.{7,}$"`
	LogtoURL       string `env:"LOGTO_URL;optional;trimspace;format=URL;regex=^.{7,}$"`
	LogtoAppID     string `env:"LOGTO_APP_ID;optional;trimspace;format=IDENTIFIER;regex=^.{12,}$"`
	LogtoAppSecret string `env:"LOGTO_APP_SECRET;optional;trimspace;format=IDENTIFIER;regex=^.{12,}$"`

	OTelOTLPEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT;optional;trimspace"`
	OTelOTLPInsecure bool   `env:"OTEL_EXPORTER_OTLP_INSECURE;optional"`
	OTelServiceName  string `env:"OTEL_SERVICE_NAME;optional;trimspace"`
	OTelTracerServer string `env:"OTEL_TRACER_SERVER;optional;trimspace"`
	OTelTracerAuth   string `env:"OTEL_TRACER_AUTH;optional;trimspace"`

	JobsEnabled    bool   `env:"JOBS_ENABLED;optional"`
	JobsUIEnabled  bool   `env:"JOBS_UI_ENABLED;optional"`
	ValkeyHost     string `env:"VALKEY_HOST;optional;trimspace"`
	ValkeyPort     int    `env:"VALKEY_PORT;optional;min=1;max=65535"`
	ValkeyPassword string `env:"VALKEY_PASSWORD;optional;allowempty"`
	ValkeyDB       int    `env:"VALKEY_DB;optional;min=0;max=15"`

	AssetsDir string `env:"ASSETS_DIR"`

	StorageBackend string `env:"STORAGE_BACKEND;optional;trimspace;oneof=disk,s3"`
	S3Endpoint     string `env:"S3_ENDPOINT;optional;allowempty;trimspace"`
	S3UseSSL       bool   `env:"S3_USE_SSL;optional"`
	S3Region       string `env:"S3_REGION;optional;allowempty;trimspace"`
	S3AccessKey    string `env:"S3_ACCESS_KEY;optional;allowempty;trimspace"`
	S3SecretKey    string `env:"S3_SECRET_KEY;optional;allowempty"`
	S3Bucket       string `env:"S3_BUCKET;optional;allowempty;trimspace;regex=^([a-z0-9][a-z0-9-]{1,61}[a-z0-9])?$"`
}

const (
	StorageBackendDisk = "disk"
	StorageBackendS3   = "s3"
)

// ErrS3ConfigRequired is returned by New when STORAGE_BACKEND is s3 and an S3 setting is missing.
var ErrS3ConfigRequired = errors.New("S3 settings are required when STORAGE_BACKEND is s3")

// ErrOTLPEndpointRequired is returned by New when production has no OTLP endpoint.
var ErrOTLPEndpointRequired = errors.New("OTEL_EXPORTER_OTLP_ENDPOINT is required in production")

func New() (*Env, error) {
	env := &Env{
		AppShutdownTimeout: 10 * time.Second,
		RateLimiterEnabled: true,
		DBMaxOpenConns:     10,
		DBMaxIdleConns:     5,
		DBConnMaxLifetime:  30 * time.Minute,
		DBConnMaxIdleTime:  5 * time.Minute,
		OTelServiceName:    "miconsul",
		OTelTracerServer:   "miconsul.server",
		OTelTracerAuth:     "miconsul.auth",
		JobsEnabled:        false,
		JobsUIEnabled:      false,
		ValkeyHost:         "127.0.0.1",
		ValkeyPort:         6379,
		ValkeyDB:           0,
		StorageBackend:     StorageBackendDisk,

		RuntimeSampleInterval: 5 * time.Second,
		RuntimeSampleSize:     720,
	}
	err := simpleenv.Load(env)
	if err != nil {
		return nil, fmt.Errorf("load ENV variables: %w", err)
	}

	env.AssetsDir, err = normalizeAbsPath(env.AssetsDir)
	if err != nil {
		return nil, fmt.Errorf("normalize ASSETS_DIR: %w", err)
	}

	if env.IsProduction() && env.OTelOTLPEndpoint == "" {
		return nil, ErrOTLPEndpointRequired
	}

	if missing := env.missingS3Settings(); len(missing) > 0 {
		return nil, fmt.Errorf("%w: missing %s", ErrS3ConfigRequired, strings.Join(missing, ", "))
	}

	return env, nil
}

func normalizeAbsPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("path is required")
	}

	absPath, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}

	return absPath, nil
}

func (e *Env) IsDevelopment() bool {
	if e == nil {
		return false
	}

	return e.Environment == EnvironmentDevelopment
}

func (e *Env) IsTest() bool {
	if e == nil {
		return false
	}

	return e.Environment == EnvironmentTest
}

func (e *Env) IsDevOrTest() bool {
	if e == nil {
		return false
	}

	return e.Environment == EnvironmentDevelopment || e.Environment == EnvironmentTest
}

func (e *Env) IsProduction() bool {
	if e == nil {
		return false
	}

	return e.Environment == EnvironmentProduction
}

func (e *Env) IsValidEnvironment() bool {
	if e == nil {
		return false
	}

	return e.Environment.IsValid()
}
