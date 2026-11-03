package db

import (
	"time"

	"gorm.io/gorm"
)

// ServiceItem is a recurring maintenance task or a custom reminder for a vehicle.
// Trigger is "first" (whichever of miles / months comes first), "miles" or "date".
type ServiceItem struct {
	Base
	VehicleID      string     `json:"vehicleId" gorm:"index"`
	Name           string     `json:"name"`
	Kind           string     `json:"kind" gorm:"default:service"` // service | custom
	Trigger        string     `json:"trigger" gorm:"default:first"`
	IntervalMiles  int        `json:"intervalMiles"`
	IntervalMonths int        `json:"intervalMonths"`
	LastOdo        int        `json:"lastOdo"`
	LastDate       *time.Time `json:"lastDate"`
	EstCost        float32    `json:"estCost"`
	LeadDays       int        `json:"leadDays" gorm:"default:14"`
	LeadMiles      int        `json:"leadMiles" gorm:"default:500"`
	SnoozedUntil   *time.Time `json:"snoozedUntil"`
	IsActive       bool       `json:"isActive" gorm:"default:true"`
}

// TireSet is a set of four tires (e.g. all-season, winter). One set per vehicle is mounted.
type TireSet struct {
	Base
	VehicleID     string     `json:"vehicleId" gorm:"index"`
	Name          string     `json:"name"`
	Brand         string     `json:"brand"`
	Size          string     `json:"size"`
	IsMounted     bool       `json:"isMounted"`
	InstalledDate *time.Time `json:"installedDate"`
	InstalledOdo  int        `json:"installedOdo"`
	MilesBefore   int        `json:"milesBefore"` // miles the set had before its current mounting
	StoredAt      string     `json:"storedAt"`
	TreadNew      float32    `json:"treadNew" gorm:"default:10"`
}

type TireRotation struct {
	Base
	VehicleID  string    `json:"vehicleId" gorm:"index"`
	TireSetID  string    `json:"tireSetId"`
	Date       time.Time `json:"date"`
	OdoReading int       `json:"odoReading"`
	Pattern    string    `json:"pattern"`
}

// TreadReading stores tread depth (32nds of an inch) and optional pressure per corner.
type TreadReading struct {
	Base
	VehicleID  string    `json:"vehicleId" gorm:"index"`
	TireSetID  string    `json:"tireSetId" gorm:"index"`
	Date       time.Time `json:"date"`
	OdoReading int       `json:"odoReading"`
	FL         float32   `json:"fl"`
	FR         float32   `json:"fr"`
	RL         float32   `json:"rl"`
	RR         float32   `json:"rr"`
	PressureFL float32   `json:"pressureFl"`
	PressureFR float32   `json:"pressureFr"`
	PressureRL float32   `json:"pressureRl"`
	PressureRR float32   `json:"pressureRr"`
	Source     string    `json:"source"` // reading | rotation
}

// Document is a stored vehicle document (insurance card, registration, title, warranty...).
type Document struct {
	Base
	VehicleID    string     `json:"vehicleId" gorm:"index"`
	Type         string     `json:"type"` // insurance | registration | title | warranty | inspection | other
	Title        string     `json:"title"`
	Provider     string     `json:"provider"`
	Number       string     `json:"number"`
	ExpiryDate   *time.Time `json:"expiryDate"`
	AttachmentID string     `json:"attachmentId"`
	Notes        string     `json:"notes"`
}

// NotificationSettings is a single-row table holding notification channel config.
type NotificationSettings struct {
	Base
	NtfyURL      string `json:"ntfyUrl"`
	NtfyTopic    string `json:"ntfyTopic"`
	NtfyToken    string `json:"-"`
	WebhookURL   string `json:"webhookUrl"`
	SMTPHost     string `json:"smtpHost"`
	SMTPPort     int    `json:"smtpPort"`
	SMTPUser     string `json:"smtpUser"`
	SMTPPassword string `json:"-"`
	SMTPFrom     string `json:"smtpFrom"`
	EmailTo      string `json:"emailTo"`
	NtfyEnabled  bool   `json:"ntfyEnabled"`
	EmailEnabled bool   `json:"emailEnabled"`
	HookEnabled  bool   `json:"hookEnabled"`
}

// NotificationLog prevents sending the same reminder more than once per window.
type NotificationLog struct {
	Base
	Key    string    `json:"key" gorm:"index"`
	SentAt time.Time `json:"sentAt"`
}

// UpdateVehicleColumns targets only the vehicles row. Updating through a vehicle
// that has fillups/expenses preloaded would make GORM upsert those associations,
// and Base.BeforeCreate assigns fresh IDs, which duplicates every record.
func UpdateVehicleColumns(vehicleID string) *gorm.DB {
	return DB.Model(&Vehicle{}).Where("id = ?", vehicleID)
}
