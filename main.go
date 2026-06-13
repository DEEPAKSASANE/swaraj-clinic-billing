package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var db *sql.DB

// Change database name if needed.
const localConnStr = "host=localhost port=5432 user=postgres password=root dbname=deepak_new_kalva_online_2 sslmode=disable"

type Dashboard struct {
	CashTotal     float64
	OnlineTotal   float64
	ExpenseTotal  float64
	FinalBalance  float64
	DoctorPresent int
	SisterPresent int
}

type Attendance struct {
	ID             int
	ClinicName     string
	StaffName      string
	Role           string
	AttendanceDate string
	InTime         string
	OutTime        string
	Hours          string
	Status         string
	Remark         string
	SelfiePath     string
}

type Collection struct {
	ID           int
	ClinicName   string
	Date         string
	Cash         float64
	Online       float64
	Total        float64
	Expense      float64
	FinalBalance float64
	CashInHand   float64
	Reason       string
	Remark       string
	EnteredBy    string
}

type InvoiceSummary struct {
	SrNo           int
	ClinicName     string
	InvoiceID      int
	CustomerName   string
	Age            sql.NullInt64
	MobileNo       string
	CreatedAt      string
	TotalAmountSum float64
}

type FullInvoiceData struct {
	SrNo             int
	ClinicName       string
	InvoiceID        int
	CustomerName     string
	Age              int
	MobileNo         string
	CreatedAt        time.Time
	TestID           int
	TestName         string
	Price            float64
	Discount         float64
	TotalAmount      float64
	TotalAmountWords string
}

type TestItem struct {
	TestName         string
	Price            float64
	Discount         float64
	TotalAmount      float64
	TotalAmountWords string
}

type Invoice struct {
	InvoiceNo        int
	ClinicName       string
	InvoiceDatetime  string
	CustomerName     string
	Gender           string
	Age              int
	Mobile           string
	Address          string
	Tests            []TestItem
	TotalAmount      float64
	TotalAmountWords string
}

type PageData struct {
	Data       []FullInvoiceData
	Summaries  []InvoiceSummary
	GrandTotal float64
}
type DailyReport struct {
	ClinicName string
	ReportDate string
	DoctorName string
	SisterName string

	DrInTime  string
	DrOutTime string

	SisterInTime  string
	SisterOutTime string

	TotalPatient  int
	TotalBilling  float64
	CashBilling   float64
	OnlineBilling float64
	Expense       float64
	CashInHand    float64
}

func main() {
	var err error

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = localConnStr
	}

	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Database connection failed: ", err)
	}

	fmt.Println("CONNECTED DB:", connStr)
	createTables()

	r := gin.Default()

	r.SetFuncMap(template.FuncMap{
		"safe": func(s string) template.HTML {
			return template.HTML(s)
		},
		"add": func(a, b int) int {
			return a + b
		},
		"formatFloat": func(f float64) string {
			return strconv.FormatFloat(f, 'f', 2, 64)
		},
	})

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	// Login / Register
	r.GET("/", func(c *gin.Context) { c.Redirect(http.StatusSeeOther, "/login") })
	r.GET("/login", loginPage)
	r.POST("/login", login)
	r.GET("/logout", logout)

	// Register only admin can open after login
	r.GET("/register", registerPage)
	r.POST("/register", registerUser)
	// Dashboard
	r.GET("/dashboard", adminOnly(dashboard))

	// Staff Dashboard
	r.GET("/staff-dashboard", loginRequired(staffDashboardPage))
	r.GET("/voice-report", loginRequired(voiceReportPage))
	r.GET("/my-report", loginRequired(myReportPage))

	// Attendance Check In / Check Out
	// /attendance will open Check In page to avoid missing attendance.html template error.
	r.GET("/attendance", loginRequired(attendanceInTimePage))

	r.GET("/attendance/intime", loginRequired(attendanceInTimePage))
	r.POST("/attendance/intime", loginRequired(saveInTime))

	r.GET("/attendance/outtime", loginRequired(attendanceOutTimePage))
	r.POST("/attendance/outtime", loginRequired(saveOutTime))

	// Extra URLs for HTML form actions
	r.GET("/attendance/checkin", loginRequired(attendanceInTimePage))
	r.POST("/attendance/checkin", loginRequired(saveInTime))

	r.GET("/attendance/checkout", loginRequired(attendanceOutTimePage))
	r.POST("/attendance/checkout", loginRequired(saveOutTime))

	// Admin only attendance management
	r.GET("/attendance-report", adminOnly(attendanceReportPage))
	r.GET("/attendance-update/:id", adminOnly(showUpdateAttendancePage))
	r.POST("/attendance-update/:id", adminOnly(updateAttendance))
	r.GET("/attendance-delete/:id", adminOnly(deleteAttendance))

	// Daily Collection - Staff and Admin can add collection
	r.GET("/collection", loginRequired(collectionPage))
	r.POST("/collection", loginRequired(saveCollection))

	// Admin only collection management/report
	r.GET("/collection-report", adminOnly(collectionReportPage))
	r.GET("/collection-update/:id", adminOnly(showUpdateCollectionPage))
	r.POST("/collection-update/:id", adminOnly(updateCollection))
	r.GET("/collection-delete/:id", adminOnly(deleteCollection))

	// Online Collection - Admin only
	r.GET("/collection-online-report", adminOnly(collectionOnlineReportPage))
	r.GET("/collection-deepak", adminOnly(collectionDeepakPage))
	r.POST("/collection-deepak", adminOnly(saveCollectionDeepak))
	r.GET("/collection-report-deepak", adminOnly(collectionReportDeepakPage))
	r.GET("/collection-delete-deepak/:id", adminOnly(deleteCollectionDeepak))

	// Invoice billing - Staff and Admin
	r.GET("/form", loginRequired(func(c *gin.Context) {
		c.HTML(http.StatusOK, "form.html", nil)
	}))
	r.POST("/submit", loginRequired(submitInvoice))
	r.GET("/print", loginRequired(displayAllInvoiceDataHandler()))
	r.GET("/filter-invoices", loginRequired(filterInvoicesHandler()))
	r.GET("/invoice/:id", loginRequired(getInvoiceByID()))

	r.GET("/invoice-search-page", loginRequired(func(c *gin.Context) {
		c.HTML(http.StatusOK, "invoice_search_by_id.html", nil)
	}))

	r.GET("/invoice-search", loginRequired(func(c *gin.Context) {
		id := strings.TrimSpace(c.Query("id"))
		if id == "" {
			c.String(http.StatusBadRequest, "Invoice ID is required")
			return
		}
		c.Redirect(http.StatusSeeOther, "/invoice/"+id)
	}))

	// Daily Report - Staff and Admin
	r.GET("/daily-report", loginRequired(dailyReportPage))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Println("Server running: http://localhost:" + port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

func getCurrentRole(c *gin.Context) string {
	role, err := c.Cookie("role")
	if err != nil {
		return ""
	}
	return strings.ToLower(role)
}

func getCurrentUsername(c *gin.Context) string {
	username, err := c.Cookie("username")
	if err != nil {
		return ""
	}
	return username
}

func getCurrentClinicName(c *gin.Context) string {
	clinicName, err := c.Cookie("clinic_name")
	if err != nil {
		return ""
	}
	return clinicName
}

func clinicWhere(c *gin.Context, column string, prefix string, args *[]interface{}) string {
	role := getCurrentRole(c)
	clinicName := getCurrentClinicName(c)
	if role == "superadmin" || clinicName == "" {
		return ""
	}
	*args = append(*args, clinicName)
	return fmt.Sprintf(" %s %s = $%d ", prefix, column, len(*args))
}

func loginRequired(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if getCurrentRole(c) == "" {
			c.Redirect(http.StatusFound, "/login")
			return
		}
		next(c)
	}
}

func adminOnly(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := getCurrentRole(c)
		if role != "admin" && role != "superadmin" {
			c.Redirect(http.StatusFound, "/staff-dashboard")
			return
		}
		next(c)
	}
}

func staffDashboardPage(c *gin.Context) { c.HTML(http.StatusOK, "staff_dashboard.html", nil) }
func voiceReportPage(c *gin.Context)    { c.HTML(http.StatusOK, "voice_report.html", nil) }

func myReportPage(c *gin.Context) {
	role := getCurrentRole(c)
	username := getCurrentUsername(c)
	clinicName := getCurrentClinicName(c)

	query := `
		SELECT
			i.id,
			i.staff_name,
			i.role,
			i.attendance_date,
			COALESCE(TO_CHAR(i.in_time, 'HH24:MI'), ''),
			COALESCE(TO_CHAR(o.out_time, 'HH24:MI'), ''),
			COALESCE(CAST(EXTRACT(EPOCH FROM (o.out_time - i.in_time)) / 60 AS INTEGER), 0),
			'Present',
			'',
			COALESCE(i.clinic_name, ''),
			COALESCE(i.selfie_path, '')
		FROM attendance_intime i
		LEFT JOIN attendance_outtime o
		ON LOWER(i.staff_name)=LOWER(o.staff_name)
		AND i.attendance_date=o.attendance_date
		AND i.clinic_name=o.clinic_name
		WHERE i.clinic_name=$3
		AND (LOWER(i.role)=LOWER($1) OR LOWER(i.staff_name)=LOWER($2))
		ORDER BY i.attendance_date DESC, i.id DESC
	`

	rows, err := db.Query(query, role, username, clinicName)
	if err != nil {
		c.String(http.StatusInternalServerError, "My report error: %v", err)
		return
	}
	defer rows.Close()

	var attendance []Attendance
	for rows.Next() {
		var a Attendance
		var mins int
		var d time.Time
		if err := rows.Scan(&a.ID, &a.StaffName, &a.Role, &d, &a.InTime, &a.OutTime, &mins, &a.Status, &a.Remark, &a.ClinicName, &a.SelfiePath); err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		a.AttendanceDate = d.Format("2006-01-02")
		a.Hours = minutesToHours(mins)
		attendance = append(attendance, a)
	}

	c.HTML(http.StatusOK, "attendance_report.html", gin.H{"Attendance": attendance, "ClinicName": clinicName})
}

func loginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

func login(c *gin.Context) {
	clinicName := strings.TrimSpace(c.PostForm("clinic_name"))
	username := strings.TrimSpace(c.PostForm("username"))
	password := strings.TrimSpace(c.PostForm("password"))

	if clinicName == "" || username == "" || password == "" {
		c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Please select clinic and enter username/password"})
		return
	}

	var id int
	var role string
	var dbClinicName string

	err := db.QueryRow(`
		SELECT id, COALESCE(role,'doctor'), COALESCE(clinic_name,'')
		FROM users
		WHERE username=$1
		AND password=$2
		AND (clinic_name=$3 OR role='superadmin')
	`, username, password, clinicName).Scan(&id, &role, &dbClinicName)

	if err != nil {
		c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Invalid clinic, username or password"})
		return
	}

	role = strings.ToLower(role)
	if role == "superadmin" {
		dbClinicName = clinicName
	}

	c.SetCookie("username", username, 3600*8, "/", "", false, false)
	c.SetCookie("role", role, 3600*8, "/", "", false, false)
	c.SetCookie("clinic_name", dbClinicName, 3600*8, "/", "", false, false)

	if role == "admin" || role == "superadmin" {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}
	c.Redirect(http.StatusFound, "/staff-dashboard")
}

func logout(c *gin.Context) {
	c.SetCookie("username", "", -1, "/", "", false, false)
	c.SetCookie("role", "", -1, "/", "", false, false)
	c.SetCookie("clinic_name", "", -1, "/", "", false, false)
	c.Redirect(http.StatusFound, "/login")
}

func registerPage(c *gin.Context) {
	c.HTML(http.StatusOK, "register.html", nil)
}

func registerUser(c *gin.Context) {

	name := strings.TrimSpace(c.PostForm("name"))
	address := strings.TrimSpace(c.PostForm("address"))
	mobile := strings.TrimSpace(c.PostForm("mobile"))
	username := strings.TrimSpace(c.PostForm("username"))
	password := strings.TrimSpace(c.PostForm("password"))
	role := strings.ToLower(strings.TrimSpace(c.PostForm("role")))
	clinicName := strings.TrimSpace(c.PostForm("clinic_name"))
	if role != "admin" && role != "doctor" && role != "sister" {
		role = "doctor"
	}

	if name == "" || address == "" || mobile == "" || username == "" || password == "" || clinicName == "" {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Error": "All fields are required",
		})
		return
	}

	var existingID int
	err := db.QueryRow(
		"SELECT id FROM users WHERE username=$1",
		username,
	).Scan(&existingID)

	if err == nil {
		c.HTML(http.StatusBadRequest, "register.html", gin.H{
			"Error": "Username already exists. Please use another username.",
		})
		return
	}

	var newUserID int

	err = db.QueryRow(`
		INSERT INTO users
		(name, address, mobile, username, password, role, clinic_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`,
		name,
		address,
		mobile,
		username,
		password,
		role,
		clinicName,
	).Scan(&newUserID)

	if err != nil {
		fmt.Println("REGISTER DB ERROR:", err)

		c.HTML(http.StatusInternalServerError, "register.html", gin.H{
			"Error": err.Error(),
		})
		return
	}

	fmt.Println("USER REGISTERED SUCCESSFULLY ID:", newUserID)

	c.Redirect(http.StatusSeeOther, "/login")
}
func dashboard(c *gin.Context) {
	d := Dashboard{}
	clinicName := getCurrentClinicName(c)
	role := getCurrentRole(c)

	fmt.Println("Dashboard Query Running")

	query := `
		SELECT
			COALESCE(SUM(COALESCE(cash_amount,0)),0),
			COALESCE(SUM(COALESCE(online_amount,0)),0),
			COALESCE(SUM(COALESCE(expense_amount,0)),0)
		FROM daily_collections`
	var args []interface{}
	if role != "superadmin" {
		query += " WHERE clinic_name=$1"
		args = append(args, clinicName)
	}

	err := db.QueryRow(query, args...).Scan(&d.CashTotal, &d.OnlineTotal, &d.ExpenseTotal)
	if err != nil {
		c.String(http.StatusInternalServerError, "Dashboard collection error: %v", err)
		return
	}

	d.FinalBalance = d.CashTotal + d.OnlineTotal - d.ExpenseTotal

	if role == "superadmin" {
		_ = db.QueryRow("SELECT COUNT(*) FROM attendance_intime WHERE role='Doctor' AND attendance_date=CURRENT_DATE").Scan(&d.DoctorPresent)
		_ = db.QueryRow("SELECT COUNT(*) FROM attendance_intime WHERE role='Sister' AND attendance_date=CURRENT_DATE").Scan(&d.SisterPresent)
	} else {
		_ = db.QueryRow("SELECT COUNT(*) FROM attendance_intime WHERE role='Doctor' AND attendance_date=CURRENT_DATE AND clinic_name=$1", clinicName).Scan(&d.DoctorPresent)
		_ = db.QueryRow("SELECT COUNT(*) FROM attendance_intime WHERE role='Sister' AND attendance_date=CURRENT_DATE AND clinic_name=$1", clinicName).Scan(&d.SisterPresent)
	}
	c.HTML(http.StatusOK, "admin_dashboard.html", d)
}

func attendanceInTimePage(c *gin.Context) {
	c.HTML(http.StatusOK, "attendance_intime.html", gin.H{
		"CurrentDate": time.Now().Format("2006-01-02"),
		"CurrentTime": time.Now().Format("15:04"),
	})
}

func attendanceOutTimePage(c *gin.Context) {
	c.HTML(http.StatusOK, "attendance_outtime.html", gin.H{
		"CurrentDate": time.Now().Format("2006-01-02"),
		"CurrentTime": time.Now().Format("15:04"),
	})
}

func saveInTime(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("staff_name"))
	role := strings.TrimSpace(c.PostForm("role"))
	date := strings.TrimSpace(c.PostForm("attendance_date"))
	inTime := strings.TrimSpace(c.PostForm("in_time"))
	clinicName := getCurrentClinicName(c)
	if clinicName == "" {
		clinicName = strings.TrimSpace(c.PostForm("clinic_name"))
	}
	if clinicName == "" {
		clinicName = "Kalwa"
	}

	currentRole := getCurrentRole(c)
	if currentRole == "doctor" || currentRole == "sister" {
		role = strings.Title(currentRole)
		name = getCurrentUsername(c)
	}

	if name == "" || role == "" || date == "" || inTime == "" {
		c.String(http.StatusBadRequest, "Staff name, role, date and in time are required")
		return
	}

	var existingID int
	err := db.QueryRow(`
		SELECT id
		FROM attendance_intime
		WHERE LOWER(staff_name)=LOWER($1)
		AND attendance_date=$2
		AND clinic_name=$3
		ORDER BY id DESC
		LIMIT 1
	`, name, date, clinicName).Scan(&existingID)

	if err == nil {
		c.String(http.StatusBadRequest, "Already checked in for today. Please check out or delete old entry first.")
		return
	}
	if err != sql.ErrNoRows {
		c.String(http.StatusInternalServerError, "Check In Search Error: %v", err)
		return
	}

	selfiePath := saveSelfieImage(c.PostForm("selfie_data"))

	_, err = db.Exec(`
		INSERT INTO attendance_intime
		(staff_name, role, attendance_date, in_time, clinic_name, selfie_path)
		VALUES($1,$2,$3,$4,$5,$6)
	`, name, role, date, inTime, clinicName, selfiePath)

	if err != nil {
		c.String(http.StatusInternalServerError, "Check In Error: %v", err)
		return
	}

	if currentRole == "doctor" || currentRole == "sister" {
		c.Redirect(http.StatusSeeOther, "/my-report")
		return
	}
	c.Redirect(http.StatusSeeOther, "/attendance-report")
}

func saveOutTime(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("staff_name"))
	role := strings.TrimSpace(c.PostForm("role"))
	date := strings.TrimSpace(c.PostForm("attendance_date"))
	outTime := strings.TrimSpace(c.PostForm("out_time"))
	clinicName := getCurrentClinicName(c)
	if clinicName == "" {
		clinicName = strings.TrimSpace(c.PostForm("clinic_name"))
	}
	if clinicName == "" {
		clinicName = "Kalwa"
	}

	currentRole := getCurrentRole(c)
	if currentRole == "doctor" || currentRole == "sister" {
		role = strings.Title(currentRole)
		name = getCurrentUsername(c)
	}
	if role == "" {
		role = "Doctor"
	}

	if name == "" || date == "" || outTime == "" {
		c.String(http.StatusBadRequest, "Staff name, date and out time are required")
		return
	}

	var inID int
	var inTime string
	err := db.QueryRow(`
		SELECT id, TO_CHAR(in_time, 'HH24:MI')
		FROM attendance_intime
		WHERE LOWER(staff_name)=LOWER($1)
		AND attendance_date=$2
		AND clinic_name=$3
		ORDER BY id DESC
		LIMIT 1
	`, name, date, clinicName).Scan(&inID, &inTime)

	if err == sql.ErrNoRows {
		c.String(http.StatusBadRequest, "Check In record not found. Please check in first.")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "Check Out Search Error: %v", err)
		return
	}

	var existingOutID int
	err = db.QueryRow(`
		SELECT id
		FROM attendance_outtime
		WHERE LOWER(staff_name)=LOWER($1)
		AND attendance_date=$2
		AND clinic_name=$3
		ORDER BY id DESC
		LIMIT 1
	`, name, date, clinicName).Scan(&existingOutID)

	if err == nil {
		c.String(http.StatusBadRequest, "Already checked out for today.")
		return
	}
	if err != sql.ErrNoRows {
		c.String(http.StatusInternalServerError, "Check Out Duplicate Search Error: %v", err)
		return
	}

	selfiePath := saveSelfieImage(c.PostForm("selfie_data"))

	_, err = db.Exec(`
		INSERT INTO attendance_outtime
		(staff_name, role, attendance_date, out_time, clinic_name, selfie_path)
		VALUES($1,$2,$3,$4,$5,$6)
	`, name, role, date, outTime, clinicName, selfiePath)

	if err != nil {
		c.String(http.StatusInternalServerError, "Check Out Error: %v", err)
		return
	}

	_ = calculateMinutes(inTime, outTime)

	if currentRole == "doctor" || currentRole == "sister" {
		c.Redirect(http.StatusSeeOther, "/my-report")
		return
	}
	c.Redirect(http.StatusSeeOther, "/attendance-report")
}

func saveAttendance(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("staff_name"))
	role := strings.TrimSpace(c.PostForm("role"))
	date := strings.TrimSpace(c.PostForm("attendance_date"))
	inTime := strings.TrimSpace(c.PostForm("in_time"))
	outTime := strings.TrimSpace(c.PostForm("out_time"))
	clinicName := getCurrentClinicName(c)
	if clinicName == "" {
		clinicName = strings.TrimSpace(c.PostForm("clinic_name"))
	}
	if clinicName == "" {
		clinicName = "Kalwa"
	}

	currentRole := getCurrentRole(c)
	if currentRole == "doctor" || currentRole == "sister" {
		role = strings.Title(currentRole)
		if name == "" {
			name = getCurrentUsername(c)
		}
	}

	selfiePath := saveSelfieImage(c.PostForm("selfie_data"))

	if inTime != "" {
		_, err := db.Exec(`
			INSERT INTO attendance_intime
			(staff_name, role, attendance_date, in_time, clinic_name, selfie_path)
			VALUES($1,$2,$3,$4,$5,$6)
		`, name, role, date, inTime, clinicName, selfiePath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Attendance in time save error: %v", err)
			return
		}
	}

	if outTime != "" {
		_, err := db.Exec(`
			INSERT INTO attendance_outtime
			(staff_name, role, attendance_date, out_time, clinic_name, selfie_path)
			VALUES($1,$2,$3,$4,$5,$6)
		`, name, role, date, outTime, clinicName, selfiePath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Attendance out time save error: %v", err)
			return
		}
	}

	if currentRole == "doctor" || currentRole == "sister" {
		c.Redirect(http.StatusFound, "/my-report")
		return
	}
	c.Redirect(http.StatusFound, "/attendance-report")
}

func saveSelfieImage(selfieData string) string {
	selfieData = strings.TrimSpace(selfieData)
	if selfieData == "" {
		return ""
	}

	selfieData = strings.Replace(selfieData, "data:image/png;base64,", "", 1)
	selfieData = strings.Replace(selfieData, "data:image/jpeg;base64,", "", 1)

	imageBytes, err := base64.StdEncoding.DecodeString(selfieData)
	if err != nil {
		log.Println("selfie decode error:", err)
		return ""
	}

	if err := os.MkdirAll("static/uploads", os.ModePerm); err != nil {
		log.Println("upload folder error:", err)
		return ""
	}

	fileName := time.Now().Format("20060102150405") + "_selfie.png"
	filePath := "static/uploads/" + fileName

	if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
		log.Println("selfie save error:", err)
		return ""
	}

	return "/" + filePath
}

func showUpdateAttendancePage(c *gin.Context) {
	id := c.Param("id")
	var a Attendance
	var d time.Time
	var mins int

	err := db.QueryRow(`
		SELECT
			i.id,
			i.staff_name,
			i.role,
			i.attendance_date,
			COALESCE(TO_CHAR(i.in_time, 'HH24:MI'), ''),
			COALESCE(TO_CHAR(o.out_time, 'HH24:MI'), ''),
			COALESCE(CAST(EXTRACT(EPOCH FROM (o.out_time - i.in_time)) / 60 AS INTEGER), 0),
			'Present',
			''
		FROM attendance_intime i
		LEFT JOIN attendance_outtime o
		ON LOWER(i.staff_name)=LOWER(o.staff_name)
		AND i.attendance_date=o.attendance_date
		AND i.clinic_name=o.clinic_name
		WHERE i.id=$1
	`, id).Scan(&a.ID, &a.StaffName, &a.Role, &d, &a.InTime, &a.OutTime, &mins, &a.Status, &a.Remark)

	if err != nil {
		c.String(http.StatusInternalServerError, "Attendance fetch error: %v", err)
		return
	}

	a.AttendanceDate = d.Format("2006-01-02")
	a.Hours = minutesToHours(mins)
	c.HTML(http.StatusOK, "attendance_update.html", a)
}

func updateAttendance(c *gin.Context) {
	id := c.Param("id")
	name := strings.TrimSpace(c.PostForm("staff_name"))
	role := strings.TrimSpace(c.PostForm("role"))
	date := strings.TrimSpace(c.PostForm("attendance_date"))
	inTime := strings.TrimSpace(c.PostForm("in_time"))
	outTime := strings.TrimSpace(c.PostForm("out_time"))
	clinicName := getCurrentClinicName(c)
	if clinicName == "" {
		clinicName = strings.TrimSpace(c.PostForm("clinic_name"))
	}
	if clinicName == "" {
		clinicName = "Kalwa"
	}

	currentRole := getCurrentRole(c)
	if currentRole == "doctor" || currentRole == "sister" {
		role = strings.Title(currentRole)
		if name == "" {
			name = getCurrentUsername(c)
		}
	}

	_, err := db.Exec(`
		UPDATE attendance_intime
		SET staff_name=$1, role=$2, attendance_date=$3, in_time=$4, clinic_name=$5
		WHERE id=$6
	`, name, role, date, inTime, clinicName, id)
	if err != nil {
		c.String(http.StatusInternalServerError, "In time update error: %v", err)
		return
	}

	_, _ = db.Exec(`
		DELETE FROM attendance_outtime
		WHERE LOWER(staff_name)=LOWER($1)
		AND attendance_date=$2
		AND clinic_name=$3
	`, name, date, clinicName)

	if outTime != "" {
		_, err = db.Exec(`
			INSERT INTO attendance_outtime
			(staff_name, role, attendance_date, out_time, clinic_name)
			VALUES($1,$2,$3,$4,$5)
		`, name, role, date, outTime, clinicName)
		if err != nil {
			c.String(http.StatusInternalServerError, "Out time update error: %v", err)
			return
		}
	}

	c.Redirect(http.StatusSeeOther, "/attendance-report")
}

func deleteAttendance(c *gin.Context) {
	id := c.Param("id")

	var staffName, clinicName string
	var attendanceDate time.Time
	err := db.QueryRow(`
		SELECT staff_name, clinic_name, attendance_date
		FROM attendance_intime
		WHERE id=$1
	`, id).Scan(&staffName, &clinicName, &attendanceDate)

	if err != nil {
		c.String(http.StatusInternalServerError, "Attendance fetch before delete error: %v", err)
		return
	}

	_, err = db.Exec(`
		DELETE FROM attendance_outtime
		WHERE LOWER(staff_name)=LOWER($1)
		AND clinic_name=$2
		AND attendance_date=$3
	`, staffName, clinicName, attendanceDate)
	if err != nil {
		c.String(http.StatusInternalServerError, "Out time delete error: %v", err)
		return
	}

	_, err = db.Exec("DELETE FROM attendance_intime WHERE id=$1", id)
	if err != nil {
		c.String(http.StatusInternalServerError, "In time delete error: %v", err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/attendance-report")
}

func calculateMinutes(inTime, outTime string) int {
	if inTime == "" || outTime == "" {
		return 0
	}
	start, err1 := time.Parse("15:04", inTime)
	end, err2 := time.Parse("15:04", outTime)
	if err1 != nil || err2 != nil {
		return 0
	}
	if end.Before(start) {
		end = end.Add(24 * time.Hour)
	}
	return int(end.Sub(start).Minutes())
}

func minutesToHours(mins int) string {
	return fmt.Sprintf("%dh %dm", mins/60, mins%60)
}

func attendanceReportPage(c *gin.Context) {
	clinicName := getCurrentClinicName(c)
	roleLogin := getCurrentRole(c)

	query := `
		SELECT
			i.id,
			i.staff_name,
			i.role,
			i.attendance_date,
			COALESCE(TO_CHAR(i.in_time, 'HH24:MI'), ''),
			COALESCE(TO_CHAR(o.out_time, 'HH24:MI'), ''),
			COALESCE(CAST(EXTRACT(EPOCH FROM (o.out_time - i.in_time)) / 60 AS INTEGER), 0),
			'Present' AS status,
			'' AS remark,
			COALESCE(i.clinic_name, ''),
			COALESCE(i.selfie_path, '')
		FROM attendance_intime i
		LEFT JOIN attendance_outtime o
		ON LOWER(i.staff_name)=LOWER(o.staff_name)
		AND i.attendance_date=o.attendance_date
		AND i.clinic_name=o.clinic_name
	`
	var args []interface{}
	if roleLogin != "superadmin" {
		query += " WHERE i.clinic_name=$1"
		args = append(args, clinicName)
	}
	query += " ORDER BY i.attendance_date DESC, i.id DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.String(http.StatusInternalServerError, "Attendance report query error: %v", err)
		return
	}
	defer rows.Close()

	var attendance []Attendance
	var doctorPresent, doctorAbsent, doctorHalfDay int
	var sisterPresent, sisterAbsent, sisterHalfDay int

	for rows.Next() {
		var a Attendance
		var mins int
		var d time.Time

		err := rows.Scan(&a.ID, &a.StaffName, &a.Role, &d, &a.InTime, &a.OutTime, &mins, &a.Status, &a.Remark, &a.ClinicName, &a.SelfiePath)
		if err != nil {
			c.String(http.StatusInternalServerError, "Attendance scan error: %v", err)
			return
		}

		a.AttendanceDate = d.Format("2006-01-02")
		a.InTime = strings.TrimSpace(a.InTime)
		a.OutTime = strings.TrimSpace(a.OutTime)
		a.Hours = minutesToHours(mins)

		if a.Role == "Doctor" {
			doctorPresent++
		}
		if a.Role == "Sister" {
			sisterPresent++
		}

		attendance = append(attendance, a)
	}

	c.HTML(http.StatusOK, "attendance_report.html", gin.H{
		"Attendance":    attendance,
		"DoctorPresent": doctorPresent, "DoctorAbsent": doctorAbsent, "DoctorHalfDay": doctorHalfDay,
		"SisterPresent": sisterPresent, "SisterAbsent": sisterAbsent, "SisterHalfDay": sisterHalfDay,
		"ClinicName": clinicName,
	})
}

func collectionPage(c *gin.Context) {
	c.HTML(http.StatusOK, "collection.html", nil)
}

func saveCollection(c *gin.Context) {
	date := c.PostForm("collection_date")
	cash, _ := strconv.ParseFloat(c.PostForm("cash_amount"), 64)
	online, _ := strconv.ParseFloat(c.PostForm("online_amount"), 64)
	expense, _ := strconv.ParseFloat(c.PostForm("expense_amount"), 64)
	reason := c.PostForm("expense_reason")
	remark := c.PostForm("remark")
	enteredBy := c.PostForm("entered_by")
	clinicName := getCurrentClinicName(c)

	_, err := db.Exec(`
		INSERT INTO daily_collections
		(collection_date, cash_amount, online_amount, expense_amount, expense_reason, remark, entered_by, clinic_name)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
	`, date, cash, online, expense, reason, remark, enteredBy, clinicName)
	if err != nil {
		c.String(http.StatusInternalServerError, "Collection save error: %v", err)
		return
	}

	c.Redirect(http.StatusFound, "/collection-report")
}

func collectionReportPage(c *gin.Context) {
	fromDate := c.Query("from_date")
	toDate := c.Query("to_date")
	clinicName := getCurrentClinicName(c)
	role := getCurrentRole(c)

	query := `
		SELECT id, collection_date, COALESCE(cash_amount,0), COALESCE(online_amount,0),
		COALESCE(expense_amount,0), COALESCE(expense_reason,''), COALESCE(remark,''), COALESCE(entered_by,''), COALESCE(clinic_name,'')
		FROM daily_collections
	`
	var args []interface{}
	var where []string
	if role != "superadmin" {
		args = append(args, clinicName)
		where = append(where, fmt.Sprintf("clinic_name=$%d", len(args)))
	}
	if fromDate != "" && toDate != "" {
		args = append(args, fromDate)
		where = append(where, fmt.Sprintf("collection_date >= $%d", len(args)))
		args = append(args, toDate)
		where = append(where, fmt.Sprintf("collection_date <= $%d", len(args)))
	} else if fromDate != "" {
		args = append(args, fromDate)
		where = append(where, fmt.Sprintf("collection_date >= $%d", len(args)))
	} else if toDate != "" {
		args = append(args, toDate)
		where = append(where, fmt.Sprintf("collection_date <= $%d", len(args)))
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY collection_date ASC, id ASC"

	rows, err := db.Query(query, args...)
	if err != nil {
		c.String(http.StatusInternalServerError, "Collection report query error: %v", err)
		return
	}
	defer rows.Close()

	var collections []Collection
	var totalAmount, totalExpense, totalCash, totalOnline, finalBalance float64

	for rows.Next() {
		var x Collection
		var d time.Time
		err := rows.Scan(&x.ID, &d, &x.Cash, &x.Online, &x.Expense, &x.Reason, &x.Remark, &x.EnteredBy, &x.ClinicName)
		if err != nil {
			c.String(http.StatusInternalServerError, "Collection scan error: %v", err)
			return
		}

		x.Date = d.Format("2006-01-02")
		x.Total = x.Cash + x.Online
		x.FinalBalance = x.Total - x.Expense
		x.CashInHand = x.Cash - x.Expense

		totalCash += x.Cash
		totalOnline += x.Online
		totalExpense += x.Expense
		totalAmount += x.Total
		finalBalance += x.FinalBalance

		collections = append(collections, x)
	}

	c.HTML(http.StatusOK, "collection_report.html", gin.H{
		"Collections": collections, "TotalAmount": totalAmount, "TotalExpense": totalExpense,
		"TotalCash": totalCash, "TotalOnline": totalOnline, "FinalBalance": finalBalance,
		"TotalCashInHand": totalCash - totalExpense,
		"FromDate":        fromDate, "ToDate": toDate, "ClinicName": clinicName,
	})
}

func collectionOnlineReportPage(c *gin.Context) {
	rows, err := db.Query(`
		SELECT id, collection_date, COALESCE(online_amount,0), COALESCE(expense_amount,0), COALESCE(remark,'')
		FROM daily_collections_online ORDER BY collection_date ASC, id ASC
	`)
	if err != nil {
		c.String(http.StatusInternalServerError, "Collection online report error: %v", err)
		return
	}
	defer rows.Close()

	var collections []Collection
	var totalOnline, totalExpense, totalCashInHand float64

	for rows.Next() {
		var x Collection
		var d time.Time
		err := rows.Scan(&x.ID, &d, &x.Online, &x.Expense, &x.Remark)
		if err != nil {
			c.String(http.StatusInternalServerError, "Collection scan error: %v", err)
			return
		}
		x.Date = d.Format("2006-01-02")
		x.CashInHand = x.Online - x.Expense
		totalOnline += x.Online
		totalExpense += x.Expense
		totalCashInHand += x.CashInHand
		collections = append(collections, x)
	}

	c.HTML(http.StatusOK, "collection_report.html", gin.H{
		"Collections": collections, "TotalOnline": totalOnline,
		"TotalExpense": totalExpense, "TotalCashInHand": totalCashInHand,
	})
}

func showUpdateCollectionPage(c *gin.Context) {
	id := c.Param("id")
	var x Collection
	var d time.Time

	err := db.QueryRow(`
		SELECT id, collection_date, COALESCE(cash_amount,0), COALESCE(online_amount,0), COALESCE(expense_amount,0),
		COALESCE(expense_reason,''), COALESCE(remark,''), COALESCE(entered_by,'')
		FROM daily_collections WHERE id=$1
	`, id).Scan(&x.ID, &d, &x.Cash, &x.Online, &x.Expense, &x.Reason, &x.Remark, &x.EnteredBy)
	if err != nil {
		c.String(http.StatusInternalServerError, "Collection fetch error: %v", err)
		return
	}

	x.Date = d.Format("2006-01-02")
	x.Total = x.Cash + x.Online
	x.FinalBalance = x.Total - x.Expense
	c.HTML(http.StatusOK, "collection_update.html", x)
}

func updateCollection(c *gin.Context) {
	id := c.Param("id")
	date := c.PostForm("collection_date")
	cash, _ := strconv.ParseFloat(c.PostForm("cash_amount"), 64)
	online, _ := strconv.ParseFloat(c.PostForm("online_amount"), 64)
	expense, _ := strconv.ParseFloat(c.PostForm("expense_amount"), 64)
	reason := c.PostForm("expense_reason")
	remark := c.PostForm("remark")
	enteredBy := c.PostForm("entered_by")

	_, err := db.Exec(`
		UPDATE daily_collections SET collection_date=$1, cash_amount=$2, online_amount=$3,
		expense_amount=$4, expense_reason=$5, remark=$6, entered_by=$7 WHERE id=$8
	`, date, cash, online, expense, reason, remark, enteredBy, id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Collection update error: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/collection-report")
}

func deleteCollection(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM daily_collections WHERE id=$1", id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Collection delete error: %v", err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/collection-report")
}

func submitInvoice(c *gin.Context) {
	clinicName := getCurrentClinicName(c)
	customerName := c.PostForm("customer_name")
	mobile := c.PostForm("mobile")
	address := c.PostForm("address")
	gender := c.PostForm("gender")
	ageStr := c.PostForm("age")
	age, err := strconv.Atoi(ageStr)
	if err != nil || age < 1 || age > 120 {
		c.String(http.StatusBadRequest, "Invalid age. Must be between 1 and 120.")
		return
	}

	services := c.PostFormArray("service[]")
	prices := c.PostFormArray("price[]")
	discounts := c.PostFormArray("discount[]")
	totals := c.PostFormArray("total[]")

	var tests []TestItem
	var totalAmount float64

	for i := range services {
		testName := strings.Split(services[i], "|")[0]
		price, _ := strconv.ParseFloat(prices[i], 64)
		discount, _ := strconv.ParseFloat(discounts[i], 64)
		total, _ := strconv.ParseFloat(totals[i], 64)
		totalAmount += total

		tests = append(tests, TestItem{
			TestName: testName, Price: price, Discount: discount,
			TotalAmount: total, TotalAmountWords: convertToWords(int(total)),
		})
	}

	var invoiceID int
	err = db.QueryRow(`
		INSERT INTO invoices12 (customer_name, mobile, address, age, gender, clinic_name)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id
	`, customerName, mobile, address, age, gender, clinicName).Scan(&invoiceID)
	if err != nil {
		c.String(http.StatusInternalServerError, "Invoice Insert Error: %v", err)
		return
	}

	for _, test := range tests {
		_, err := db.Exec(`INSERT INTO tests12 (invoice_id, test_name, price, discount, total_amount, total_amount_words, clinic_name) VALUES ($1,$2,$3,$4,$5,$6,$7)`, invoiceID, test.TestName, test.Price, test.Discount, test.TotalAmount, test.TotalAmountWords, clinicName)
		if err != nil {
			c.String(http.StatusInternalServerError, "Test Insert Error: %v", err)
			return
		}
	}

	invoice := Invoice{
		InvoiceNo: invoiceID, ClinicName: clinicName, InvoiceDatetime: time.Now().Format("02-Jan-2006 03:04 PM"),
		CustomerName: customerName, Gender: gender, Mobile: mobile, Age: age, Address: address,
		Tests: tests, TotalAmount: totalAmount, TotalAmountWords: convertToWords(int(totalAmount)),
	}

	c.HTML(http.StatusOK, "invoice.html", invoice)
}

func displayAllInvoiceDataHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		clinicName := getCurrentClinicName(c)
		role := getCurrentRole(c)
		query := `
			SELECT 
				invoices12.id,
				COALESCE(invoices12.customer_name,''),
				COALESCE(invoices12.age,0),
				COALESCE(invoices12.mobile,''),
				invoices12.created_at,
				COALESCE(tests12.id,0),
				COALESCE(tests12.test_name,''),
				COALESCE(tests12.price,0),
				COALESCE(tests12.discount,0),
				COALESCE(tests12.total_amount,0),
				COALESCE(tests12.total_amount_words,''),
				COALESCE(invoices12.clinic_name,'')
			FROM invoices12
			LEFT JOIN tests12 ON invoices12.id = tests12.invoice_id`
		var args []interface{}
		if role != "superadmin" {
			query += " WHERE invoices12.clinic_name=$1"
			args = append(args, clinicName)
		}
		query += " ORDER BY invoices12.id, tests12.id"

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println("Query error:", err)
			c.String(http.StatusInternalServerError, "Database error")
			return
		}
		defer rows.Close()

		var allData []FullInvoiceData
		srNo := 1

		for rows.Next() {
			var data FullInvoiceData

			err := rows.Scan(
				&data.InvoiceID,
				&data.CustomerName,
				&data.Age,
				&data.MobileNo,
				&data.CreatedAt,
				&data.TestID,
				&data.TestName,
				&data.Price,
				&data.Discount,
				&data.TotalAmount,
				&data.TotalAmountWords,
				&data.ClinicName,
			)
			if err != nil {
				log.Println("Scan error:", err)
				continue
			}

			data.SrNo = srNo
			srNo++
			allData = append(allData, data)
		}

		var grandTotal float64
		if role == "superadmin" {
			_ = db.QueryRow(`SELECT COALESCE(SUM(total_amount),0) FROM tests12`).Scan(&grandTotal)
		} else {
			_ = db.QueryRow(`SELECT COALESCE(SUM(total_amount),0) FROM tests12 WHERE clinic_name=$1`, clinicName).Scan(&grandTotal)
		}

		c.HTML(http.StatusOK, "print.html", PageData{
			Data:       allData,
			GrandTotal: grandTotal,
		})
	}
}

func filterInvoicesHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := c.Query("start")
		end := c.Query("end")
		clinicName := getCurrentClinicName(c)
		role := getCurrentRole(c)
		if start == "" || end == "" {
			c.String(http.StatusBadRequest, "Start and end date are required.")
			return
		}

		startTime := start + " 00:00:00"
		endTime := end + " 23:59:59"

		query := `
			SELECT 
				invoices12.id,
				COALESCE(invoices12.customer_name,''),
				COALESCE(invoices12.age,0),
				COALESCE(invoices12.mobile,''),
				invoices12.created_at,
				COALESCE(tests12.id,0),
				COALESCE(tests12.test_name,''),
				COALESCE(tests12.price,0),
				COALESCE(tests12.discount,0),
				COALESCE(tests12.total_amount,0),
				COALESCE(tests12.total_amount_words,''),
				COALESCE(invoices12.clinic_name,'')
			FROM invoices12
			LEFT JOIN tests12 ON invoices12.id = tests12.invoice_id
			WHERE invoices12.created_at BETWEEN $1 AND $2`
		args := []interface{}{startTime, endTime}
		if role != "superadmin" {
			query += " AND invoices12.clinic_name=$3"
			args = append(args, clinicName)
		}
		query += " ORDER BY invoices12.id, tests12.id"

		rows, err := db.Query(query, args...)
		if err != nil {
			log.Println("Date filter query error:", err)
			c.String(http.StatusInternalServerError, "Database Error")
			return
		}
		defer rows.Close()

		var allData []FullInvoiceData
		srNo := 1

		for rows.Next() {
			var data FullInvoiceData

			err := rows.Scan(
				&data.InvoiceID,
				&data.CustomerName,
				&data.Age,
				&data.MobileNo,
				&data.CreatedAt,
				&data.TestID,
				&data.TestName,
				&data.Price,
				&data.Discount,
				&data.TotalAmount,
				&data.TotalAmountWords,
				&data.ClinicName,
			)
			if err != nil {
				log.Println("Scan error:", err)
				continue
			}

			data.SrNo = srNo
			srNo++
			allData = append(allData, data)
		}

		var grandTotal float64
		if role == "superadmin" {
			_ = db.QueryRow(`
				SELECT COALESCE(SUM(tests12.total_amount),0)
				FROM invoices12
				LEFT JOIN tests12 ON invoices12.id = tests12.invoice_id
				WHERE invoices12.created_at BETWEEN $1 AND $2
			`, startTime, endTime).Scan(&grandTotal)
		} else {
			_ = db.QueryRow(`
				SELECT COALESCE(SUM(tests12.total_amount),0)
				FROM invoices12
				LEFT JOIN tests12 ON invoices12.id = tests12.invoice_id
				WHERE invoices12.created_at BETWEEN $1 AND $2 AND invoices12.clinic_name=$3
			`, startTime, endTime, clinicName).Scan(&grandTotal)
		}

		c.HTML(http.StatusOK, "print.html", PageData{
			Data:       allData,
			GrandTotal: grandTotal,
		})
	}
}

func getInvoiceByID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.Param("id"))
		clinicName := getCurrentClinicName(c)
		role := getCurrentRole(c)

		invoiceID, err := strconv.Atoi(id)
		if err != nil || invoiceID <= 0 {
			c.String(http.StatusBadRequest, "Invalid Invoice ID")
			return
		}

		var invoice Invoice
		var createdAt time.Time
		query := `
			SELECT
				id,
				COALESCE(customer_name,''),
				COALESCE(mobile,''),
				COALESCE(address,''),
				COALESCE(age,0),
				COALESCE(gender,''),
				created_at,
				COALESCE(clinic_name,'')
			FROM invoices12
			WHERE id=$1`
		args := []interface{}{invoiceID}
		if role != "superadmin" {
			query += " AND clinic_name=$2"
			args = append(args, clinicName)
		}

		err = db.QueryRow(query, args...).Scan(
			&invoice.InvoiceNo,
			&invoice.CustomerName,
			&invoice.Mobile,
			&invoice.Address,
			&invoice.Age,
			&invoice.Gender,
			&createdAt,
			&invoice.ClinicName,
		)

		if err == sql.ErrNoRows {
			c.String(http.StatusNotFound, "Invoice ID %d Not Found", invoiceID)
			return
		}
		if err != nil {
			log.Println("Invoice fetch error:", err)
			c.String(http.StatusInternalServerError, "Invoice fetch error: %v", err)
			return
		}

		invoice.InvoiceDatetime = createdAt.Format("02-Jan-2006 03:04 PM")

		rows, err := db.Query(`
			SELECT
				COALESCE(test_name,''),
				COALESCE(price,0),
				COALESCE(discount,0),
				COALESCE(total_amount,0),
				COALESCE(total_amount_words,'')
			FROM tests12
			WHERE invoice_id=$1
			ORDER BY id ASC
		`, invoiceID)
		if err != nil {
			log.Println("Invoice tests query error:", err)
			c.String(http.StatusInternalServerError, "Invoice tests query error: %v", err)
			return
		}
		defer rows.Close()

		var tests []TestItem
		var total float64
		for rows.Next() {
			var t TestItem
			err := rows.Scan(&t.TestName, &t.Price, &t.Discount, &t.TotalAmount, &t.TotalAmountWords)
			if err != nil {
				log.Println("Invoice test scan error:", err)
				continue
			}
			if t.TotalAmountWords == "" {
				t.TotalAmountWords = convertToWords(int(t.TotalAmount))
			}
			total += t.TotalAmount
			tests = append(tests, t)
		}

		invoice.Tests = tests
		invoice.TotalAmount = total
		invoice.TotalAmountWords = convertToWords(int(total))

		c.HTML(http.StatusOK, "invoice.html", invoice)
	}
}

func convertToWords(num int) string {
	ones := []string{"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine", "Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen", "Seventeen", "Eighteen", "Nineteen"}
	tens := []string{"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety"}

	if num == 0 {
		return "Zero Rupees Only"
	}
	if num > 99999 {
		return "Amount too large"
	}

	words := ""
	if num >= 1000 {
		words += ones[num/1000] + " Thousand "
		num %= 1000
	}
	if num >= 100 {
		words += ones[num/100] + " Hundred "
		num %= 100
	}
	if num >= 20 {
		words += tens[num/10] + " "
		num %= 10
	}
	if num > 0 {
		words += ones[num]
	}

	return strings.TrimSpace(words) + " Rupees Only"
}

func createTables() {
	// USERS TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			address TEXT,
			mobile VARCHAR(15),
			username VARCHAR(50) UNIQUE NOT NULL,
			password VARCHAR(100) NOT NULL,
			role VARCHAR(20) DEFAULT 'doctor',
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			selfie_path TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("users table error:", err)
	}
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'doctor'`)
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)

	// DEFAULT ADMIN USER
	if _, err := db.Exec(`
		INSERT INTO users(name,address,mobile,username,password,role,clinic_name) VALUES
		('Super Admin','All Clinics','9999999990','superadmin','super123','superadmin','All'),
		('Admin Kalwa','Kalwa','9999999991','admin_kalwa','admin123','admin','Kalwa'),
		('Doctor Kalwa','Kalwa','9999999992','doctor_kalwa','doctor123','doctor','Kalwa'),
		('Sister Kalwa','Kalwa','9999999993','sister_kalwa','sister123','sister','Kalwa'),
		('Admin Vashi','Vashi','9999999994','admin_vashi','admin123','admin','Vashi'),
		('Doctor Vashi','Vashi','9999999995','doctor_vashi','doctor123','doctor','Vashi'),
		('Sister Vashi','Vashi','9999999996','sister_vashi','sister123','sister','Vashi'),
		('Admin Byculla','Byculla','9999999997','admin_byculla','admin123','admin','Byculla'),
		('Doctor Byculla','Byculla','9999999998','doctor_byculla','doctor123','doctor','Byculla'),
		('Sister Byculla','Byculla','9999999999','sister_byculla','sister123','sister','Byculla')
		ON CONFLICT (username) DO NOTHING
	`); err != nil {
		log.Println("admin insert error:", err)
	}

	// staff_attendance table removed. Attendance report now uses attendance_intime + attendance_outtime.

	// OPTIONAL SEPARATE IN/OUT TIME LOG TABLES
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attendance_intime (
			id SERIAL PRIMARY KEY,
			staff_name VARCHAR(100) NOT NULL,
			role VARCHAR(50) NOT NULL,
			attendance_date DATE DEFAULT CURRENT_DATE,
			in_time TIME DEFAULT CURRENT_TIME,
			clinic_name VARCHAR(20) NOT NULL CHECK (clinic_name IN ('Kalwa','Vashi','Byculla')),
			selfie_path VARCHAR(500),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("attendance_intime table error:", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attendance_outtime (
			id SERIAL PRIMARY KEY,
			staff_name VARCHAR(100) NOT NULL,
			role VARCHAR(50) NOT NULL,
			attendance_date DATE DEFAULT CURRENT_DATE,
			out_time TIME DEFAULT CURRENT_TIME,
			clinic_name VARCHAR(20) NOT NULL CHECK (clinic_name IN ('Kalwa','Vashi','Byculla')),
			selfie_path VARCHAR(500),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("attendance_outtime table error:", err)
	}

	// DAILY COLLECTIONS TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS daily_collections (
			id SERIAL PRIMARY KEY,
			collection_date DATE,
			cash_amount NUMERIC(12,2) DEFAULT 0,
			online_amount NUMERIC(12,2) DEFAULT 0,
			expense_amount NUMERIC(12,2) DEFAULT 0,
			expense_reason TEXT,
			remark TEXT,
			entered_by VARCHAR(100),
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("daily_collections table error:", err)
	}

	// ADD MISSING COLUMNS IF OLD TABLE EXISTS
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS cash_amount NUMERIC(12,2) DEFAULT 0`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS online_amount NUMERIC(12,2) DEFAULT 0`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS expense_amount NUMERIC(12,2) DEFAULT 0`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS expense_reason TEXT`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS remark TEXT`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS entered_by VARCHAR(100)`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)

	// DAILY COLLECTIONS ONLINE TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS daily_collections_online (
			id SERIAL PRIMARY KEY,
			collection_date DATE NOT NULL,
			online_amount NUMERIC(12,2) DEFAULT 0,
			expense_amount NUMERIC(12,2) DEFAULT 0,
			remark TEXT,
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("daily_collections_online table error:", err)
	}

	// INVOICE MASTER TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS invoices12 (
			id SERIAL PRIMARY KEY,
			customer_name VARCHAR(150),
			mobile VARCHAR(20),
			address TEXT,
			age INT,
			gender VARCHAR(20),
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("invoices12 table error:", err)
	}

	// ADD MISSING COLUMNS IF OLD INVOICE TABLE EXISTS
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS mobile VARCHAR(20)`)
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS address TEXT`)
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS age INT`)
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS gender VARCHAR(20)`)
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)
	db.Exec(`ALTER TABLE tests12 ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)
	db.Exec(`ALTER TABLE daily_collections_online ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)

	// INVOICE DETAILS TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tests12 (
			id SERIAL PRIMARY KEY,
			invoice_id INT REFERENCES invoices12(id) ON DELETE CASCADE,
			test_name VARCHAR(200),
			price NUMERIC(12,2),
			discount NUMERIC(12,2),
			total_amount NUMERIC(12,2),
			total_amount_words TEXT,
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("tests12 table error:", err)
	}

	fmt.Println("Database tables checked/created successfully")
}

func collectionDeepakPage(c *gin.Context) {
	c.HTML(http.StatusOK, "collection_deepak.html", nil)
}
func saveCollectionDeepak(c *gin.Context) {
	date := c.PostForm("collection_date")

	online, _ := strconv.ParseFloat(c.PostForm("online_amount"), 64)
	expense, _ := strconv.ParseFloat(c.PostForm("expense_amount"), 64)
	remark := c.PostForm("remark")
	clinicName := getCurrentClinicName(c)

	_, err := db.Exec(`
		INSERT INTO daily_collections_online
		(collection_date, online_amount, expense_amount, remark, clinic_name)
		VALUES($1,$2,$3,$4,$5)
	`, date, online, expense, remark, clinicName)

	if err != nil {
		c.String(http.StatusInternalServerError, "Collection save error: %v", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/collection-report")
}
func deleteCollectionDeepak(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM daily_collections_online WHERE id=$1", id)
	if err != nil {
		c.String(http.StatusInternalServerError, "Online collection delete error: %v", err)
		return
	}
	c.Redirect(http.StatusSeeOther, "/collection-report-deepak")
}

func collectionReportDeepakPage(c *gin.Context) {
	clinicName := getCurrentClinicName(c)
	role := getCurrentRole(c)
	query := `
		SELECT 
			id,
			collection_date,
			COALESCE(online_amount,0),
			COALESCE(expense_amount,0),
			COALESCE(remark,''),
			COALESCE(clinic_name,'')
		FROM daily_collections_online`
	var args []interface{}
	if role != "superadmin" {
		query += " WHERE clinic_name=$1"
		args = append(args, clinicName)
	}
	query += " ORDER BY collection_date ASC, id ASC"

	rows, err := db.Query(query, args...)

	if err != nil {
		c.String(http.StatusInternalServerError, "Collection report error: %v", err)
		return
	}
	defer rows.Close()

	var collections []Collection
	var totalOnline float64
	var totalExpense float64
	var totalCashInHand float64

	for rows.Next() {
		var x Collection
		var d time.Time

		err := rows.Scan(
			&x.ID,
			&d,
			&x.Online,
			&x.Expense,
			&x.Remark,
			&x.ClinicName,
		)
		if err != nil {
			c.String(http.StatusInternalServerError, "Collection scan error: %v", err)
			return
		}

		x.Date = d.Format("2006-01-02")
		x.CashInHand = x.Online - x.Expense

		totalOnline += x.Online
		totalExpense += x.Expense
		totalCashInHand += x.CashInHand

		collections = append(collections, x)
	}

	c.HTML(http.StatusOK, "collection_report_deepak.html", gin.H{
		"Collections":     collections,
		"TotalOnline":     totalOnline,
		"TotalExpense":    totalExpense,
		"TotalCashInHand": totalCashInHand,
		"ClinicName":      clinicName,
	})
}

func dailyReportPage(c *gin.Context) {
	searchDate := c.Query("date")
	clinicName := getCurrentClinicName(c)
	role := getCurrentRole(c)

	if searchDate == "" {
		searchDate = time.Now().Format("2006-01-02")
	}

	query := `
		SELECT
			COALESCE(d.clinic_name,''),
			d.collection_date::text,

			COALESCE((SELECT staff_name FROM attendance_intime
			 WHERE role ILIKE 'Doctor' AND attendance_date = d.collection_date AND (clinic_name=d.clinic_name OR $3='superadmin') LIMIT 1), ''),

			COALESCE((SELECT staff_name FROM attendance_intime
			 WHERE role ILIKE 'Sister' AND attendance_date = d.collection_date AND (clinic_name=d.clinic_name OR $3='superadmin') LIMIT 1), ''),

			COALESCE((SELECT in_time::text FROM attendance_intime
			 WHERE role ILIKE 'Doctor' AND attendance_date = d.collection_date AND (clinic_name=d.clinic_name OR $3='superadmin') LIMIT 1), ''),

			COALESCE((SELECT out_time::text FROM attendance_outtime
			 WHERE role ILIKE 'Doctor' AND attendance_date = d.collection_date AND (clinic_name=d.clinic_name OR $3='superadmin') LIMIT 1), ''),

			COALESCE((SELECT in_time::text FROM attendance_intime
			 WHERE role ILIKE 'Sister' AND attendance_date = d.collection_date AND (clinic_name=d.clinic_name OR $3='superadmin') LIMIT 1), ''),

			COALESCE((SELECT out_time::text FROM attendance_outtime
			 WHERE role ILIKE 'Sister' AND attendance_date = d.collection_date AND (clinic_name=d.clinic_name OR $3='superadmin') LIMIT 1), ''),

			COALESCE((SELECT COUNT(*) FROM invoices12
			 WHERE DATE(created_at) = d.collection_date AND (clinic_name=d.clinic_name OR $3='superadmin')), 0),

			(d.cash_amount + d.online_amount),

			d.cash_amount,
			d.online_amount,
			d.expense_amount,
			(d.cash_amount - d.expense_amount)

		FROM daily_collections d
		WHERE d.collection_date = $1`
	args := []interface{}{searchDate, clinicName, role}
	if role != "superadmin" {
		query += " AND d.clinic_name=$2"
	}
	query += " ORDER BY d.collection_date DESC"

	rows, err := db.Query(query, args...)

	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var reports []DailyReport

	for rows.Next() {
		var report DailyReport

		err := rows.Scan(
			&report.ClinicName,
			&report.ReportDate,
			&report.DoctorName,
			&report.SisterName,
			&report.DrInTime,
			&report.DrOutTime,
			&report.SisterInTime,
			&report.SisterOutTime,
			&report.TotalPatient,
			&report.TotalBilling,
			&report.CashBilling,
			&report.OnlineBilling,
			&report.Expense,
			&report.CashInHand,
		)

		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}

		reports = append(reports, report)
	}

	c.HTML(http.StatusOK, "daily_report.html", gin.H{
		"Reports":    reports,
		"SearchDate": searchDate,
		"ClinicName": clinicName,
	})
}
