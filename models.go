package main

import (
	"database/sql"
	"time"
)

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

	SelfiePath string // Add this back

	InImage  string
	OutImage string
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
	ClinicAddress    string
	ClinicMobile     string
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
	ClinicName string
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
