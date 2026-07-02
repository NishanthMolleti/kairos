// auth/oauth.go
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	ouraAuthURL  = "https://cloud.ouraring.com/oauth/authorize"
	ouraTokenURL = "https://api.ouraring.com/oauth/token"
	ouraUserURL  = "https://api.ouraring.com/v2/usercollection/personal_info"
)

type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type OuraPersonalInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

func GeneratePKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	return
}

func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (c *OAuthConfig) AuthURL(state, challenge string) string {
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {c.ClientID},
		"redirect_uri":          {c.RedirectURL},
		"scope":                 {"email personal daily workout heartrate spo2"},
		"state":                 {state},
		"code_challenge":        {challenge},
		"code_challenge_method": {"S256"},
	}
	return ouraAuthURL + "?" + params.Encode()
}

func (c *OAuthConfig) ExchangeCode(ctx context.Context, code, verifier string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.RedirectURL},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
		"code_verifier": {verifier},
	}
	return c.postToken(ctx, data)
}

func (c *OAuthConfig) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.ClientID},
		"client_secret": {c.ClientSecret},
	}
	return c.postToken(ctx, data)
}

func (c *OAuthConfig) postToken(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ouraTokenURL,
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed %d: %s", resp.StatusCode, body)
	}

	var tr TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, err
	}
	return &tr, nil
}

func FetchPersonalInfo(ctx context.Context, accessToken string) (*OuraPersonalInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ouraUserURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("personal info failed %d: %s", resp.StatusCode, body)
	}

	var info OuraPersonalInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}
