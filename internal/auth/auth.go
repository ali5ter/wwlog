// Package auth manages WW authentication and credential storage.
package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "wwlog"
	keyEmail    = "email"
	keyPassword = "password"
	keyToken    = "token"
)

// Auth handles WW authentication for a given TLD.
type Auth struct {
	TLD string
}

// Token returns a valid JWT, silently re-authenticating with stored
// credentials if the cached token is missing or within 5 minutes of expiry.
func (a *Auth) Token() (string, error) {
	token, err := keyring.Get(serviceName, keyToken)
	if err == nil && a.isValid(token) {
		return token, nil
	}

	email, err := keyring.Get(serviceName, keyEmail)
	if err != nil {
		return "", fmt.Errorf("no stored credentials — run 'wwlog --login' to authenticate")
	}
	password, err := keyring.Get(serviceName, keyPassword)
	if err != nil {
		return "", fmt.Errorf("no stored credentials — run 'wwlog --login' to authenticate")
	}

	return a.Login(email, password)
}

// Login authenticates with email and password, stores credentials and token
// in the system keychain, and returns the JWT.
func (a *Auth) Login(email, password string) (string, error) {
	tokenID, err := a.step1(email, password)
	if err != nil {
		return "", err
	}
	token, err := a.step2(tokenID)
	if err != nil {
		return "", err
	}

	_ = keyring.Set(serviceName, keyEmail, email)
	_ = keyring.Set(serviceName, keyPassword, password)
	_ = keyring.Set(serviceName, keyToken, token)

	return token, nil
}

// Expiry returns the expiry time embedded in the given JWT.
func (a *Auth) Expiry(token string) (time.Time, error) {
	return expiry(token)
}

// Logout removes all stored credentials from the system keychain.
func (a *Auth) Logout() error {
	_ = keyring.Delete(serviceName, keyEmail)
	_ = keyring.Delete(serviceName, keyPassword)
	_ = keyring.Delete(serviceName, keyToken)
	return nil
}

// isValid returns true if the token is well-formed and not expiring soon.
func (a *Auth) isValid(token string) bool {
	if len(token) < 900 {
		return false
	}
	exp, err := expiry(token)
	if err != nil {
		return false
	}
	return time.Now().Add(5 * time.Minute).Before(exp)
}

// step1 posts credentials to the WW authenticate endpoint and returns a tokenId.
func (a *Auth) step1(email, password string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"username":        email,
		"password":        password,
		"rememberMe":      false,
		"usernameEncoded": false,
		"retry":           false,
	})

	resp, err := http.Post(
		fmt.Sprintf("https://auth.weightwatchers.%s/login-apis/v1/authenticate", a.TLD),
		"application/json",
		strings.NewReader(string(body)),
	)
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed (%d): check your email and password", resp.StatusCode)
	}

	var result struct {
		Data struct {
			TokenID string `json:"tokenId"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("unexpected login response: %w", err)
	}
	return result.Data.TokenID, nil
}

// step2 exchanges the tokenId for a JWT via the OAuth2 authorize endpoint.
func (a *Auth) step2(tokenID string) (string, error) {
	nonce := make([]byte, 16)
	_, _ = rand.Read(nonce)

	redirectURI := fmt.Sprintf("https://cmx.weightwatchers.%s/auth", a.TLD)
	authURL := fmt.Sprintf(
		"https://auth.weightwatchers.%s/openam/oauth2/authorize?response_type=id_token&client_id=webCMX&redirect_uri=%s&nonce=%s",
		a.TLD,
		url.QueryEscape(redirectURI),
		base64.RawURLEncoding.EncodeToString(nonce),
	)

	req, _ := http.NewRequest(http.MethodGet, authURL, nil)
	req.AddCookie(&http.Cookie{Name: "wwAuth2", Value: tokenID})

	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("authorize request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		return "", fmt.Errorf("unexpected authorize response (%d) — API may have changed", resp.StatusCode)
	}

	u, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return "", fmt.Errorf("invalid redirect location: %w", err)
	}

	fragment, _ := url.ParseQuery(u.Fragment)
	token := fragment.Get("id_token")
	if token == "" {
		return "", fmt.Errorf("no id_token in redirect — API may have changed")
	}
	return token, nil
}

// expiry decodes the JWT payload and returns the exp claim as a time.Time.
func expiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("invalid JWT format")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid JWT payload: %w", err)
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("invalid JWT claims: %w", err)
	}
	return time.Unix(claims.Exp, 0), nil
}
