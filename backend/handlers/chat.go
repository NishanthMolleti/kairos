package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/NishanthMolleti/kairos/ai"
	"github.com/NishanthMolleti/kairos/config"
	"github.com/NishanthMolleti/kairos/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type ChatHandler struct {
	db  *sqlx.DB
	cfg *config.Config
}

func NewChatHandler(db *sqlx.DB, cfg *config.Config) *ChatHandler {
	return &ChatHandler{db: db, cfg: cfg}
}

// POST /api/chat/sessions
func (h *ChatHandler) CreateSession(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	var body struct {
		Title string `json:"title"`
	}
	_ = c.ShouldBindJSON(&body)
	if body.Title == "" {
		body.Title = "New conversation"
	}
	session, err := models.CreateSession(h.db, userID, body.Title)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, session)
}

// GET /api/chat/sessions/:id/messages
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	ownerID, err := models.GetSessionOwner(h.db, sessionID)
	if err != nil || ownerID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}
	msgs, err := models.GetSessionMessages(h.db, sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, msgs)
}

// POST /api/chat/sessions/:id/ask — streams Sage response as SSE
func (h *ChatHandler) AskSage(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	var body struct {
		Question string `json:"question" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "question is required"})
		return
	}
	ownerID, err := models.GetSessionOwner(h.db, sessionID)
	if err != nil || ownerID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "session not found"})
		return
	}

	if _, err := models.AddMessage(h.db, sessionID, "user", body.Question); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	tokenCh := make(chan string, 64)
	ctx := c.Request.Context()
	go ai.Ask(ctx, h.db, h.cfg.GroqAPIKey, h.cfg.HuggingFaceAPIKey, userID, body.Question, tokenCh)

	var fullResponse strings.Builder
	w := c.Writer
	flusher, hasFlusher := w.(http.Flusher)

	for token := range tokenCh {
		if strings.HasPrefix(token, "ERROR:") {
			fmt.Fprintf(w, "event: error\ndata: %s\n\n", token)
			if hasFlusher {
				flusher.Flush()
			}
			break
		}
		fullResponse.WriteString(token)
		escaped := strings.ReplaceAll(token, "\n", "\ndata: ")
		fmt.Fprintf(w, "data: %s\n\n", escaped)
		if hasFlusher {
			flusher.Flush()
		}
	}

	fmt.Fprintf(w, "event: done\ndata: [DONE]\n\n")
	if hasFlusher {
		flusher.Flush()
	}

	if fullResponse.Len() > 0 {
		_, _ = models.AddMessage(h.db, sessionID, "assistant", fullResponse.String())
	}
}
