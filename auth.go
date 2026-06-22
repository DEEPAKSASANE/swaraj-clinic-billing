package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func loginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}
func login(c *gin.Context) {
	selectedClinicRaw := strings.TrimSpace(c.PostForm("clinic_name"))
	clinicName := normalizeClinicName(selectedClinicRaw)
	selectedRole := strings.ToLower(strings.TrimSpace(c.PostForm("role")))
	username := strings.TrimSpace(c.PostForm("username"))
	password := strings.TrimSpace(c.PostForm("password"))

	if clinicName == "" || selectedRole == "" || username == "" || password == "" {
		c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Please select clinic, role and enter username/password"})
		return
	}

	var id int
	var role string
	var dbClinicName string
	var fullName string

	err := db.QueryRow(`
		SELECT
			id,
			LOWER(TRIM(COALESCE(role,'doctor'))),
			TRIM(COALESCE(clinic_name,'')),
			TRIM(COALESCE(name,''))
		FROM users
		WHERE LOWER(TRIM(username)) = LOWER(TRIM($1))
		AND password = $2
	`, username, password).Scan(&id, &role, &dbClinicName, &fullName)

	if err != nil {
		fmt.Println("LOGIN FAILED USER/PASS:", username, err)
		c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Invalid username or password"})
		return
	}

	role = strings.ToLower(strings.TrimSpace(role))
	dbClinicName = normalizeClinicName(dbClinicName)

	if role != selectedRole {
		fmt.Println("LOGIN FAILED ROLE - Selected:", selectedRole, "DB:", role)
		c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Wrong role selected"})
		return
	}

	if role == "superadmin" {
		if !strings.EqualFold(clinicName, "All") {
			c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Super Admin साठी All Centers select करा"})
			return
		}
		dbClinicName = "All"
	} else {
		if strings.EqualFold(clinicName, "All") {
			c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Admin/Doctor/Sister साठी proper clinic select करा"})
			return
		}

		if !strings.EqualFold(dbClinicName, clinicName) {
			fmt.Println("LOGIN FAILED CLINIC - Selected:", clinicName, "DB:", dbClinicName)
			c.HTML(http.StatusOK, "login.html", gin.H{"Error": "Wrong clinic selected"})
			return
		}
	}

	fmt.Println("LOGIN SUCCESS - User:", username, "Role:", role, "Clinic:", dbClinicName)

	c.SetCookie("username", username, 3600*8, "/", "", false, false)
	c.SetCookie("name", fullName, 3600*8, "/", "", false, false)
	c.SetCookie("role", role, 3600*8, "/", "", false, false)
	c.SetCookie("clinic_name", dbClinicName, 3600*8, "/", "", false, false)
	c.SetCookie("display_clinic_name", getClinicDisplayName(dbClinicName), 3600*8, "/", "", false, false)

	if role == "admin" || role == "superadmin" {
		c.Redirect(http.StatusFound, "/dashboard")
		return
	}

	c.Redirect(http.StatusFound, "/staff-dashboard")
}
func logout(c *gin.Context) {
	c.SetCookie("username", "", -1, "/", "", false, false)
	c.SetCookie("name", "", -1, "/", "", false, false)
	c.SetCookie("role", "", -1, "/", "", false, false)
	c.SetCookie("clinic_name", "", -1, "/", "", false, false)
	c.SetCookie("display_clinic_name", "", -1, "/", "", false, false)
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
	clinicName := normalizeClinicName(c.PostForm("clinic_name"))
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
