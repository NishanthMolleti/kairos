package handlers

import (
	"net/http"

	kauth "github.com/NishanthMolleti/kairos/auth"
	"github.com/NishanthMolleti/kairos/config"
	"github.com/NishanthMolleti/kairos/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type AuthHandler struct {
	cfg   *config.Config
	db    *sqlx.DB
	oauth *kauth.OAuthConfig
}

func NewAuthHandler(cfg *config.Config, db *sqlx.DB) *AuthHandler {
	return &AuthHandler{
		cfg: cfg,
		db:  db,
		oauth: &kauth.OAuthConfig{
			ClientID:     cfg.OuraClientID,
			ClientSecret: cfg.OuraClientSecret,
			RedirectURL:  cfg.OuraRedirectURL,
		},
	}
}

// GET /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	verifier, challenge, err := kauth.GeneratePKCE()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "pkce failed"})
		return
	}
	state, err := kauth.GenerateState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "state failed"})
		return
	}
	c.SetCookie("pkce_verifier", verifier, 600, "/", "", true, true)
	c.SetCookie("oauth_state", state, 600, "/", "", true, true)
	c.Redirect(http.StatusFound, h.oauth.AuthURL(state, challenge))
}

// GET /auth/callback
func (h *AuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	cookieState, err := c.Cookie("oauth_state")
	if err != nil || cookieState != state {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}
	verifier, err := c.Cookie("pkce_verifier")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing verifier"})
		return
	}

	tokens, err := h.oauth.ExchangeCode(c.Request.Context(), code, verifier)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token exchange failed"})
		return
	}

	info, err := kauth.FetchPersonalInfo(c.Request.Context(), tokens.AccessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "fetch user failed"})
		return
	}

	if err := models.UpsertUser(h.db, &models.User{
		OuraUserID:   info.ID,
		Email:        info.Email,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	dbUser, err := models.GetUserByOuraID(h.db, info.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}

	jwt, err := kauth.SignJWT(dbUser.ID, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "jwt error"})
		return
	}

	c.Redirect(http.StatusFound, h.cfg.FrontendURL+"/auth/callback?token="+jwt)
}

// POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}
