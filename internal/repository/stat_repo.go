package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"math"
	"strconv"
	"strings"

)

type StatRepo struct {
	db *gorm.DB
}

func NewStatRepo(db *gorm.DB) *StatRepo {
	return &StatRepo{db: db}
}

// 🌟 Get User Statistic (คำนวณตามปีงบประมาณ)
func (r *StatRepo) GetUserStatistic(userID string, year int) (map[string]interface{}, error) {
	// 1. ดึง Config วันเริ่มปีงบประมาณ (Budget Year)
	var budgetConfig struct {
		ConfigValue string `gorm:"column:config_value"`
	}
	r.db.Table("system_configs").Where("config_key = ?", "budget_year").First(&budgetConfig)
	
	startMonth, startDay := 1, 1
	if budgetConfig.ConfigValue != "" {
		var cfg map[string]int
		json.Unmarshal([]byte(budgetConfig.ConfigValue), &cfg)
		if m, ok := cfg["month"]; ok { startMonth = m }
		if d, ok := cfg["day"]; ok { startDay = d }
	}

	// คำนวณวันเริ่ม-สิ้นสุด ปีงบประมาณ
	var startDate, endDate time.Time
	if startMonth > 1 {
		// เช่น เริ่มเดือน 10 ปี 2026 -> 2025-10-01 ถึง 2026-09-30
		startDate = time.Date(year-1, time.Month(startMonth), startDay, 0, 0, 0, 0, time.Local)
		endDate = startDate.AddDate(1, 0, -1)
	} else {
		// เริ่มเดือน 1 ปี 2026 -> 2026-01-01 ถึง 2026-12-31
		startDate = time.Date(year, time.Month(startMonth), startDay, 0, 0, 0, 0, time.Local)
		endDate = startDate.AddDate(1, 0, -1)
	}

	// ⏳ วันสิ้นสุดการคำนวณสถิติ (ถ้าปีปัจจุบัน จะคิดถึงแค่วันนี้ เพื่อไม่ให้วันในอนาคตกลายเป็นขาดงาน)
	today := time.Now()
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
	calcEndDate := endDate
	if todayDate.Before(endDate) {
		calcEndDate = todayDate
	}

	// 2. ดึง Config เวลาเข้างาน (เพื่อใช้เช็คสาย)
	var attConfig struct {
		ConfigValue string `gorm:"column:config_value"`
	}
	r.db.Table("system_configs").Where("config_key = ?", "attendance_time").First(&attConfig)
	checkInHour, checkInMinute := 8, 30 // Default 08:30
	if attConfig.ConfigValue != "" {
		var cfg map[string]interface{}
		json.Unmarshal([]byte(attConfig.ConfigValue), &cfg)
		if cTime, ok := cfg["check-in-time"].(map[string]interface{}); ok {
			if h, ok := cTime["hour"].(float64); ok { checkInHour = int(h) }
			if m, ok := cTime["minute"].(float64); ok { checkInMinute = int(m) }
		}
	}
	checkInLimitStr := fmt.Sprintf("%02d:%02d:00", checkInHour, checkInMinute)

	// 3. ดึงวันหยุด (สมมติชื่อตาราง company_holidays ถ้าของลูกพี่เป็นชื่ออื่นแก้ด้วยนะครับ)
	var holidayDates []time.Time
	r.db.Table("company_holidays").
		Where("holiday_date >= ? AND holiday_date <= ?", startDate, endDate).
		Pluck("holiday_date", &holidayDates)
	holidayMap := make(map[string]bool)
	for _, hd := range holidayDates {
		holidayMap[hd.Format("2006-01-02")] = true
	}

	// 4. ดึงสิทธิ์การลา (Quotas)
	type LeaveBalanceRow struct {
		LeaveType string  `gorm:"column:name_en"`
		Quota     float64 `gorm:"column:days_allowed"`
	}
	var balances []LeaveBalanceRow
	r.db.Table("leave_balances lb").
		Select("lt.name_en, lb.days_allowed").
		Joins("JOIN leave_types lt ON lb.leave_type_id = lt.id").
		Where("lb.user_id = ? AND lb.year = ?", userID, year).
		Scan(&balances)

	leaveQuotas := make(map[string]float64)
	for _, b := range balances {
		leaveQuotas[b.LeaveType] = b.Quota
	}

	// ประเภทการลาทั้งหมดในระบบ
	var allTypes []struct{ NameEn string `gorm:"column:name_en"` }
	r.db.Table("leave_types").Scan(&allTypes)
	for _, t := range allTypes {
		if _, ok := leaveQuotas[t.NameEn]; !ok {
			leaveQuotas[t.NameEn] = 0 // ถ้าไม่เจอในตาราง quota ให้เป็น 0 ไว้ก่อน
		}
	}

	// 5. ดึงข้อมูลใบลางานที่ "อนุมัติแล้ว"
	type LeaveReq struct {
		LeaveType   string    `gorm:"column:leave_type"`
		DateFrom    time.Time `gorm:"column:date_from"`
		DateTo      time.Time `gorm:"column:date_to"`
		FromMorning bool      `gorm:"column:from_date_morning"`
		ToMorning   bool      `gorm:"column:to_date_morning"`
	}
	var leaves []LeaveReq
	r.db.Table("leave_requests").
		Where("user_id = ? AND status = 'approved' AND date_from <= ? AND date_to >= ?", userID, endDate, startDate).
		Scan(&leaves)

	leaveUsed := make(map[string]float64)
	fullLeaveDaysMap := make(map[string]bool)

	for _, lv := range leaves {
		curr := lv.DateFrom
		for !curr.After(lv.DateTo) {
			dateStr := curr.Format("2006-01-02")
			isWeekend := curr.Weekday() == time.Saturday || curr.Weekday() == time.Sunday
			isHoliday := holidayMap[dateStr]

			if !isWeekend && !isHoliday {
				dayVal := 1.0
				// คำนวณลาครึ่งวัน
				if curr.Equal(lv.DateFrom) && !lv.FromMorning { dayVal = 0.5 }
				if curr.Equal(lv.DateTo) && lv.ToMorning { dayVal = 0.5 }
				if curr.Equal(lv.DateFrom) && curr.Equal(lv.DateTo) {
					if !lv.FromMorning || lv.ToMorning { dayVal = 0.5 }
				}

				leaveUsed[lv.LeaveType] += dayVal
				if dayVal == 1.0 {
					fullLeaveDaysMap[dateStr] = true // จดไว้ว่าวันนี้ลาเต็มวัน ไม่ถือว่าขาดงาน
				}
			}
			curr = curr.AddDate(0, 0, 1)
		}
	}

	// 6. ดึงข้อมูลการสแกนนิ้วเข้างาน
	type AttRecord struct {
		Date    time.Time `gorm:"column:date"`
		CheckIn string    `gorm:"column:check_in"` 
	}
	var attendances []AttRecord
	r.db.Table("attendance").
		Select("date, check_in::text").
		Where("user_id = ? AND date >= ? AND date <= ?", userID, startDate, calcEndDate).
		Scan(&attendances)

	attMap := make(map[string]string)
	for _, a := range attendances {
		attMap[a.Date.Format("2006-01-02")] = a.CheckIn
	}

	
	
	// 7. 🚀 จำลองรายวันเพื่อคำนวณสถิติ
	var totalWorkDays, actualWorkDays, onTimeDays, lateDays, absentDays int

	curr := startDate
	for !curr.After(calcEndDate) {
		dateStr := curr.Format("2006-01-02")
		isWeekend := curr.Weekday() == time.Saturday || curr.Weekday() == time.Sunday
		isHoliday := holidayMap[dateStr]

		// ถือว่าเป็นวันทำงานปกติ
		if !isWeekend && !isHoliday {
			totalWorkDays++

			checkInTime, hasAtt := attMap[dateStr]
			if hasAtt && checkInTime != "" {
				actualWorkDays++
				if checkInTime <= checkInLimitStr {
					onTimeDays++
				} else {
					lateDays++
				}
			} else {
				// ไม่ได้เข้างาน เช็คว่าลาเต็มวันไหม? และเลยเมื่อวานมาหรือยัง?
				if !fullLeaveDaysMap[dateStr] && curr.Before(todayDate) {
					absentDays++
				}
			}
		}
		curr = curr.AddDate(0, 0, 1)
	}

	// 8. จัดรูปแบบผลลัพธ์ใส่ JSON
	totalLeaveDays := 0.0
	overLeaveDays := 0.0
	leavesOutput := make(map[string]interface{})

	for _, t := range allTypes {
		used := leaveUsed[t.NameEn]
		quota := leaveQuotas[t.NameEn]
		
		totalLeaveDays += used
		if used > quota {
			overLeaveDays += (used - quota)
		}

		leavesOutput[t.NameEn] = map[string]interface{}{
			"used_days":  used,
			"quota_days": quota,
		}
	}

	return map[string]interface{}{
		"total_work_days":  totalWorkDays,
		"actual_work_days": actualWorkDays,
		"attendance_detail": map[string]interface{}{
			"on_time_days": onTimeDays,
			"late_days":    lateDays,
			"absent_days":  absentDays,
		},
		"leave_detail": map[string]interface{}{
			"total_leave_days": totalLeaveDays,
			"over_leave_days":  overLeaveDays,
			"leaves":           leavesOutput,
		},
	}, nil
}

func (r *StatRepo) GetWorkingHoursStatistic(userID string) (map[string]interface{}, error) {
	// ดึงข้อมูลการเข้า-ออกงานที่มีทั้ง check_in และ check_out
	type AttWorkHour struct {
		Date     time.Time `gorm:"column:date"`
		CheckIn  string    `gorm:"column:check_in"`
		CheckOut string    `gorm:"column:check_out"`
	}
	var records []AttWorkHour
	r.db.Table("attendance").
		Select("date, check_in::text, check_out::text").
		Where("user_id = ? AND check_in IS NOT NULL AND check_out IS NOT NULL", userID).
		Scan(&records)

	now := time.Now()
	currYear, currWeek := now.ISOWeek()
	currMonth := now.Month()

	// ตัวแปรเก็บผลรวม
	var totalHours, weeklyHours, monthlyHours, yearlyHours float64

	// เตรียม Map สำหรับจัดกลุ่ม
	totalMap := make(map[string]float64)
	weekMap := map[string]float64{"อา.": 0, "จ.": 0, "อ.": 0, "พ.": 0, "พฤ.": 0, "ศ.": 0, "ส.": 0}
	monthMap := make(map[string]float64)
	for i := 1; i <= 31; i++ {
		monthMap[strconv.Itoa(i)] = 0
	}
	yearMap := map[string]float64{"ม.ค.": 0, "ก.พ.": 0, "มี.ค.": 0, "เม.ย.": 0, "พ.ค.": 0, "มิ.ย.": 0, "ก.ค.": 0, "ส.ค.": 0, "ก.ย.": 0, "ต.ค.": 0, "พ.ย.": 0, "ธ.ค.": 0}

	distinctYears := make(map[int]bool)
	distinctMonths := make(map[string]bool)
	distinctWeeks := make(map[string]bool)

	thaiDays := []string{"อา.", "จ.", "อ.", "พ.", "พฤ.", "ศ.", "ส."}
	thaiMonths := []string{"ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.", "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}

	for _, rec := range records {
		// หั่นเอาแค่ 08:30:00 (เผื่อข้อมูลมาเป็น 08:30:00+00)
		inStr := strings.Split(rec.CheckIn, "+")[0]
		outStr := strings.Split(rec.CheckOut, "+")[0]

		// แปลงเวลา (Format "15:04:05")
		inTime, errIn := time.Parse("15:04:05", inStr)
		outTime, errOut := time.Parse("15:04:05", outStr)
		
		// เผื่อข้อมูลเก็บบางทีมาแบบ "15:04"
		if errIn != nil { inTime, _ = time.Parse("15:04", inStr) }
		if errOut != nil { outTime, _ = time.Parse("15:04", outStr) }

		dur := outTime.Sub(inTime).Hours()
		
		// 🌟 หักพักเที่ยง (ถ้าทำงานเกิน 5 ชั่วโมง ให้หักออก 1 ชม.)
		if dur >= 5.0 {
			dur -= 1.0
		}
		if dur < 0 {
			dur = 0
		}

		dur = math.Round(dur*100) / 100 // ปัดเศษ 2 ตำแหน่ง

		totalHours += dur
		y, w := rec.Date.ISOWeek()
		
		distinctYears[rec.Date.Year()] = true
		distinctMonths[fmt.Sprintf("%d-%d", rec.Date.Year(), rec.Date.Month())] = true
		distinctWeeks[fmt.Sprintf("%d-%d", y, w)] = true

		// จัดกลุ่มตามปี พ.ศ. 2 ตัวท้าย เช่น 2026+543 = 2569 -> "69"
		thaiYear := (rec.Date.Year() + 543) % 100
		totalMap[fmt.Sprintf("%d", thaiYear)] += dur

		// สถิติสัปดาห์ปัจจุบัน
		if y == currYear && w == currWeek {
			weeklyHours += dur
			dayIdx := int(rec.Date.Weekday()) // 0=Sunday
			weekMap[thaiDays[dayIdx]] += dur
		}

		// สถิติเดือนปัจจุบัน
		if rec.Date.Year() == now.Year() && rec.Date.Month() == currMonth {
			monthlyHours += dur
			dayStr := strconv.Itoa(rec.Date.Day())
			monthMap[dayStr] += dur
		}

		// สถิติปีปัจจุบัน
		if rec.Date.Year() == now.Year() {
			yearlyHours += dur
			monthIdx := int(rec.Date.Month()) - 1
			yearMap[thaiMonths[monthIdx]] += dur
		}
	}

	// คำนวณค่าเฉลี่ย
	var totalAvg, yearlyAvg, monthlyAvg, weeklyAvg float64
	if len(distinctYears) > 0 {
		totalAvg = totalHours / float64(len(distinctYears))
		yearlyAvg = totalAvg
	}
	if len(distinctMonths) > 0 {
		monthlyAvg = totalHours / float64(len(distinctMonths))
	}
	if len(distinctWeeks) > 0 {
		weeklyAvg = totalHours / float64(len(distinctWeeks))
	}

	// ประกอบร่าง JSON ส่งกลับไป
	return map[string]interface{}{
		"total-working-hour":   math.Round(totalHours*100) / 100,
		"total-average-hour":   math.Round(totalAvg*100) / 100,
		"weekly-working-hour":  math.Round(weeklyHours*100) / 100,
		"weekly-average-hour":  math.Round(weeklyAvg*100) / 100,
		"monthly-working-hour": math.Round(monthlyHours*100) / 100,
		"monthly-average-hour": math.Round(monthlyAvg*100) / 100,
		"yearly-working-hour":  math.Round(yearlyHours*100) / 100,
		"yearly-average-hour":  math.Round(yearlyAvg*100) / 100,
		"total":                totalMap,
		"week":                 weekMap,
		"month":                monthMap,
		"year":                 yearMap,
	}, nil
}

// 🌟 Get Statistic Filter Range (หาขอบเขตวันที่มีข้อมูลการเข้างาน)
func (r *StatRepo) GetStatisticFilterRange(userID string) (map[string]interface{}, error) {
	var bounds struct {
		MinDate *time.Time `gorm:"column:min_date"`
		MaxDate *time.Time `gorm:"column:max_date"`
	}

	// หาดือน/ปี ที่เก่าที่สุด และ ใหม่ที่สุด จากตาราง attendance
	r.db.Table("attendance").
		Select("MIN(date) as min_date, MAX(date) as max_date").
		Where("user_id = ?", userID).
		Scan(&bounds)

	// ถ้าคนนี้ยังไม่มีประวัติการเข้างานเลย (กันแอปพัง)
	now := time.Now()
	start := now
	end := now

	if bounds.MinDate != nil {
		start = *bounds.MinDate
	}
	if bounds.MaxDate != nil {
		end = *bounds.MaxDate
	}

	// ส่งกลับในรูปแบบ ISO8601 (RFC3339) ตามที่ Frontend ต้องการเป๊ะๆ
	return map[string]interface{}{
		"start": start.Format("2006-01-02T15:04:05.000Z"),
		"end":   end.Format("2006-01-02T15:04:05.000Z"),
	}, nil
}