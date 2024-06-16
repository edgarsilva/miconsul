package server

import "github.com/gofiber/fiber/v2/middleware/helmet"

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
		CrossOriginEmbedderPolicy: "false",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "*.dicebear.com",
	}
}
