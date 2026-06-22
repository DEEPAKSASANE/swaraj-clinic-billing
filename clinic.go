package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"strings"
)

func normalizeClinicName(clinic string) string {
	c := strings.ToLower(strings.TrimSpace(clinic))
	switch c {
	case "kalwa", "swaraj clinic kalwa":
		return "Kalwa"
	case "vashi", "wellness clinic vashi":
		return "Vashi"
	case "byculla", "wellness clinic byculla":
		return "Byculla"
	case "all", "all clinic", "all clinics", "all centers":
		return "All"
	default:
		return strings.TrimSpace(clinic)
	}
}
func getClinicDisplayName(clinic string) string {
	switch normalizeClinicName(clinic) {
	case "Kalwa":
		return "Swaraj Clinic Kalwa"
	case "Vashi":
		return "Wellness Clinic Vashi"
	case "Byculla":
		return "Wellness Clinic Byculla"
	case "All":
		return "All Clinics"
	default:
		return "Clinic Management System"
	}
}
func getClinicTitle(clinic string) string {
	switch normalizeClinicName(clinic) {
	case "Kalwa":
		return "🏥 Swaraj EMR Kalwa - Daily Report"
	case "Vashi":
		return "🏥 Wellness EMR Vashi - Daily Report"
	case "Byculla":
		return "🏥 Wellness EMR Byculla - Daily Report"
	case "All":
		return "🏥 All Clinics EMR - Daily Report"
	default:
		return "🏥 EMR Daily Report"
	}
}
func getClinicAddress(clinic string) string {
	switch normalizeClinicName(clinic) {
	case "Kalwa":
		return "Swaraj Clinic Railway Station Platform No. 1 Exit, Kalwa"
	case "Vashi":
		return "Wellness Clinic Sector-30, Vashi Station, Inorbit Mall, near Platform No. 1, VRSCCL Company, Vashi"
	case "Byculla":
		return "Wellness Clinic Byculla Near Ticket Window (West)"
	case "All":
		return "All Clinics"
	default:
		return "Clinic Address"
	}
}
func getClinicMobile(clinic string) string {
	switch normalizeClinicName(clinic) {
	case "Kalwa":
		return "9167265266"
	case "Vashi", "Byculla", "All":
		return "8369422482"
	default:
		return "8369422482"
	}
}
func fillInvoiceClinicDetails(c *gin.Context, invoice *Invoice) {
	clinicCode := normalizeClinicName(invoice.ClinicName)
	if clinicCode == "" || clinicCode == "Clinic Management System" {
		clinicCode = getCurrentClinicName(c)
	}
	if clinicCode == "" || clinicCode == "All" {
		clinicCode = getCurrentClinicName(c)
	}
	if clinicCode == "" || clinicCode == "All" {
		clinicCode = "Kalwa"
	}

	invoice.ClinicName = getClinicDisplayName(clinicCode)
	invoice.ClinicAddress = getClinicAddress(clinicCode)
	invoice.ClinicMobile = getClinicMobile(clinicCode)
}
func getDisplayClinicName(c *gin.Context) string {
	displayName, err := c.Cookie("display_clinic_name")
	if err != nil || strings.TrimSpace(displayName) == "" {
		return getClinicDisplayName(getCurrentClinicName(c))
	}
	return displayName
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
