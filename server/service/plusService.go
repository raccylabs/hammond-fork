package service

// Hammond+ services: maintenance schedules, reminders, tires, documents,
// VIN decoding, recall checks and notifications.

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/smtp"
	"net/url"
	"sort"
	"strings"
	"time"

	"hammond/db"
	"hammond/models"
)

var plusHTTP = &http.Client{Timeout: 20 * time.Second}

func applyVehicleSpecs(v *db.Vehicle, m models.CreateVehicleRequest) {
	v.Trim = m.Trim
	v.Drivetrain = strings.ToUpper(m.Drivetrain)
	v.OilType = m.OilType
	v.TireSize = m.TireSize
	v.TirePressureFront = m.TirePressureFront
	v.TirePressureRear = m.TirePressureRear
	v.PurchaseDate = m.PurchaseDate
	v.PurchasePrice = m.PurchasePrice
	v.EstimatedValue = m.EstimatedValue
	v.InsurancePremium = m.InsurancePremium
}

func VehicleDisplayName(v *db.Vehicle) string {
	if v.Nickname != "" {
		return v.Nickname
	}
	return strings.TrimSpace(v.Make + " " + v.Model)
}

// ---------------------------------------------------------------- odometer

// CurrentOdometer returns the highest odometer reading recorded anywhere for a vehicle.
func CurrentOdometer(vehicleID string) int {
	row := db.DB.Raw(`SELECT MAX(o) FROM (
		SELECT MAX(odo_reading) AS o FROM fillups WHERE vehicle_id = ?
		UNION ALL SELECT MAX(odo_reading) FROM expenses WHERE vehicle_id = ?
		UNION ALL SELECT MAX(odo_reading) FROM tire_rotations WHERE vehicle_id = ?
		UNION ALL SELECT MAX(odo_reading) FROM tread_readings WHERE vehicle_id = ?
		UNION ALL SELECT MAX(last_odo) FROM service_items WHERE vehicle_id = ?)`,
		vehicleID, vehicleID, vehicleID, vehicleID, vehicleID).Row()
	var v sql.NullInt64
	if err := row.Scan(&v); err != nil {
		return 0
	}
	return int(v.Int64)
}

// MilesPerDay estimates daily distance from the last year of fillups (fallback 30).
func MilesPerDay(vehicleID string) float64 {
	var fills []db.Fillup
	db.DB.Where("vehicle_id = ? AND date >= ?", vehicleID, time.Now().AddDate(-1, 0, 0)).Order("date asc").Find(&fills)
	if len(fills) < 2 {
		return 30
	}
	first, last := fills[0], fills[len(fills)-1]
	days := last.Date.Sub(first.Date).Hours() / 24
	if days < 7 || last.OdoReading <= first.OdoReading {
		return 30
	}
	return float64(last.OdoReading-first.OdoReading) / days
}

// ---------------------------------------------------------------- defaults

var defaultServices = []struct {
	Name          string
	Miles, Months int
	Cost          float32
}{
	{"Oil change", 5000, 6, 85},
	{"Tire rotation", 6000, 12, 30},
	{"Engine air filter", 15000, 24, 45},
	{"Cabin air filter", 15000, 12, 40},
	{"Brake fluid", 30000, 36, 120},
	{"Wiper blades", 0, 12, 35},
}

// SeedServiceDefaults adds the standard maintenance schedule, starting from today.
func SeedServiceDefaults(v *db.Vehicle, odo int) error {
	if cur := CurrentOdometer(v.ID); cur > odo {
		odo = cur
	}
	now := time.Now()
	for _, s := range defaultServices {
		trig := "first"
		if s.Miles == 0 {
			trig = "date"
		}
		item := db.ServiceItem{VehicleID: v.ID, Name: s.Name, Kind: "service", Trigger: trig, IntervalMiles: s.Miles,
			IntervalMonths: s.Months, LastOdo: odo, LastDate: &now, EstCost: s.Cost, LeadDays: 14, LeadMiles: 500, IsActive: true}
		if err := db.DB.Create(&item).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureTireSet(v *db.Vehicle, odo int) {
	var count int64
	db.DB.Model(&db.TireSet{}).Where("vehicle_id = ?", v.ID).Count(&count)
	if count == 0 {
		now := time.Now()
		db.DB.Create(&db.TireSet{VehicleID: v.ID, Name: "Current tires", Size: v.TireSize, IsMounted: true,
			InstalledDate: &now, InstalledOdo: maxInt(odo, CurrentOdometer(v.ID)), TreadNew: 10})
	}
}

// SeedVehicleDefaults runs when a vehicle is created.
func SeedVehicleDefaults(v *db.Vehicle, odo int) {
	if err := SeedServiceDefaults(v, odo); err != nil {
		log.Println("seed services:", err)
	}
	ensureTireSet(v, odo)
}

// ---------------------------------------------------------------- reminders

type Reminder struct {
	Key           string     `json:"key"`
	Kind          string     `json:"kind"` // service | custom | document | recall
	RefID         string     `json:"refId"`
	VehicleID     string     `json:"vehicleId"`
	VehicleName   string     `json:"vehicleName"`
	Title         string     `json:"title"`
	Level         int        `json:"level"` // 0 ok, 1 soon, 2 due/overdue
	Status        string     `json:"status"`
	Pct           float64    `json:"pct"`
	MilesLeft     *int       `json:"milesLeft"`
	DaysLeft      *int       `json:"daysLeft"`
	DueOdo        int        `json:"dueOdo"`
	DueDate       *time.Time `json:"dueDate"`
	ProjectedDate *time.Time `json:"projectedDate"`
	SnoozedUntil  *time.Time `json:"snoozedUntil"`
	EstCost       float32    `json:"estCost"`
}

func levelStatus(l int) string {
	switch l {
	case 2:
		return "due"
	case 1:
		return "soon"
	}
	return "ok"
}

func serviceReminder(v *db.Vehicle, s db.ServiceItem, odo int, mpd float64, now time.Time) Reminder {
	r := Reminder{Key: "svc:" + s.ID, Kind: s.Kind, RefID: s.ID, VehicleID: v.ID, VehicleName: VehicleDisplayName(v), Title: s.Name, EstCost: s.EstCost}
	if r.Kind == "" {
		r.Kind = "service"
	}
	pm, pd := 0.0, 0.0
	if s.Trigger != "date" && s.IntervalMiles > 0 {
		due := s.LastOdo + s.IntervalMiles
		left := due - odo
		r.DueOdo = due
		r.MilesLeft = &left
		pm = float64(odo-s.LastOdo) / float64(s.IntervalMiles)
		if mpd > 0 {
			p := now.Add(time.Duration(float64(left)/mpd*24) * time.Hour)
			r.ProjectedDate = &p
		}
	}
	if s.Trigger != "miles" && s.IntervalMonths > 0 && s.LastDate != nil {
		dd := s.LastDate.AddDate(0, s.IntervalMonths, 0)
		left := int(math.Floor(dd.Sub(now).Hours() / 24))
		r.DueDate = &dd
		r.DaysLeft = &left
		pd = now.Sub(*s.LastDate).Hours() / 24 / (float64(s.IntervalMonths) * 30.44)
	}
	r.Pct = math.Max(0, math.Min(1.5, math.Max(pm, pd)))
	soon := r.Pct >= 0.9 || (r.MilesLeft != nil && *r.MilesLeft <= s.LeadMiles) || (r.DaysLeft != nil && *r.DaysLeft <= s.LeadDays)
	switch {
	case r.Pct >= 1:
		r.Level = 2
	case soon:
		r.Level = 1
	}
	r.Status = levelStatus(r.Level)
	if s.SnoozedUntil != nil && s.SnoozedUntil.After(now) {
		r.Status = "snoozed"
		r.SnoozedUntil = s.SnoozedUntil
	}
	return r
}

func documentReminder(v *db.Vehicle, d db.Document, now time.Time) *Reminder {
	if d.ExpiryDate == nil {
		return nil
	}
	left := int(math.Floor(d.ExpiryDate.Sub(now).Hours() / 24))
	title := d.Title
	if title == "" {
		title = strings.Title(d.Type)
	}
	r := Reminder{Key: "doc:" + d.ID, Kind: "document", RefID: d.ID, VehicleID: v.ID, VehicleName: VehicleDisplayName(v),
		Title: title + " renewal", DaysLeft: &left, DueDate: d.ExpiryDate, Pct: math.Max(0, math.Min(1.5, 1-float64(left)/365))}
	if left < 0 {
		r.Level = 2
	} else if left <= 30 {
		r.Level = 1
	}
	r.Status = levelStatus(r.Level)
	return &r
}

// RemindersForVehicle returns every reminder for a vehicle (services, custom, documents, recalls).
func RemindersForVehicle(v *db.Vehicle) []Reminder {
	now := time.Now()
	odo := CurrentOdometer(v.ID)
	mpd := MilesPerDay(v.ID)
	var out []Reminder
	var items []db.ServiceItem
	db.DB.Where("vehicle_id = ? AND is_active = ?", v.ID, true).Find(&items)
	for i := range items {
		// Items created before any odometer reading existed start counting from the first real reading.
		if items[i].LastOdo == 0 && odo > 0 && items[i].IntervalMiles > 0 {
			items[i].LastOdo = odo
			db.DB.Model(&items[i]).Update("last_odo", odo)
		}
	}
	for _, s := range items {
		out = append(out, serviceReminder(v, s, odo, mpd, now))
	}
	var docs []db.Document
	db.DB.Where("vehicle_id = ?", v.ID).Find(&docs)
	for _, d := range docs {
		if r := documentReminder(v, d, now); r != nil {
			out = append(out, *r)
		}
	}
	if v.RecallCount > 0 {
		out = append(out, Reminder{Key: "recall:" + v.ID, Kind: "recall", RefID: v.ID, VehicleID: v.ID, VehicleName: VehicleDisplayName(v),
			Title: fmt.Sprintf("%d open recall%s", v.RecallCount, plural(v.RecallCount)), Level: 1, Status: "soon", Pct: 0.95})
	}
	return out
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func SortReminders(list []Reminder) {
	sort.SliceStable(list, func(i, j int) bool {
		si, sj := list[i].Status == "snoozed", list[j].Status == "snoozed"
		if si != sj {
			return !si
		}
		if list[i].Level != list[j].Level {
			return list[i].Level > list[j].Level
		}
		return list[i].Pct > list[j].Pct
	})
}

func RemindersForUser(userID string) ([]Reminder, error) {
	vehicles, err := db.GetUserVehicles(userID)
	if err != nil {
		return nil, err
	}
	out := []Reminder{}
	for i := range *vehicles {
		out = append(out, RemindersForVehicle(&(*vehicles)[i])...)
	}
	SortReminders(out)
	return out, nil
}

// ---------------------------------------------------------------- services

func CompleteService(item *db.ServiceItem, date time.Time, odo int, amount float32, comments, userID string) (*db.Expense, error) {
	item.LastOdo = odo
	item.LastDate = &date
	item.SnoozedUntil = nil
	if err := db.DB.Save(item).Error; err != nil {
		return nil, err
	}
	if amount <= 0 {
		return nil, nil
	}
	user, err := db.GetUserById(userID)
	if err != nil {
		return nil, err
	}
	exp := db.Expense{VehicleID: item.VehicleID, Amount: amount, OdoReading: odo, Comments: comments, ExpenseType: item.Name,
		ServiceItemID: item.ID, UserID: userID, Date: date, Currency: user.Currency, DistanceUnit: user.DistanceUnit, Source: "service"}
	return &exp, db.DB.Create(&exp).Error
}

func SnoozeService(item *db.ServiceItem, days int) error {
	if days <= 0 {
		item.SnoozedUntil = nil
	} else {
		t := time.Now().AddDate(0, 0, days)
		item.SnoozedUntil = &t
	}
	return db.DB.Save(item).Error
}

// ---------------------------------------------------------------- tires

type TireOverview struct {
	Sets      []db.TireSet      `json:"sets"`
	Rotations []db.TireRotation `json:"rotations"`
	Readings  []db.TreadReading `json:"readings"`
	Pattern   string            `json:"pattern"`
	Odometer  int               `json:"odometer"`
	MilesDay  float64           `json:"milesPerDay"`
}

var rotationPatterns = map[string]struct {
	Name string
	Perm func(fl, fr, rl, rr float32) (float32, float32, float32, float32)
}{
	"FWD": {"Forward cross", func(fl, fr, rl, rr float32) (float32, float32, float32, float32) { return rl, rr, fr, fl }},
	"AWD": {"Front to rear, same side", func(fl, fr, rl, rr float32) (float32, float32, float32, float32) { return rl, rr, fl, fr }},
	"4WD": {"Rearward cross", func(fl, fr, rl, rr float32) (float32, float32, float32, float32) { return rr, rl, fl, fr }},
	"RWD": {"Rearward cross", func(fl, fr, rl, rr float32) (float32, float32, float32, float32) { return rr, rl, fl, fr }},
}

func patternFor(drivetrain string) string {
	if _, ok := rotationPatterns[drivetrain]; ok {
		return drivetrain
	}
	return "FWD"
}

func GetTireOverview(v *db.Vehicle) TireOverview {
	ensureTireSet(v, 0)
	o := TireOverview{Pattern: rotationPatterns[patternFor(v.Drivetrain)].Name, Odometer: CurrentOdometer(v.ID), MilesDay: MilesPerDay(v.ID)}
	if o.Odometer > 0 {
		db.DB.Model(&db.TireSet{}).Where("vehicle_id = ? AND is_mounted = ? AND installed_odo = 0", v.ID, true).Update("installed_odo", o.Odometer)
	}
	db.DB.Where("vehicle_id = ?", v.ID).Order("is_mounted desc, created_at asc").Find(&o.Sets)
	db.DB.Where("vehicle_id = ?", v.ID).Order("date desc").Find(&o.Rotations)
	db.DB.Where("vehicle_id = ?", v.ID).Order("date asc, created_at asc").Find(&o.Readings)
	return o
}

func mountedSet(vehicleID string) (*db.TireSet, error) {
	var s db.TireSet
	err := db.DB.Where("vehicle_id = ? AND is_mounted = ?", vehicleID, true).First(&s).Error
	return &s, err
}

func MountTireSet(v *db.Vehicle, setID string, date time.Time, odo int) error {
	var target db.TireSet
	if err := db.DB.Where("id = ? AND vehicle_id = ?", setID, v.ID).First(&target).Error; err != nil {
		return err
	}
	if target.IsMounted {
		return nil
	}
	if cur, err := mountedSet(v.ID); err == nil {
		cur.MilesBefore += maxInt(0, odo-cur.InstalledOdo)
		cur.IsMounted = false
		if err := db.DB.Save(cur).Error; err != nil {
			return err
		}
	}
	target.IsMounted = true
	target.InstalledOdo = odo
	target.InstalledDate = &date
	target.StoredAt = ""
	return db.DB.Save(&target).Error
}

func latestReading(setID string) (*db.TreadReading, error) {
	var r db.TreadReading
	err := db.DB.Where("tire_set_id = ?", setID).Order("date desc, created_at desc").First(&r).Error
	return &r, err
}

// LogRotation records a rotation, carries the latest tread reading to the new corners,
// and resets the "rotation" service item if there is one.
func LogRotation(v *db.Vehicle, date time.Time, odo int) (*db.TireRotation, error) {
	set, err := mountedSet(v.ID)
	if err != nil {
		return nil, errors.New("no mounted tire set")
	}
	p := rotationPatterns[patternFor(v.Drivetrain)]
	rot := db.TireRotation{VehicleID: v.ID, TireSetID: set.ID, Date: date, OdoReading: odo, Pattern: p.Name}
	if err := db.DB.Create(&rot).Error; err != nil {
		return nil, err
	}
	if last, err := latestReading(set.ID); err == nil {
		fl, fr, rl, rr := p.Perm(last.FL, last.FR, last.RL, last.RR)
		db.DB.Create(&db.TreadReading{VehicleID: v.ID, TireSetID: set.ID, Date: date, OdoReading: odo, FL: fl, FR: fr, RL: rl, RR: rr, Source: "rotation"})
	}
	var items []db.ServiceItem
	db.DB.Where("vehicle_id = ? AND lower(name) LIKE ?", v.ID, "%rotation%").Find(&items)
	for i := range items {
		items[i].LastOdo = odo
		items[i].LastDate = &date
		items[i].SnoozedUntil = nil
		db.DB.Save(&items[i])
	}
	return &rot, nil
}

func AddTreadReading(v *db.Vehicle, r *db.TreadReading) error {
	set, err := mountedSet(v.ID)
	if err != nil {
		return errors.New("no mounted tire set")
	}
	r.VehicleID = v.ID
	r.TireSetID = set.ID
	if r.Source == "" {
		r.Source = "reading"
	}
	return db.DB.Create(r).Error
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ---------------------------------------------------------------- VIN + recalls

type VinInfo struct {
	VIN        string       `json:"vin"`
	Make       string       `json:"make"`
	Model      string       `json:"model"`
	Year       int          `json:"year"`
	Trim       string       `json:"trim"`
	Engine     string       `json:"engine"`
	EngineSize float32      `json:"engineSize"`
	FuelType   *db.FuelType `json:"fuelType"`
	FuelLabel  string       `json:"fuelLabel"`
	Drivetrain string       `json:"drivetrain"`
	Body       string       `json:"body"`
	Warning    string       `json:"warning"`
}

func titleCase(s string) string {
	s = strings.TrimSpace(s)
	if s == strings.ToUpper(s) && len(s) > 3 {
		return strings.Title(strings.ToLower(s))
	}
	return s
}

func DecodeVIN(vin string) (*VinInfo, error) {
	vin = strings.ToUpper(strings.TrimSpace(vin))
	if len(vin) != 17 {
		return nil, errors.New("a VIN is 17 characters")
	}
	resp, err := plusHTTP.Get("https://vpic.nhtsa.dot.gov/api/vehicles/DecodeVinValuesExtended/" + url.PathEscape(vin) + "?format=json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload struct {
		Results []map[string]string `json:"Results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Results) == 0 {
		return nil, errors.New("no result from NHTSA")
	}
	r := payload.Results[0]
	info := &VinInfo{VIN: vin, Make: titleCase(r["Make"]), Model: r["Model"], Trim: r["Trim"], Body: r["BodyClass"]}
	fmt.Sscanf(r["ModelYear"], "%d", &info.Year)
	var disp float64
	fmt.Sscanf(r["DisplacementL"], "%f", &disp)
	info.EngineSize = float32(math.Round(disp*10) / 10)
	parts := []string{}
	if disp > 0 {
		parts = append(parts, fmt.Sprintf("%.1fL", disp))
	}
	if r["EngineConfiguration"] != "" || r["EngineCylinders"] != "" {
		cfg := strings.Split(r["EngineConfiguration"], "-")[0]
		parts = append(parts, strings.TrimSpace(cfg+" "+r["EngineCylinders"]))
	}
	if strings.Contains(strings.ToLower(r["Turbo"]), "yes") {
		parts = append(parts, "turbo")
	}
	info.Engine = strings.Join(parts, " ")
	fuel := strings.ToLower(r["FuelTypePrimary"])
	info.FuelLabel = r["FuelTypePrimary"]
	var ft db.FuelType = -1
	switch {
	case strings.Contains(fuel, "electric"):
		ft = db.ELECTRIC
	case strings.Contains(fuel, "diesel"):
		ft = db.DIESEL
	case strings.Contains(fuel, "ethanol") || strings.Contains(fuel, "e85"):
		ft = db.ETHANOL
	case strings.Contains(fuel, "compressed natural"):
		ft = db.CNG
	case strings.Contains(fuel, "liquefied petroleum") || strings.Contains(fuel, "propane"):
		ft = db.LPG
	case strings.Contains(fuel, "gasoline"):
		ft = db.PETROL
	}
	if ft >= 0 {
		info.FuelType = &ft
	}
	d := strings.ToUpper(r["DriveType"])
	switch {
	case strings.Contains(d, "AWD") || strings.Contains(d, "ALL-WHEEL") || strings.Contains(d, "ALL WHEEL"):
		info.Drivetrain = "AWD"
	case strings.Contains(d, "4WD") || strings.Contains(d, "4X4") || strings.Contains(d, "4-WHEEL"):
		info.Drivetrain = "4WD"
	case strings.Contains(d, "RWD") || strings.Contains(d, "REAR"):
		info.Drivetrain = "RWD"
	case strings.Contains(d, "FWD") || strings.Contains(d, "FRONT"):
		info.Drivetrain = "FWD"
	}
	if code := r["ErrorCode"]; code != "" && code != "0" {
		info.Warning = r["ErrorText"]
	}
	if info.Make == "" && info.Model == "" {
		return nil, errors.New("NHTSA could not decode this VIN")
	}
	return info, nil
}

type RecallItem struct {
	Campaign    string `json:"campaign"`
	Component   string `json:"component"`
	Summary     string `json:"summary"`
	Consequence string `json:"consequence"`
	Remedy      string `json:"remedy"`
	ReportDate  string `json:"reportDate"`
	Dismissed   bool   `json:"dismissed"`
}

func dismissedSet(v *db.Vehicle) map[string]bool {
	set := map[string]bool{}
	for _, c := range strings.Split(v.RecallDismissed, ",") {
		if c = strings.TrimSpace(c); c != "" {
			set[c] = true
		}
	}
	return set
}

func GetRecalls(v *db.Vehicle) []RecallItem {
	items := []RecallItem{}
	if v.RecallData != "" {
		json.Unmarshal([]byte(v.RecallData), &items)
	}
	dis := dismissedSet(v)
	for i := range items {
		items[i].Dismissed = dis[items[i].Campaign]
	}
	return items
}

func recountRecalls(v *db.Vehicle) {
	n := 0
	for _, r := range GetRecalls(v) {
		if !r.Dismissed {
			n++
		}
	}
	v.RecallCount = n
}

// CheckRecalls fetches recalls for the vehicle's make/model/year from NHTSA.
// NHTSA's public API is per model year, not per VIN, so the user can dismiss
// campaigns that don't apply or have already been fixed.
func CheckRecalls(v *db.Vehicle) ([]RecallItem, error) {
	if v.Make == "" || v.Model == "" || v.YearOfManufacture == 0 {
		return nil, errors.New("make, model and year are needed to check recalls")
	}
	u := fmt.Sprintf("https://api.nhtsa.gov/recalls/recallsByVehicle?make=%s&model=%s&modelYear=%d",
		url.QueryEscape(v.Make), url.QueryEscape(v.Model), v.YearOfManufacture)
	resp, err := plusHTTP.Get(u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload struct {
		Results []struct {
			NHTSACampaignNumber string `json:"NHTSACampaignNumber"`
			Component           string `json:"Component"`
			Summary             string `json:"Summary"`
			Consequence         string `json:"Consequence"`
			Remedy              string `json:"Remedy"`
			ReportReceivedDate  string `json:"ReportReceivedDate"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	items := []RecallItem{}
	for _, r := range payload.Results {
		items = append(items, RecallItem{Campaign: r.NHTSACampaignNumber, Component: r.Component, Summary: r.Summary,
			Consequence: r.Consequence, Remedy: r.Remedy, ReportDate: r.ReportReceivedDate})
	}
	data, _ := json.Marshal(items)
	now := time.Now()
	v.RecallData = string(data)
	v.RecallCheckedAt = &now
	recountRecalls(v)
	err = db.UpdateVehicleColumns(v.ID).Updates(map[string]interface{}{"recall_data": v.RecallData, "recall_checked_at": now, "recall_count": v.RecallCount}).Error
	return GetRecalls(v), err
}

func SetRecallDismissed(v *db.Vehicle, campaign string, dismissed bool) error {
	set := dismissedSet(v)
	if dismissed {
		set[campaign] = true
	} else {
		delete(set, campaign)
	}
	keys := []string{}
	for k := range set {
		keys = append(keys, k)
	}
	v.RecallDismissed = strings.Join(keys, ",")
	recountRecalls(v)
	return db.UpdateVehicleColumns(v.ID).Updates(map[string]interface{}{"recall_dismissed": v.RecallDismissed, "recall_count": v.RecallCount}).Error
}

func RunRecallChecks() {
	var vehicles []db.Vehicle
	db.DB.Where("recall_checked_at IS NULL OR recall_checked_at < ?", time.Now().AddDate(0, 0, -7)).Find(&vehicles)
	for i := range vehicles {
		if _, err := CheckRecalls(&vehicles[i]); err != nil {
			log.Println("recall check", vehicles[i].ID, err)
		}
		time.Sleep(2 * time.Second)
	}
}

// ---------------------------------------------------------------- notifications

func GetNotificationSettings() *db.NotificationSettings {
	var s db.NotificationSettings
	if err := db.DB.First(&s).Error; err != nil {
		s = db.NotificationSettings{NtfyURL: "https://ntfy.sh", SMTPPort: 587}
		db.DB.Create(&s)
	}
	return &s
}

func SendNotification(title, body string) []error {
	s := GetNotificationSettings()
	var errs []error
	if s.NtfyEnabled && s.NtfyTopic != "" {
		base := strings.TrimRight(s.NtfyURL, "/")
		if base == "" {
			base = "https://ntfy.sh"
		}
		req, _ := http.NewRequest("POST", base+"/"+url.PathEscape(s.NtfyTopic), strings.NewReader(body))
		req.Header.Set("Title", title)
		req.Header.Set("Tags", "red_car")
		if s.NtfyToken != "" {
			req.Header.Set("Authorization", "Bearer "+s.NtfyToken)
		}
		if resp, err := plusHTTP.Do(req); err != nil {
			errs = append(errs, fmt.Errorf("ntfy: %w", err))
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 300 {
				errs = append(errs, fmt.Errorf("ntfy: HTTP %d", resp.StatusCode))
			}
		}
	}
	if s.HookEnabled && s.WebhookURL != "" {
		payload, _ := json.Marshal(map[string]string{"title": title, "message": body, "source": "hammond"})
		if resp, err := plusHTTP.Post(s.WebhookURL, "application/json", bytes.NewReader(payload)); err != nil {
			errs = append(errs, fmt.Errorf("webhook: %w", err))
		} else {
			resp.Body.Close()
			if resp.StatusCode >= 300 {
				errs = append(errs, fmt.Errorf("webhook: HTTP %d", resp.StatusCode))
			}
		}
	}
	if s.EmailEnabled && s.SMTPHost != "" && s.EmailTo != "" {
		from := s.SMTPFrom
		if from == "" {
			from = s.SMTPUser
		}
		msg := "From: " + from + "\r\nTo: " + s.EmailTo + "\r\nSubject: " + title + "\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + body + "\r\n"
		var auth smtp.Auth
		if s.SMTPUser != "" {
			auth = smtp.PlainAuth("", s.SMTPUser, s.SMTPPassword, s.SMTPHost)
		}
		port := s.SMTPPort
		if port == 0 {
			port = 587
		}
		if err := smtp.SendMail(fmt.Sprintf("%s:%d", s.SMTPHost, port), auth, from, strings.Split(s.EmailTo, ","), []byte(msg)); err != nil {
			errs = append(errs, fmt.Errorf("email: %w", err))
		}
	}
	return errs
}

func reminderMessage(r Reminder) (string, string) {
	title := r.Title + " · " + r.VehicleName
	var parts []string
	if r.MilesLeft != nil {
		if *r.MilesLeft < 0 {
			parts = append(parts, fmt.Sprintf("overdue by %d mi", -*r.MilesLeft))
		} else {
			parts = append(parts, fmt.Sprintf("%d mi left", *r.MilesLeft))
		}
	}
	if r.DaysLeft != nil {
		if *r.DaysLeft < 0 {
			parts = append(parts, fmt.Sprintf("%d days overdue", -*r.DaysLeft))
		} else {
			parts = append(parts, fmt.Sprintf("due in %d days", *r.DaysLeft))
		}
	}
	if len(parts) == 0 {
		parts = append(parts, "needs attention")
	}
	return title, strings.Join(parts, ", ") + ". Open Hammond to log it or snooze."
}

// RunReminderNotifications sends each due/soon reminder at most once a week per status.
func RunReminderNotifications() {
	var vehicles []db.Vehicle
	db.DB.Find(&vehicles)
	for i := range vehicles {
		for _, r := range RemindersForVehicle(&vehicles[i]) {
			if r.Level == 0 || r.Status == "snoozed" {
				continue
			}
			key := r.Key + ":" + r.Status
			var count int64
			db.DB.Model(&db.NotificationLog{}).Where("key = ? AND sent_at > ?", key, time.Now().AddDate(0, 0, -7)).Count(&count)
			if count > 0 {
				continue
			}
			title, body := reminderMessage(r)
			if errs := SendNotification(title, body); len(errs) > 0 {
				log.Println("notify:", errs)
			}
			db.DB.Create(&db.NotificationLog{Key: key, SentAt: time.Now()})
		}
	}
}
