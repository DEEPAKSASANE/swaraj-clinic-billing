package main

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine) {

	r.GET("/login", loginPage)
	r.POST("/login", login)
	r.GET("/logout", logout)

	r.GET("/register", registerPage)
	r.POST("/register", registerUser)

	r.GET("/dashboard", adminOnly(dashboard))

	r.GET("/staff-dashboard", loginRequired(staffDashboardPage))

	r.GET("/attendance", loginRequired(attendanceInTimePage))
	r.POST("/attendance/intime", loginRequired(saveInTime))
	r.POST("/attendance/outtime", loginRequired(saveOutTime))

	r.GET("/attendance-report", adminOnly(attendanceReportPage))

	r.GET("/collection", loginRequired(collectionPage))
	r.POST("/collection", loginRequired(saveCollection))

	r.GET("/daily-report", loginRequired(dailyReportPage))
}
