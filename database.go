package main

import (
	"fmt"
	"log"
)

func createTables() {
	// USERS TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(100),
			address TEXT,
			mobile VARCHAR(15),
			username VARCHAR(50) UNIQUE NOT NULL,
			password VARCHAR(100) NOT NULL,
			role VARCHAR(20) DEFAULT 'doctor',
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			selfie_path TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("users table error:", err)
	}
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) DEFAULT 'doctor'`)
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)

	// DEFAULT ADMIN USER
	if _, err := db.Exec(`
		INSERT INTO users(name,address,mobile,username,password,role,clinic_name) VALUES
		('Super Admin','All Clinics','9999999990','superadmin','super123','superadmin','All'),
		('Admin Kalwa','Kalwa','9999999991','admin_kalwa','admin123','admin','Kalwa'),
		('Doctor Kalwa','Kalwa','9999999992','doctor_kalwa','doctor123','doctor','Kalwa'),
		('Sister Kalwa','Kalwa','9999999993','sister_kalwa','sister123','sister','Kalwa'),
		('Admin Vashi','Vashi','9999999994','admin_vashi','admin123','admin','Vashi'),
		('Doctor Vashi','Vashi','9999999995','doctor_vashi','doctor123','doctor','Vashi'),
		('Sister Vashi','Vashi','9999999996','sister_vashi','sister123','sister','Vashi'),
		('Admin Byculla','Byculla','9999999997','admin_byculla','admin123','admin','Byculla'),
		('Doctor Byculla','Byculla','9999999998','doctor_byculla','doctor123','doctor','Byculla'),
		('Sister Byculla','Byculla','9999999999','sister_byculla','sister123','sister','Byculla')
		ON CONFLICT (username) DO NOTHING
	`); err != nil {
		log.Println("admin insert error:", err)
	}

	// staff_attendance table removed. Attendance report now uses attendance_intime + attendance_outtime.

	// OPTIONAL SEPARATE IN/OUT TIME LOG TABLES
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attendance_intime (
			id SERIAL PRIMARY KEY,
			staff_name VARCHAR(100) NOT NULL,
			role VARCHAR(50) NOT NULL,
			attendance_date DATE DEFAULT CURRENT_DATE,
			in_time TIME DEFAULT CURRENT_TIME,
			clinic_name VARCHAR(20) NOT NULL CHECK (clinic_name IN ('Kalwa','Vashi','Byculla')),
			selfie_path VARCHAR(500),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("attendance_intime table error:", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS attendance_outtime (
			id SERIAL PRIMARY KEY,
			staff_name VARCHAR(100) NOT NULL,
			role VARCHAR(50) NOT NULL,
			attendance_date DATE DEFAULT CURRENT_DATE,
			out_time TIME DEFAULT CURRENT_TIME,
			clinic_name VARCHAR(20) NOT NULL CHECK (clinic_name IN ('Kalwa','Vashi','Byculla')),
			selfie_path VARCHAR(500),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("attendance_outtime table error:", err)
	}

	// DAILY COLLECTIONS TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS daily_collections (
			id SERIAL PRIMARY KEY,
			collection_date DATE,
			cash_amount NUMERIC(12,2) DEFAULT 0,
			online_amount NUMERIC(12,2) DEFAULT 0,
			expense_amount NUMERIC(12,2) DEFAULT 0,
			expense_reason TEXT,
			remark TEXT,
			entered_by VARCHAR(100),
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("daily_collections table error:", err)
	}

	// ADD MISSING COLUMNS IF OLD TABLE EXISTS
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS cash_amount NUMERIC(12,2) DEFAULT 0`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS online_amount NUMERIC(12,2) DEFAULT 0`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS expense_amount NUMERIC(12,2) DEFAULT 0`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS expense_reason TEXT`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS remark TEXT`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS entered_by VARCHAR(100)`)
	db.Exec(`ALTER TABLE daily_collections ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)

	// DAILY COLLECTIONS ONLINE TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS daily_collections_online (
			id SERIAL PRIMARY KEY,
			collection_date DATE NOT NULL,
			online_amount NUMERIC(12,2) DEFAULT 0,
			expense_amount NUMERIC(12,2) DEFAULT 0,
			remark TEXT,
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("daily_collections_online table error:", err)
	}

	// INVOICE MASTER TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS invoices12 (
			id SERIAL PRIMARY KEY,
			customer_name VARCHAR(150),
			mobile VARCHAR(20),
			address TEXT,
			age INT,
			gender VARCHAR(20),
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("invoices12 table error:", err)
	}

	// ADD MISSING COLUMNS IF OLD INVOICE TABLE EXISTS
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS mobile VARCHAR(20)`)
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS address TEXT`)
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS age INT`)
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS gender VARCHAR(20)`)
	db.Exec(`ALTER TABLE invoices12 ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)
	db.Exec(`ALTER TABLE tests12 ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)
	db.Exec(`ALTER TABLE daily_collections_online ADD COLUMN IF NOT EXISTS clinic_name VARCHAR(100) DEFAULT 'Kalwa'`)

	// INVOICE DETAILS TABLE
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS tests12 (
			id SERIAL PRIMARY KEY,
			invoice_id INT REFERENCES invoices12(id) ON DELETE CASCADE,
			test_name VARCHAR(200),
			price NUMERIC(12,2),
			discount NUMERIC(12,2),
			total_amount NUMERIC(12,2),
			total_amount_words TEXT,
			clinic_name VARCHAR(100) DEFAULT 'Kalwa',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		log.Println("tests12 table error:", err)
	}

	// NORMALIZE OLD DATA: keep clinic_name as short code in DB (Kalwa/Vashi/Byculla/All).
	// Display name is handled by getClinicDisplayName().
	db.Exec(`UPDATE users SET username=TRIM(username), role=LOWER(TRIM(role)), clinic_name=TRIM(clinic_name)`)
	db.Exec(`UPDATE users SET clinic_name='Kalwa' WHERE LOWER(clinic_name) IN ('kalwa','swaraj clinic kalwa')`)
	db.Exec(`UPDATE users SET clinic_name='Vashi' WHERE LOWER(clinic_name) IN ('vashi','wellness clinic vashi')`)
	db.Exec(`UPDATE users SET clinic_name='Byculla' WHERE LOWER(clinic_name) IN ('byculla','wellness clinic byculla')`)
	db.Exec(`UPDATE users SET clinic_name='All' WHERE LOWER(clinic_name) IN ('all','all clinic','all clinics','all centers')`)

	db.Exec(`UPDATE daily_collections SET clinic_name='Kalwa' WHERE LOWER(TRIM(clinic_name)) IN ('kalwa','swaraj clinic kalwa')`)
	db.Exec(`UPDATE daily_collections SET clinic_name='Vashi' WHERE LOWER(TRIM(clinic_name)) IN ('vashi','wellness clinic vashi')`)
	db.Exec(`UPDATE daily_collections SET clinic_name='Byculla' WHERE LOWER(TRIM(clinic_name)) IN ('byculla','wellness clinic byculla')`)

	db.Exec(`UPDATE daily_collections_online SET clinic_name='Kalwa' WHERE LOWER(TRIM(clinic_name)) IN ('kalwa','swaraj clinic kalwa')`)
	db.Exec(`UPDATE daily_collections_online SET clinic_name='Vashi' WHERE LOWER(TRIM(clinic_name)) IN ('vashi','wellness clinic vashi')`)
	db.Exec(`UPDATE daily_collections_online SET clinic_name='Byculla' WHERE LOWER(TRIM(clinic_name)) IN ('byculla','wellness clinic byculla')`)

	db.Exec(`UPDATE invoices12 SET clinic_name='Kalwa' WHERE LOWER(TRIM(clinic_name)) IN ('kalwa','swaraj clinic kalwa')`)
	db.Exec(`UPDATE invoices12 SET clinic_name='Vashi' WHERE LOWER(TRIM(clinic_name)) IN ('vashi','wellness clinic vashi')`)
	db.Exec(`UPDATE invoices12 SET clinic_name='Byculla' WHERE LOWER(TRIM(clinic_name)) IN ('byculla','wellness clinic byculla')`)

	db.Exec(`UPDATE tests12 SET clinic_name='Kalwa' WHERE LOWER(TRIM(clinic_name)) IN ('kalwa','swaraj clinic kalwa')`)
	db.Exec(`UPDATE tests12 SET clinic_name='Vashi' WHERE LOWER(TRIM(clinic_name)) IN ('vashi','wellness clinic vashi')`)
	db.Exec(`UPDATE tests12 SET clinic_name='Byculla' WHERE LOWER(TRIM(clinic_name)) IN ('byculla','wellness clinic byculla')`)

	fmt.Println("Database tables checked/created successfully")
}
