package auth

type RawData struct {
	Sub           string `json:"sub"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Picture       string `json:"picture"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	EmailVerified bool   `json:"email_verified"`
}

type Details struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Email   string  `json:"email"`
	Avatar  string  `json:"avatar"`
	RawData RawData `json:"rawData"`
}

type GoogleIdentity struct {
	UserID  string  `json:"userId"`
	Details Details `json:"details"`
}

type Identities struct {
	Google GoogleIdentity `json:"google"`
}

type LogtoUser struct {
	UID           string     `json:"uid"`
	Name          string     `json:"name"`
	Username      string     `json:"username"`
	Picture       string     `json:"picture"`
	Email         string     `json:"email"`
	PhoneNumber   string     `json:"phoneNumber"`
	Roles         []string   `json:"roles"`
	Organizations []string   `json:"organizations"`
	Identities    Identities `json:"identities"`
	JTI           string     `json:"jti"`
	Sub           string     `json:"sub"`
	IAT           int64      `json:"iat"`
	EXP           int64      `json:"exp"`
	Scope         string     `json:"scope"`
	ClientID      string     `json:"client_id"`
	ISS           string     `json:"iss"`
	AUD           string     `json:"aud"`
}
