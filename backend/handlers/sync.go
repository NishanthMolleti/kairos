package handlers

import (
	"context"
	"net/http"
	"time"

	kauth "github.com/NishanthMolleti/kairos/auth"
	"github.com/NishanthMolleti/kairos/models"
	"github.com/NishanthMolleti/kairos/oura"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SyncHandler struct {
	db       *sqlx.DB
	hfAPIKey string
	oauth    *kauth.OAuthConfig
}

func NewSyncHandler(db *sqlx.DB, hfAPIKey string, oauth *kauth.OAuthConfig) *SyncHandler {
	return &SyncHandler{db: db, hfAPIKey: hfAPIKey, oauth: oauth}
}

// POST /api/sync
func (h *SyncHandler) Sync(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	user, err := models.GetUserByID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := oura.SyncUser(ctx, h.db, userID, user.AccessToken, user.RefreshToken, h.oauth, h.hfAPIKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sync complete"})
}
