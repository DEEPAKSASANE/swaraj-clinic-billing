package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

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
func getCurrentStaffName(c *gin.Context) string {
	staffName, err := c.Cookie("name")
	if err == nil && strings.TrimSpace(staffName) != "" {
		return strings.TrimSpace(staffName)
	}

	username := getCurrentUsername(c)
	if username == "" {
		return ""
	}

	var name string
	err = db.QueryRow("SELECT COALESCE(name,'') FROM users WHERE LOWER(TRIM(username))=LOWER(TRIM($1))", username).Scan(&name)
	if err != nil || strings.TrimSpace(name) == "" {
		return username
	}
	return strings.TrimSpace(name)
}
func getCurrentClinicName(c *gin.Context) string {
	clinicName, err := c.Cookie("clinic_name")
	if err != nil {
		return ""
	}
	return clinicName
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
