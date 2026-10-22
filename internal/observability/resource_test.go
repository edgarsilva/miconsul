package observability

import (
	"testing"

	"miconsul/internal/lib/appenv"
)

func TestNewResource(t *testing.T) {
	env := &appenv.Env{AppName: "miconsul", OTelServiceName: "miconsul.prod", AppVersion: "1.2.3", Environment: appenv.EnvironmentProduction}

	res, err := NewResource(t.Context(), env)
	if err != nil {
		t.Fatalf("new resource: %v", err)
	}

	got := map[string]string{}
	for _, kv := range res.Attributes() {
		got[string(kv.Key)] = kv.Value.Emit()
	}
	want := map[string]string{
		"service.name":                "miconsul.prod",
		"service.version":             "1.2.3",
		"deployment.environment.name": string(appenv.EnvironmentProduction),
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if _, ok := got["deployment.environment"]; ok {
		t.Errorf("deprecated deployment.environment must not be set")
	}

	env.OTelServiceName = ""
	res, err = NewResource(t.Context(), env)
	if err != nil {
		t.Fatalf("new resource without service name: %v", err)
	}
	for _, kv := range res.Attributes() {
		if kv.Key == "service.name" && kv.Value.Emit() != "miconsul" {
			t.Errorf("service.name falls back to AppName, got %q", kv.Value.Emit())
		}
	}
}
