package web

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"volcengine-updater/internal/config"
	"volcengine-updater/internal/models"
	"volcengine-updater/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

type Handler struct {
	Cron  *cron.Cron
	JobID *cron.EntryID
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/", h.Index)
	r.GET("/settings", h.SettingsPage)
	r.POST("/settings", h.SaveSettings)
	r.POST("/run_now", h.RunNow)
	r.GET("/logs", h.LogsPage)
}

func (h *Handler) Index(c *gin.Context) {
	settings := config.GetSettings()
	var logs []models.UpdateLog
	config.DB.Order("timestamp desc").Limit(10).Find(&logs)

	var nextRun string
	if h.JobID != nil {
		entry := h.Cron.Entry(*h.JobID)
		if !entry.Next.IsZero() {
			nextRun = entry.Next.Format("2006-01-02 15:04:05")
		} else {
			nextRun = "Running..."
		}
	} else {
		nextRun = "Not Scheduled"
	}
	
	flash := c.Query("flash")

	c.HTML(http.StatusOK, "index.html", gin.H{
		"Settings": settings,
		"Logs":     logs,
		"NextRun":  nextRun,
		"Flash":    flash,
	})
}

func (h *Handler) SettingsPage(c *gin.Context) {
	settings := config.GetSettings()
	c.HTML(http.StatusOK, "settings.html", gin.H{
		"Settings": settings,
	})
}

func (h *Handler) SaveSettings(c *gin.Context) {
	var form struct {
		AccessKey       string `form:"access_key"`
		SecretKey       string `form:"secret_key"`
		Region          string `form:"region"`
		SecurityGroupID string `form:"security_group_id"`
		SSHPort         int    `form:"ssh_port"`
		CheckInterval   int    `form:"check_interval"`
		IPServices      string `form:"ip_services"`
	}

	if err := c.ShouldBind(&form); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings := config.GetSettings()
	settings.AccessKey = form.AccessKey
	settings.SecretKey = form.SecretKey
	settings.Region = form.Region
	settings.SecurityGroupID = form.SecurityGroupID
	settings.SSHPort = form.SSHPort
	settings.CheckInterval = form.CheckInterval
	settings.IPServices = form.IPServices

	config.DB.Save(settings)

	// Update Job
	if h.JobID != nil {
		h.Cron.Remove(*h.JobID)
	}

	id, _ := h.Cron.AddFunc(fmt.Sprintf("@every %ds", settings.CheckInterval), service.CheckAndUpdate)
	h.JobID = &id

	c.Redirect(http.StatusFound, "/?flash=Settings+Saved")
}

func (h *Handler) RunNow(c *gin.Context) {
	go service.CheckAndUpdate()
	c.Redirect(http.StatusFound, "/?flash=Manual+Update+Triggered")
}

func (h *Handler) LogsPage(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	pageSize := 20

	var total int64
	config.DB.Model(&models.UpdateLog{}).Count(&total)

	var logs []models.UpdateLog
	offset := (page - 1) * pageSize
	config.DB.Order("timestamp desc").Limit(pageSize).Offset(offset).Find(&logs)

	totalPages := int(math.Ceil(float64(total) / float64(pageSize)))

	// Helper for pagination
	hasPrev := page > 1
	hasNext := page < totalPages

	c.HTML(http.StatusOK, "logs.html", gin.H{
		"Logs":       logs,
		"Page":       page,
		"TotalPages": totalPages,
		"HasPrev":    hasPrev,
		"HasNext":    hasNext,
		"PrevPage":   page - 1,
		"NextPage":   page + 1,
	})
}
