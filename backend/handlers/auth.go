package handlers

import (
	"net/http"
	"sync"
	"time"

	kauth "github.com/NishanthMolleti/kairos/auth"
	"github.com/NishanthMolleti/kairos/config"
	"github.com/NishanthMolleti/kairos/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type pkceEntry struct {
	verifier  string
	expiresAt time.Time
}

type AuthHandler struct {
	cfg       *config.Config
	db        *sqlx.DB
	oauth     *kauth.OAuthConfig
	statesMu  sync.Mutex
	states    map[string]pkceEntry
}

func NewAuthHandler(cfg *config.Config, db *sqlx.DB) *AuthHandler {
	h := &AuthHandler{
		cfg: cfg,
		db:  db,
		oauth: &kauth.OAuthConfig{
			ClientID:     cfg.OuraClientID,
			ClientSecret: cfg.OuraClientSecret,
			RedirectURL:  cfg.OuraRedirectURL,
		},
		states: make(map[string]pkceEntry),
	}
	go h.cleanupStates()
	return h
}

func (h *AuthHandler) cleanupStates() {
	for range time.Tick(5 * time.Minute) {
		h.statesMu.Lock()
		now := time.Now()
		for k, v := range h.states {
			if now.After(v.expiresAt) {
				delete(h.states, k)
			}
		}
		h.statesMu.Unlock()
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
	h.statesMu.Lock()
	h.states[state] = pkceEntry{verifier: verifier, expiresAt: time.Now().Add(10 * time.Minute)}
	h.statesMu.Unlock()
	c.Redirect(http.StatusFound, h.oauth.AuthURL(state, challenge))
}

// GET /auth/callback
func (h *AuthHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	h.statesMu.Lock()
	entry, ok := h.states[state]
	if ok {
		delete(h.states, state)
	}
	h.statesMu.Unlock()

	if !ok || time.Now().After(entry.expiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}

	tokens, err := h.oauth.ExchangeCode(c.Request.Context(), code, entry.verifier)
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
