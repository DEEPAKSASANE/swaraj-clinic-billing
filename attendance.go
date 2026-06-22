package main

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func attendanceInTimePage(c *gin.Context) {
	c.HTML(http.StatusOK, "attendance_intime.html", gin.H{
		"CurrentDate": time.Now().Format("2006-01-02"),
		"CurrentTime": time.Now().Format("15:04"),
		"ClinicName":  getDisplayClinicName(c),
	})
}
func attendanceOutTimePage(c *gin.Context) {
	c.HTML(http.StatusOK, "attendance_outtime.html", gin.H{
		"CurrentDate": time.Now().Format("2006-01-02"),
		"CurrentTime": time.Now().Format("15:04"),
		"ClinicName":  getDisplayClinicName(c),
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
		name = getCurrentStaffName(c)
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
		name = getCurrentStaffName(c)
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
			name = getCurrentStaffName(c)
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
	a.ClinicName = getDisplayClinicName(c)
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
			name = getCurrentStaffName(c)
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
			COALESCE(i.selfie_path, '') AS in_image,
			COALESCE(o.selfie_path, '') AS out_image
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

		err := rows.Scan(&a.ID, &a.StaffName, &a.Role, &d, &a.InTime, &a.OutTime, &mins, &a.Status, &a.Remark, &a.ClinicName, &a.InImage, &a.OutImage)
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
		"ClinicName":  getDisplayClinicName(c),
		"ClinicTitle": getDisplayClinicName(c),
	})
}
