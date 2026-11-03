package controllers

// Hammond+ endpoints: maintenance schedule, reminders, tires, documents,
// vehicle photo, VIN decode, recalls and notification settings.

import (
	"net/http"
	"strings"
	"time"

	"hammond/common"
	"hammond/db"
	"hammond/service"

	"github.com/gin-gonic/gin"
)

func RegisterPlusController(router *gin.RouterGroup) {
	router.GET("/me/reminders", getMyReminders)
	router.GET("/vin/:vin", decodeVin)

	router.GET("/vehicles/:id/services", listServices)
	router.POST("/vehicles/:id/services", createService)
	router.POST("/vehicles/:id/services/defaults", seedServices)
	router.PUT("/vehicles/:id/services/:subId", updateService)
	router.DELETE("/vehicles/:id/services/:subId", deleteService)
	router.POST("/vehicles/:id/services/:subId/complete", completeService)
	router.POST("/vehicles/:id/services/:subId/snooze", snoozeService)

	router.GET("/vehicles/:id/tires", getTires)
	router.POST("/vehicles/:id/tires", createTireSet)
	router.PUT("/vehicles/:id/tires/:subId", updateTireSet)
	router.DELETE("/vehicles/:id/tires/:subId", deleteTireSet)
	router.POST("/vehicles/:id/tires/:subId/mount", mountTireSet)
	router.POST("/vehicles/:id/rotations", logRotation)
	router.POST("/vehicles/:id/treadReadings", addTreadReading)

	router.GET("/vehicles/:id/documents", listDocuments)
	router.POST("/vehicles/:id/documents", createDocument)
	router.PUT("/vehicles/:id/documents/:subId", updateDocument)
	router.DELETE("/vehicles/:id/documents/:subId", deleteDocument)

	router.POST("/vehicles/:id/photo", uploadVehiclePhoto)
	router.DELETE("/vehicles/:id/photo", deleteVehiclePhoto)

	router.GET("/vehicles/:id/recalls", getRecalls)
	router.POST("/vehicles/:id/recalls/check", checkRecalls)
	router.POST("/vehicles/:id/recalls/:subId/dismiss", dismissRecall)
	router.DELETE("/vehicles/:id/recalls/:subId/dismiss", undismissRecall)

	router.GET("/notifications/settings", ShouldBeAdmin(), getNotificationSettings)
	router.POST("/notifications/settings", ShouldBeAdmin(), saveNotificationSettings)
	router.POST("/notifications/test", ShouldBeAdmin(), testNotification)
}

// vehicleFor loads the vehicle in :id and checks the current user can access it.
func vehicleFor(c *gin.Context) (*db.Vehicle, bool) {
	id := c.Param("id")
	userID := c.MustGet("userId").(string)
	var count int64
	db.DB.Model(&db.UserVehicle{}).Where("vehicle_id = ? AND user_id = ?", id, userID).Count(&count)
	if count == 0 {
		if u, ok := c.MustGet("userModel").(db.User); !ok || u.Role != db.ADMIN {
			c.JSON(http.StatusForbidden, gin.H{"error": "no access to this vehicle"})
			return nil, false
		}
	}
	v, err := db.GetVehicleById(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "vehicle not found"})
		return nil, false
	}
	return v, true
}

func fail(c *gin.Context, key string, err error) {
	c.JSON(http.StatusUnprocessableEntity, common.NewError(key, err))
}

func dateOrNow(t *time.Time) time.Time {
	if t == nil || t.IsZero() {
		return time.Now()
	}
	return *t
}

// ---------------------------------------------------------------- reminders + VIN

func getMyReminders(c *gin.Context) {
	list, err := service.RemindersForUser(c.MustGet("userId").(string))
	if err != nil {
		fail(c, "reminders", err)
		return
	}
	c.JSON(http.StatusOK, list)
}

func decodeVin(c *gin.Context) {
	info, err := service.DecodeVIN(c.Param("vin"))
	if err != nil {
		fail(c, "vin", err)
		return
	}
	c.JSON(http.StatusOK, info)
}

// ---------------------------------------------------------------- services

type serviceRequest struct {
	Name           string     `json:"name" binding:"required"`
	Kind           string     `json:"kind"`
	Trigger        string     `json:"trigger"`
	IntervalMiles  int        `json:"intervalMiles"`
	IntervalMonths int        `json:"intervalMonths"`
	LastOdo        *int       `json:"lastOdo"`
	LastDate       *time.Time `json:"lastDate"`
	EstCost        float32    `json:"estCost"`
	LeadDays       *int       `json:"leadDays"`
	LeadMiles      *int       `json:"leadMiles"`
}

func (r serviceRequest) apply(v *db.Vehicle, s *db.ServiceItem) {
	s.VehicleID = v.ID
	s.Name = strings.TrimSpace(r.Name)
	s.Kind = r.Kind
	if s.Kind == "" {
		s.Kind = "service"
	}
	s.Trigger = r.Trigger
	if s.Trigger != "miles" && s.Trigger != "date" {
		s.Trigger = "first"
	}
	s.IntervalMiles = r.IntervalMiles
	s.IntervalMonths = r.IntervalMonths
	if r.LastOdo != nil {
		s.LastOdo = *r.LastOdo
	} else if s.ID == "" {
		s.LastOdo = service.CurrentOdometer(v.ID)
	}
	if r.LastDate != nil {
		s.LastDate = r.LastDate
	} else if s.LastDate == nil {
		now := time.Now()
		s.LastDate = &now
	}
	s.EstCost = r.EstCost
	if r.LeadDays != nil {
		s.LeadDays = *r.LeadDays
	} else if s.ID == "" {
		s.LeadDays = 14
	}
	if r.LeadMiles != nil {
		s.LeadMiles = *r.LeadMiles
	} else if s.ID == "" {
		s.LeadMiles = 500
	}
	s.IsActive = true
}

func listServices(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var items []db.ServiceItem
	db.DB.Where("vehicle_id = ?", v.ID).Order("created_at asc").Find(&items)
	reminders := map[string]service.Reminder{}
	for _, r := range service.RemindersForVehicle(v) {
		reminders[r.RefID] = r
	}
	out := []gin.H{}
	for _, s := range items {
		out = append(out, gin.H{"item": s, "reminder": reminders[s.ID]})
	}
	c.JSON(http.StatusOK, gin.H{"items": out, "odometer": service.CurrentOdometer(v.ID), "milesPerDay": service.MilesPerDay(v.ID)})
}

func createService(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var req serviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	var s db.ServiceItem
	req.apply(v, &s)
	if err := db.DB.Create(&s).Error; err != nil {
		fail(c, "service", err)
		return
	}
	c.JSON(http.StatusCreated, s)
}

func seedServices(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	if err := service.SeedServiceDefaults(v); err != nil {
		fail(c, "service", err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{})
}

func serviceFor(c *gin.Context, v *db.Vehicle) (*db.ServiceItem, bool) {
	var s db.ServiceItem
	if err := db.DB.Where("id = ? AND vehicle_id = ?", c.Param("subId"), v.ID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return nil, false
	}
	return &s, true
}

func updateService(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	s, ok := serviceFor(c, v)
	if !ok {
		return
	}
	var req serviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	req.apply(v, s)
	if err := db.DB.Save(s).Error; err != nil {
		fail(c, "service", err)
		return
	}
	c.JSON(http.StatusOK, s)
}

func deleteService(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	db.DB.Where("id = ? AND vehicle_id = ?", c.Param("subId"), v.ID).Delete(&db.ServiceItem{})
	c.JSON(http.StatusOK, gin.H{})
}

func completeService(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	s, ok := serviceFor(c, v)
	if !ok {
		return
	}
	var req struct {
		Date       *time.Time `json:"date"`
		OdoReading int        `json:"odoReading"`
		Amount     float32    `json:"amount"`
		Comments   string     `json:"comments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	if req.OdoReading == 0 {
		req.OdoReading = service.CurrentOdometer(v.ID)
	}
	exp, err := service.CompleteService(s, dateOrNow(req.Date), req.OdoReading, req.Amount, req.Comments, c.MustGet("userId").(string))
	if err != nil {
		fail(c, "service", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": s, "expense": exp})
}

func snoozeService(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	s, ok := serviceFor(c, v)
	if !ok {
		return
	}
	var req struct {
		Days int `json:"days"`
	}
	c.ShouldBindJSON(&req)
	if err := service.SnoozeService(s, req.Days); err != nil {
		fail(c, "service", err)
		return
	}
	c.JSON(http.StatusOK, s)
}

// ---------------------------------------------------------------- tires

func getTires(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, service.GetTireOverview(v))
}

type tireSetRequest struct {
	Name          string     `json:"name" binding:"required"`
	Brand         string     `json:"brand"`
	Size          string     `json:"size"`
	StoredAt      string     `json:"storedAt"`
	InstalledDate *time.Time `json:"installedDate"`
	InstalledOdo  *int       `json:"installedOdo"`
	MilesBefore   int        `json:"milesBefore"`
	TreadNew      float32    `json:"treadNew"`
}

func (r tireSetRequest) apply(s *db.TireSet) {
	s.Name, s.Brand, s.Size, s.StoredAt, s.MilesBefore = r.Name, r.Brand, r.Size, r.StoredAt, r.MilesBefore
	if r.InstalledDate != nil {
		s.InstalledDate = r.InstalledDate
	}
	if r.InstalledOdo != nil {
		s.InstalledOdo = *r.InstalledOdo
	}
	if r.TreadNew > 0 {
		s.TreadNew = r.TreadNew
	} else if s.TreadNew == 0 {
		s.TreadNew = 10
	}
}

func createTireSet(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var req tireSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	s := db.TireSet{VehicleID: v.ID}
	req.apply(&s)
	if err := db.DB.Create(&s).Error; err != nil {
		fail(c, "tires", err)
		return
	}
	c.JSON(http.StatusCreated, s)
}

func tireSetFor(c *gin.Context, v *db.Vehicle) (*db.TireSet, bool) {
	var s db.TireSet
	if err := db.DB.Where("id = ? AND vehicle_id = ?", c.Param("subId"), v.ID).First(&s).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return nil, false
	}
	return &s, true
}

func updateTireSet(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	s, ok := tireSetFor(c, v)
	if !ok {
		return
	}
	var req tireSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	req.apply(s)
	if err := db.DB.Save(s).Error; err != nil {
		fail(c, "tires", err)
		return
	}
	c.JSON(http.StatusOK, s)
}

func deleteTireSet(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	s, ok := tireSetFor(c, v)
	if !ok {
		return
	}
	if s.IsMounted {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "mount another set before deleting this one"})
		return
	}
	db.DB.Where("tire_set_id = ?", s.ID).Delete(&db.TreadReading{})
	db.DB.Delete(s)
	c.JSON(http.StatusOK, gin.H{})
}

type odoDateRequest struct {
	Date       *time.Time `json:"date"`
	OdoReading int        `json:"odoReading"`
}

func mountTireSet(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var req odoDateRequest
	c.ShouldBindJSON(&req)
	if req.OdoReading == 0 {
		req.OdoReading = service.CurrentOdometer(v.ID)
	}
	if err := service.MountTireSet(v, c.Param("subId"), dateOrNow(req.Date), req.OdoReading); err != nil {
		fail(c, "tires", err)
		return
	}
	c.JSON(http.StatusOK, service.GetTireOverview(v))
}

func logRotation(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var req odoDateRequest
	c.ShouldBindJSON(&req)
	if req.OdoReading == 0 {
		req.OdoReading = service.CurrentOdometer(v.ID)
	}
	if _, err := service.LogRotation(v, dateOrNow(req.Date), req.OdoReading); err != nil {
		fail(c, "tires", err)
		return
	}
	c.JSON(http.StatusOK, service.GetTireOverview(v))
}

func addTreadReading(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var r db.TreadReading
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	r.ID = ""
	if r.Date.IsZero() {
		r.Date = time.Now()
	}
	if r.OdoReading == 0 {
		r.OdoReading = service.CurrentOdometer(v.ID)
	}
	if err := service.AddTreadReading(v, &r); err != nil {
		fail(c, "tires", err)
		return
	}
	c.JSON(http.StatusOK, service.GetTireOverview(v))
}

// ---------------------------------------------------------------- documents

type documentRequest struct {
	Type         string     `json:"type" binding:"required"`
	Title        string     `json:"title"`
	Provider     string     `json:"provider"`
	Number       string     `json:"number"`
	ExpiryDate   *time.Time `json:"expiryDate"`
	AttachmentID string     `json:"attachmentId"`
	Notes        string     `json:"notes"`
}

func (r documentRequest) apply(d *db.Document) {
	d.Type, d.Title, d.Provider, d.Number, d.ExpiryDate, d.Notes = r.Type, r.Title, r.Provider, r.Number, r.ExpiryDate, r.Notes
	if r.AttachmentID != "" {
		d.AttachmentID = r.AttachmentID
	}
}

func listDocuments(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var docs []db.Document
	db.DB.Where("vehicle_id = ?", v.ID).Order("created_at asc").Find(&docs)
	c.JSON(http.StatusOK, docs)
}

func createDocument(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var req documentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	d := db.Document{VehicleID: v.ID}
	req.apply(&d)
	if err := db.DB.Create(&d).Error; err != nil {
		fail(c, "document", err)
		return
	}
	c.JSON(http.StatusCreated, d)
}

func updateDocument(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	var d db.Document
	if err := db.DB.Where("id = ? AND vehicle_id = ?", c.Param("subId"), v.ID).First(&d).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	var req documentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	req.apply(&d)
	if err := db.DB.Save(&d).Error; err != nil {
		fail(c, "document", err)
		return
	}
	c.JSON(http.StatusOK, d)
}

func deleteDocument(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	db.DB.Where("id = ? AND vehicle_id = ?", c.Param("subId"), v.ID).Delete(&db.Document{})
	c.JSON(http.StatusOK, gin.H{})
}

// ---------------------------------------------------------------- photo

func uploadVehiclePhoto(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	att, err := saveUploadedFile(c, "file")
	if err != nil {
		fail(c, "photo", err)
		return
	}
	if err := db.DB.Model(v).Update("photo_attachment_id", att.ID).Error; err != nil {
		fail(c, "photo", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"photoAttachmentId": att.ID})
}

func deleteVehiclePhoto(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	db.DB.Model(v).Update("photo_attachment_id", "")
	c.JSON(http.StatusOK, gin.H{})
}

// ---------------------------------------------------------------- recalls

func getRecalls(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"recalls": service.GetRecalls(v), "checkedAt": v.RecallCheckedAt, "openCount": v.RecallCount})
}

func checkRecalls(c *gin.Context) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	items, err := service.CheckRecalls(v)
	if err != nil {
		fail(c, "recalls", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"recalls": items, "checkedAt": v.RecallCheckedAt, "openCount": v.RecallCount})
}

func setRecall(c *gin.Context, dismissed bool) {
	v, ok := vehicleFor(c)
	if !ok {
		return
	}
	if err := service.SetRecallDismissed(v, c.Param("subId"), dismissed); err != nil {
		fail(c, "recalls", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"recalls": service.GetRecalls(v), "checkedAt": v.RecallCheckedAt, "openCount": v.RecallCount})
}

func dismissRecall(c *gin.Context)   { setRecall(c, true) }
func undismissRecall(c *gin.Context) { setRecall(c, false) }

// ---------------------------------------------------------------- notifications

func notificationView(s *db.NotificationSettings) gin.H {
	return gin.H{"ntfyEnabled": s.NtfyEnabled, "ntfyUrl": s.NtfyURL, "ntfyTopic": s.NtfyTopic, "hasNtfyToken": s.NtfyToken != "",
		"hookEnabled": s.HookEnabled, "webhookUrl": s.WebhookURL,
		"emailEnabled": s.EmailEnabled, "smtpHost": s.SMTPHost, "smtpPort": s.SMTPPort, "smtpUser": s.SMTPUser, "hasSmtpPassword": s.SMTPPassword != "",
		"smtpFrom": s.SMTPFrom, "emailTo": s.EmailTo}
}

func getNotificationSettings(c *gin.Context) {
	c.JSON(http.StatusOK, notificationView(service.GetNotificationSettings()))
}

func saveNotificationSettings(c *gin.Context) {
	var req struct {
		NtfyEnabled  bool    `json:"ntfyEnabled"`
		NtfyURL      string  `json:"ntfyUrl"`
		NtfyTopic    string  `json:"ntfyTopic"`
		NtfyToken    *string `json:"ntfyToken"`
		HookEnabled  bool    `json:"hookEnabled"`
		WebhookURL   string  `json:"webhookUrl"`
		EmailEnabled bool    `json:"emailEnabled"`
		SMTPHost     string  `json:"smtpHost"`
		SMTPPort     int     `json:"smtpPort"`
		SMTPUser     string  `json:"smtpUser"`
		SMTPPassword *string `json:"smtpPassword"`
		SMTPFrom     string  `json:"smtpFrom"`
		EmailTo      string  `json:"emailTo"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, common.NewValidatorError(err))
		return
	}
	s := service.GetNotificationSettings()
	s.NtfyEnabled, s.NtfyURL, s.NtfyTopic = req.NtfyEnabled, req.NtfyURL, req.NtfyTopic
	s.HookEnabled, s.WebhookURL = req.HookEnabled, req.WebhookURL
	s.EmailEnabled, s.SMTPHost, s.SMTPPort, s.SMTPUser, s.SMTPFrom, s.EmailTo = req.EmailEnabled, req.SMTPHost, req.SMTPPort, req.SMTPUser, req.SMTPFrom, req.EmailTo
	if req.NtfyToken != nil {
		s.NtfyToken = *req.NtfyToken
	}
	if req.SMTPPassword != nil {
		s.SMTPPassword = *req.SMTPPassword
	}
	if err := db.DB.Save(s).Error; err != nil {
		fail(c, "notifications", err)
		return
	}
	c.JSON(http.StatusOK, notificationView(s))
}

func testNotification(c *gin.Context) {
	errs := service.SendNotification("Hammond test", "Notifications are working. You'll hear from Hammond when maintenance is coming up.")
	msgs := []string{}
	for _, e := range errs {
		msgs = append(msgs, e.Error())
	}
	c.JSON(http.StatusOK, gin.H{"errors": msgs})
}
