package handlers

import (
	"net/http"
	"time"

	"github.com/NishanthMolleti/kairos/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MetricsHandler struct{ db *sqlx.DB }

func NewMetricsHandler(db *sqlx.DB) *MetricsHandler { return &MetricsHandler{db: db} }

func (h *MetricsHandler) dateRange(c *gin.Context) (string, string) {
	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	if f := c.Query("from"); f != "" {
		from = f
	}
	if t := c.Query("to"); t != "" {
		to = t
	}
	return from, to
}

func (h *MetricsHandler) Sleep(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetSleepRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) Readiness(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetReadinessRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) Activity(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetActivityRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) HRV(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetHRVRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) HeartRate(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetHeartRateRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) SpO2(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetSpO2Range(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) Stress(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetStressRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *MetricsHandler) Workouts(c *gin.Context) {
	userID := c.MustGet("userID").(uuid.UUID)
	from, to := h.dateRange(c)
	data, err := models.GetWorkoutsRange(h.db, userID, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}
