package controller

// Tokens represents the structure of the OAuth2 token response
type Tokens struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int
	RefreshExpiresIn int
}

// Constants
var (
	STATUS_FAILED    = "FAILED"
	STATUS_REFRESHED = "REFRESHED"
)
