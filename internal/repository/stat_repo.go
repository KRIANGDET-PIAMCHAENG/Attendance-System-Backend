package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"math"
	//"strconv"
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

// 🌟 Get Working Hours Statistic (สถิติชั่วโมงการทำงาน สำหรับพนักงานดูของตัวเอง)
func (r *StatRepo) GetWorkingHoursStatistic(userID string) (map[string]interface{}, error) {
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

	var totalHours, weeklyHours, monthlyHours, yearlyHours float64
	totalMap := make(map[string]float64)

	// 🌟 เรียกใช้ Struct ที่เราประกาศไว้แล้ว เพื่อล็อกลำดับฟิลด์เวลาแปลงเป็น JSON
	var weekStat WeekStat
	var monthStat MonthStat
	var yearStat YearStat

	distinctYears, distinctMonths, distinctWeeks := make(map[int]bool), make(map[string]bool), make(map[string]bool)

	for _, rec := range records {
		inStr := strings.Split(rec.CheckIn, "+")[0]
		outStr := strings.Split(rec.CheckOut, "+")[0]

		inTime, _ := time.Parse("15:04:05", inStr)
		outTime, _ := time.Parse("15:04:05", outStr)
		dur := outTime.Sub(inTime).Hours()

		if dur >= 5.0 {
			dur -= 1.0
		} // หักพักเที่ยง
		if dur < 0 {
			dur = 0
		}
		dur = math.Round(dur*100) / 100

		totalHours += dur
		y, w := rec.Date.ISOWeek()
		distinctYears[rec.Date.Year()] = true
		distinctMonths[fmt.Sprintf("%d-%d", rec.Date.Year(), rec.Date.Month())] = true
		distinctWeeks[fmt.Sprintf("%d-%d", y, w)] = true

		thaiYear := (rec.Date.Year() + 543) % 100
		totalMap[fmt.Sprintf("%d", thaiYear)] += dur

		// ยัดใส่ตัวแปร WeekStat (จันทร์ - อาทิตย์)
		if y == currYear && w == currWeek {
			weeklyHours += dur
			switch rec.Date.Weekday() {
			case time.Monday: weekStat.Mon += dur
			case time.Tuesday: weekStat.Tue += dur
			case time.Wednesday: weekStat.Wed += dur
			case time.Thursday: weekStat.Thu += dur
			case time.Friday: weekStat.Fri += dur
			case time.Saturday: weekStat.Sat += dur
			case time.Sunday: weekStat.Sun += dur
			}
		}

		// ยัดใส่ตัวแปร MonthStat (วันที่ 1 - 31)
		if rec.Date.Year() == now.Year() && rec.Date.Month() == currMonth {
			monthlyHours += dur
			switch rec.Date.Day() {
			case 1: monthStat.D1 += dur; case 2: monthStat.D2 += dur; case 3: monthStat.D3 += dur; case 4: monthStat.D4 += dur; case 5: monthStat.D5 += dur
			case 6: monthStat.D6 += dur; case 7: monthStat.D7 += dur; case 8: monthStat.D8 += dur; case 9: monthStat.D9 += dur; case 10: monthStat.D10 += dur
			case 11: monthStat.D11 += dur; case 12: monthStat.D12 += dur; case 13: monthStat.D13 += dur; case 14: monthStat.D14 += dur; case 15: monthStat.D15 += dur
			case 16: monthStat.D16 += dur; case 17: monthStat.D17 += dur; case 18: monthStat.D18 += dur; case 19: monthStat.D19 += dur; case 20: monthStat.D20 += dur
			case 21: monthStat.D21 += dur; case 22: monthStat.D22 += dur; case 23: monthStat.D23 += dur; case 24: monthStat.D24 += dur; case 25: monthStat.D25 += dur
			case 26: monthStat.D26 += dur; case 27: monthStat.D27 += dur; case 28: monthStat.D28 += dur; case 29: monthStat.D29 += dur; case 30: monthStat.D30 += dur
			case 31: monthStat.D31 += dur
			}
		}

		// ยัดใส่ตัวแปร YearStat (ม.ค. - ธ.ค.)
		if rec.Date.Year() == now.Year() {
			yearlyHours += dur
			switch rec.Date.Month() {
			case time.January: yearStat.Jan += dur
			case time.February: yearStat.Feb += dur
			case time.March: yearStat.Mar += dur
			case time.April: yearStat.Apr += dur
			case time.May: yearStat.May += dur
			case time.June: yearStat.Jun += dur
			case time.July: yearStat.Jul += dur
			case time.August: yearStat.Aug += dur
			case time.September: yearStat.Sep += dur
			case time.October: yearStat.Oct += dur
			case time.November: yearStat.Nov += dur
			case time.December: yearStat.Dec += dur
			}
		}
	}

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
		"week":                 weekStat,  // 🌟 บังคับเรียงลำดับด้วย Struct
		"month":                monthStat, // 🌟 บังคับเรียงลำดับด้วย Struct
		"year":                 yearStat,  // 🌟 บังคับเรียงลำดับด้วย Struct
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