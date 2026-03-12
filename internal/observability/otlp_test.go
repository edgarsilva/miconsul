package observability

import "testing"

func TestIsInternalOTLPEndpoint(t *testing.T) {
	cases := []struct {
		endpoint string
		want     bool
	}{
		{endpoint: "localhost:4317", want: true},
		{endpoint: "127.0.0.1:4317", want: true},
		{endpoint: "lgtm:4317", want: true},
		{endpoint: "tempo:4317", want: true},
		{endpoint: "collector.example.com:4317", want: false},
	}

	for _, tc := range cases {
		if got := IsInternalOTLPEndpoint(tc.endpoint); got != tc.want {
			t.Fatalf("endpoint %q expected %v, got %v", tc.endpoint, tc.want, got)
		}
	}
}
