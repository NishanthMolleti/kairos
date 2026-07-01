package handlers

import (
	"net/http"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/NishanthMolleti/kairos/oura"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SyncHandler struct {
	db       *sqlx.DB
	hfAPIKey string
}

func NewSyncHandler(db *sqlx.DB, hfAPIKey string) *SyncHandler {
	return &SyncHandler{db: db, hfAPIKey: hfAPIKey}
}

// POST /api/sync
func (h *SyncHandler) Sync(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	user, err := models.GetUserByID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := oura.SyncUser(c.Request.Context(), h.db, userID, user.AccessToken, h.hfAPIKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "sync complete"})
}
