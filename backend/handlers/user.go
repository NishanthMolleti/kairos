package handlers

import (
	"net/http"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type UserHandler struct{ db *sqlx.DB }

func NewUserHandler(db *sqlx.DB) *UserHandler { return &UserHandler{db: db} }

// GET /api/user
func (h *UserHandler) GetUser(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	user, err := models.GetUserByID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": user.ID, "email": user.Email, "last_sync": user.LastSync})
}
