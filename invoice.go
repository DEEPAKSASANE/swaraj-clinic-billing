package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func formPage(c *gin.Context) {
	c.HTML(http.StatusOK, "form.html", gin.H{
		"ClinicName":  getDisplayClinicName(c),
		"ClinicTitle": getDisplayClinicName(c),
	})
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
		InvoiceNo:        invoiceID,
		ClinicName:       clinicName,
		InvoiceDatetime:  time.Now().Format("02-Jan-2006 03:04 PM"),
		CustomerName:     customerName,
		Gender:           gender,
		Mobile:           mobile,
		Age:              age,
		Address:          address,
		Tests:            tests,
		TotalAmount:      totalAmount,
		TotalAmountWords: convertToWords(int(totalAmount)),
	}
	fillInvoiceClinicDetails(c, &invoice)

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
			ClinicName: getDisplayClinicName(c),
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
			ClinicName: getDisplayClinicName(c),
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
		fillInvoiceClinicDetails(c, &invoice)

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
