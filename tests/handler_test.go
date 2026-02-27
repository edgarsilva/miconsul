package tests

import (
	"io"
	"miconsul/internal/server"
	"miconsul/internal/service/theme"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHandler(t *testing.T) {
	// Create a Fiber fiberApp for testing
	fiberApp := fiber.New()

	// Inject the Fiber app into the server
	s := &server.Server{App: fiberApp}
	rc := theme.NewService(s)

	// Define a route in the Fiber app
	fiberApp.Get("/", rc.HandleToggleTheme)

	// Create a test HTTP request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatalf("error creating request. Err: %v", err)
	}

	// Perform the request
	resp, err := fiberApp.Test(req)
	if err != nil {
		t.Fatalf("error making request to server. Err: %v", err)
	}

	// Your test assertions...
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("error reading response body. Err: %v", err)
	}

	bodyStr := string(body)
	if !strings.Contains(bodyStr, "theme_toggle") {
		t.Errorf("expected response body to include theme toggle markup; got %v", bodyStr)
	}
}
