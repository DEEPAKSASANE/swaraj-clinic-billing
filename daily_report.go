package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

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
		"Reports":     reports,
		"SearchDate":  searchDate,
		"ClinicName":  getDisplayClinicName(c),
		"ClinicTitle": getClinicTitle(getCurrentClinicName(c)),
	})
}
