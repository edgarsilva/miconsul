package auth

import "testing"

func TestSignupInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		in      signupInput
		wantErr string
	}{
		{"valid", signupInput{Email: "ana@example.com", Password: "Password1!", Confirm: "Password1!"}, ""},
		{"blank email", signupInput{Password: "Password1!", Confirm: "Password1!"}, "email and password can't be blank"},
		{"blank password", signupInput{Email: "ana@example.com"}, "email and password can't be blank"},
		{"blank confirmation", signupInput{Email: "ana@example.com", Password: "Password1!"}, "passwords don't match"},
		{"different confirmation", signupInput{Email: "ana@example.com", Password: "Password1!", Confirm: "Different1!"}, "passwords don't match"},
		{"password of spaces", signupInput{Email: "ana@example.com", Password: "   ", Confirm: "   "}, "email and password can't be blank"},
		{"malformed email", signupInput{Email: "bad-email", Password: "Password1!", Confirm: "Password1!"}, "email address is invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.trim().validate()
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != tt.wantErr {
				t.Fatalf("validate() = %q, want %q", got, tt.wantErr)
			}
		})
	}
}

func TestSignupInputTrim(t *testing.T) {
	in := signupInput{Email: "  ana@example.com\t", Password: " keep spaces ", Confirm: " keep spaces "}.trim()
	if in.Email != "ana@example.com" {
		t.Fatalf("email = %q, want it trimmed", in.Email)
	}
	if in.Password != " keep spaces " || in.Confirm != " keep spaces " {
		t.Fatalf("password and confirmation must stay as typed: %q %q", in.Password, in.Confirm)
	}
}
