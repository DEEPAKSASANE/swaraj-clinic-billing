package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"time"
)

func staffDashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "staff_dashboard.html", gin.H{"ClinicName": getDisplayClinicName(c), "ClinicTitle": getDisplayClinicName(c)})
}
func voiceReportPage(c *gin.Context) {
	c.HTML(http.StatusOK, "voice_report.html", gin.H{"ClinicName": getDisplayClinicName(c), "ClinicTitle": getDisplayClinicName(c)})
}
func myReportPage(c *gin.Context) {
	role := getCurrentRole(c)
	username := getCurrentUsername(c)
	staffName := getCurrentStaffName(c)
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
		AND (LOWER(i.role)=LOWER($1) OR LOWER(i.staff_name)=LOWER($2) OR LOWER(i.staff_name)=LOWER($4))
		ORDER BY i.attendance_date DESC, i.id DESC
	`

	rows, err := db.Query(query, role, username, clinicName, staffName)
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

	c.HTML(http.StatusOK, "attendance_report.html", gin.H{"Attendance": attendance, "ClinicName": getDisplayClinicName(c), "ClinicTitle": getDisplayClinicName(c)})
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
	c.HTML(http.StatusOK, "admin_dashboard.html", gin.H{
		"CashTotal":     d.CashTotal,
		"OnlineTotal":   d.OnlineTotal,
		"ExpenseTotal":  d.ExpenseTotal,
		"FinalBalance":  d.FinalBalance,
		"DoctorPresent": d.DoctorPresent,
		"SisterPresent": d.SisterPresent,
		"ClinicName":    getDisplayClinicName(c),
		"ClinicTitle":   getDisplayClinicName(c),
	})
}
