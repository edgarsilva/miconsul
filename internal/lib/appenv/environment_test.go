package appenv

import "testing"

func TestEnvironmentUnmarshalText(t *testing.T) {
	t.Run("accepts valid values", func(t *testing.T) {
		cases := []Environment{EnvironmentDevelopment, EnvironmentTest, EnvironmentStaging, EnvironmentProduction}
		for _, tc := range cases {
			v := Environment("")
			if err := (&v).UnmarshalText([]byte(tc)); err != nil {
				t.Fatalf("expected %q to unmarshal: %v", tc, err)
			}
			if v != tc {
				t.Fatalf("expected unmarshaled value %q, got %q", tc, v)
			}
		}
	})

	t.Run("rejects invalid value", func(t *testing.T) {
		v := Environment("")
		if err := (&v).UnmarshalText([]byte("qa")); err == nil {
			t.Fatalf("expected invalid environment error")
		}
	})
}
