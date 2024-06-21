package server

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/session"
	logto "github.com/logto-io/go/client"
)

const csp = "default-src 'self';base-uri 'self';font-src 'self' https: data:;" +
	"form-action 'self';frame-ancestors 'self';" +
	"img-src 'self' data: *.dicebear.com *.pravatar.cc images.unsplash.com;object-src 'none';" +
	"script-src 'self' 'unsafe-eval' unpkg.com *.unpkg.com *.jsdelivr.net;" +
	"script-src-attr 'none';style-src 'self' https: 'unsafe-inline';upgrade-insecure-requests"

func helmetConfig() helmet.Config {
	return helmet.Config{
		ContentSecurityPolicy:     csp,
		XSSProtection:             "0",
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "SAMEORIGIN",
		ReferrerPolicy:            "no-referrer",
		CrossOriginEmbedderPolicy: "credentialless",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "cross-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "*.dicebear.com",
	}
}

// LocaleLang defines a universal middleware to stract Locale lang en-US, es-MX, etc
func LocaleLang(st *session.Store) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		lang := ""

		switch c.AcceptsLanguages("en-US", "es-MX", "es-US", "en", "es") {
		case "es-US", "es-MX", "es":
			lang = "es-MX"
		case "en-US", "en":
			lang = "en-US"
		default:
			lang = "es-MX"
		}

		c.Locals("locale", lang)
		sess, err := st.Get(c)
		if err == nil {
			sess.Set("locale", lang)
		}

		return c.Next()
	}
}

func LogtoConfig() *logto.LogtoConfig {
	endpoint := os.Getenv("LOGTO_URL")
	appid := os.Getenv("LOGTO_APP_ID")
	appsecret := os.Getenv("LOGTO_APP_SECRET")
	logtoConfig := logto.LogtoConfig{
		Endpoint:  endpoint,
		AppId:     appid,
		AppSecret: appsecret,
	}

	return &logtoConfig
}
