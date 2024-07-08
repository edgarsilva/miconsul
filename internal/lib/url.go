package lib

import "os"

func AppURL() string {
	p := os.Getenv("APP_PROTOCOL")
	d := os.Getenv("APP_DOMAIN")

	return p + "://" + d
}
