package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func collectionPage(c *gin.Context) {
	c.HTML(http.StatusOK, "collection.html", gin.H{"ClinicName": getDisplayClinicName(c), "ClinicTitle": getDisplayClinicName(c)})
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
		"FromDate":        fromDate, "ToDate": toDate, "ClinicName": getDisplayClinicName(c),
		"ClinicTitle": getDisplayClinicName(c),
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
	x.ClinicName = getDisplayClinicName(c)
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
func collectionDeepakPage(c *gin.Context) {
	c.HTML(http.StatusOK, "collection_deepak.html", gin.H{"ClinicName": getDisplayClinicName(c), "ClinicTitle": getDisplayClinicName(c)})
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

	c.Redirect(http.StatusSeeOther, "/collection-report-deepak")
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
		"ClinicName":      getDisplayClinicName(c),
	})
}
