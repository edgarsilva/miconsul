// Package appenv provides Runtime configuration for the App
package appenv

import (
	"log"
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

	CookieSecret string `env:"COOKIE_SECRET;regex=^.{32,}$"`
	JWTSecret    string `env:"JWT_SECRET"`

	DBPath        string `env:"DB_PATH"`
	SessionDBPath string `env:"SESSION_DB_PATH"`
	CacheDBPath   string `env:"CACHE_DB_PATH;optional"`

	EmailSender      string `env:"EMAIL_SENDER"`
	EmailSecret      string `env:"EMAIL_SECRET"`
	EmailFromAddress string `env:"EMAIL_FROM_ADDRESS"`
	EmailSMTPURL     string `env:"EMAIL_SMTP_URL"`

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

	AssetsDir string `env:"ASSETS_DIR"`
}

func New() *Env {
	env := &Env{
		AppShutdownTimeout: 10 * time.Second,
		OTelServiceName:    "miconsul",
		OTelTracerServer:   "miconsul.server",
		OTelTracerAuth:     "miconsul.auth",
	}
	err := simpleenv.Load(env)
	if err != nil {
		log.Fatal("failed to load loading ENV variables:", err)
	}

	return env
}
