package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"math"
	"strconv"
	"sort"
	//"strings"

)

type StatRepo struct {
	db *gorm.DB
}

func NewStatRepo(db *gorm.DB) *StatRepo {
	return &StatRepo{db: db}
}
// 🌟 Get User Statistic (อัปเดตโลจิกเหมือนฝั่ง Manager 100%)
func (r *StatRepo) GetUserStatistic(userID string, targetYear int) (map[string]interface{}, error) {
	// 1. 📅 โหลด Config ปีงบประมาณ (Budget Year)
	var byConfig struct {
		Month int `json:"month"`
		Day   int `json:"day"`
	}
	byConfig.Month = 1
	byConfig.Day = 1

	var rawBY string
	r.db.Table("system_configs").Where("config_key = 'budget_year'").Select("config_value").Scan(&rawBY)
	if rawBY != "" {
		json.Unmarshal([]byte(rawBY), &byConfig)
	}

	startYear := targetYear
	if byConfig.Month > 6 {
		startYear = targetYear - 1
	}
	startDate := time.Date(startYear, time.Month(byConfig.Month), byConfig.Day, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(1, 0, 0).AddDate(0, 0, -1)

	// 2. ⏱️ โหลด Config เวลาเข้างาน
	var attConfig struct {
		CheckInTime struct {
			Hour   int `json:"hour"`
			Minute int `json:"minute"`
		} `json:"check-in-time"`
	}
	attConfig.CheckInTime.Hour = 8
	attConfig.CheckInTime.Minute = 30

	var rawAtt string
	r.db.Table("system_configs").Where("config_key = 'attendance_time'").Select("config_value").Scan(&rawAtt)
	if rawAtt != "" {
		json.Unmarshal([]byte(rawAtt), &attConfig)
	}
	checkInLimit, _ := time.Parse("15:04:05", fmt.Sprintf("%02d:%02d:00", attConfig.CheckInTime.Hour, attConfig.CheckInTime.Minute))

	// 3. 🏖️ โหลดวันหยุด
	var holidays []time.Time
	r.db.Table("company_holidays").
		Select("holiday_date").
		Where("holiday_date >= ? AND holiday_date <= ?", startDate, endDate).
		Pluck("holiday_date", &holidays)

	holidayMap := make(map[string]bool)
	for _, h := range holidays {
		holidayMap[h.Format("2006-01-02")] = true
	}

	// 4. 💼 คำนวณวันทำงานทั้งหมด
	totalWorkDays := 0
	workDayMap := make(map[string]bool)
	now := time.Now()

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		dateStr := d.Format("2006-01-02")
		if holidayMap[dateStr] {
			continue
		}
		totalWorkDays++
		if !d.After(now) {
			workDayMap[dateStr] = true
		}
	}

	// 5. 🤒 โหลดข้อมูลการลา (Approved)
	type LeaveReq struct {
		LeaveType       string    `gorm:"column:name_en"`
		DateFrom        time.Time `gorm:"column:date_from"`
		DateTo          time.Time `gorm:"column:date_to"`
		FromDateMorning bool      `gorm:"column:from_date_morning"`
		ToDateMorning   bool      `gorm:"column:to_date_morning"`
	}
	var leaves []LeaveReq
	r.db.Table("leave_requests lr").
		Select("lt.name_en, lr.date_from, lr.date_to, lr.from_date_morning, lr.to_date_morning").
		Joins("JOIN leave_types lt ON lr.leave_type = lt.name_en").
		Where("lr.user_id = ? AND lr.status = 'approved' AND lr.date_from <= ? AND lr.date_to >= ?", userID, endDate, startDate).
		Scan(&leaves)

	leaveUsedPerType := make(map[string]float64)
	leaveDaysMap := make(map[string]float64)

	for _, l := range leaves {
		for d := l.DateFrom; !d.After(l.DateTo); d = d.AddDate(0, 0, 1) {
			if d.Before(startDate) || d.After(endDate) {
				continue
			}
			dateStr := d.Format("2006-01-02")
			if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday || holidayMap[dateStr] {
				continue
			}

			dayVal := 1.0
			if d.Format("2006-01-02") == l.DateFrom.Format("2006-01-02") && !l.FromDateMorning {
				dayVal -= 0.5
			}
			if d.Format("2006-01-02") == l.DateTo.Format("2006-01-02") && l.ToDateMorning {
				dayVal -= 0.5
			}

			leaveUsedPerType[l.LeaveType] += dayVal
			if !d.After(now) {
				leaveDaysMap[dateStr] += dayVal
			}
		}
	}

	// 6. ⏰ โหลดข้อมูลการสแกนเข้างาน (ต้องมีทั้ง In และ Out)
	type AttRecord struct {
		Date     time.Time
		CheckIn  string `gorm:"column:check_in"`
		CheckOut string `gorm:"column:check_out"`
	}
	var atts []AttRecord
	r.db.Table("attendance").
		Select("date, check_in, check_out").
		Where("user_id = ? AND date >= ? AND date <= ?", userID, startDate, now).
		Scan(&atts)

	attMap := make(map[string]bool)
	onTimeCount := 0
	lateCount := 0

	for _, a := range atts {
		dateStr := a.Date.Format("2006-01-02")
		if !workDayMap[dateStr] {
			continue
		}
		
		// 🌟 เงื่อนไขต้องมี CheckIn และ CheckOut เท่านั้น
		if a.CheckIn != "" && a.CheckOut != "" {
			attMap[dateStr] = true

			t, err := time.Parse("15:04:05", a.CheckIn)
			if err == nil {
				if t.After(checkInLimit) {
					lateCount++
				} else {
					onTimeCount++
				}
			}
		} else {
			attMap[dateStr] = false
		}
	}

	// 7. ❌ คำนวณวันขาดงาน
	absentCount := 0
	for dateStr, isWorkDay := range workDayMap {
		if !isWorkDay {
			continue
		}
		if attMap[dateStr] {
			continue // สมบูรณ์ ไม่ขาด
		}
		if leaveDaysMap[dateStr] >= 1.0 {
			continue // ลาเต็ม ไม่ขาด
		}
		// 🌟 ถ้าวันนี้ยังไม่ถึง 16:00 (ยังไม่เลิกงาน) ยังไม่นับขาด
		if dateStr == now.Format("2006-01-02") && now.Hour() < 16 {
			continue
		}
		absentCount++
	}

	// 8. 📊 โหลดโควต้าวันลา
	type Balance struct {
		LeaveType   string  `gorm:"column:name_en"`
		DaysAllowed float64 `gorm:"column:days_allowed"`
	}
	var balances []Balance
	r.db.Table("leave_balances lb").
		Select("lt.name_en, lb.days_allowed").
		Joins("JOIN leave_types lt ON lb.leave_type_id = lt.id").
		Where("lb.user_id = ? AND (lb.year = ? OR lb.year IS NULL)", userID, targetYear).
		Scan(&balances)

	quotaMap := make(map[string]float64)
	for _, b := range balances {
		quotaMap[b.LeaveType] = b.DaysAllowed
	}

	allLeaveTypes := []string{"sick", "personal", "vacation", "maternity", "paternity", "parental"}
	leavesMap := make(map[string]interface{})
	totalLeaveDays := 0.0
	totalOverLeave := 0.0

	for _, lt := range allLeaveTypes {
		used := leaveUsedPerType[lt]
		quota := quotaMap[lt]

		if used > quota {
			totalOverLeave += (used - quota)
		}
		totalLeaveDays += used

		leavesMap[lt] = map[string]interface{}{
			"used_days":  used,
			"quota_days": quota,
		}
	}

	return map[string]interface{}{
		"total_work_days":  totalWorkDays,
		"actual_work_days": float64(totalWorkDays) - totalLeaveDays,
		"attendance_detail": map[string]interface{}{
			"on_time_days": onTimeCount,
			"late_days":    lateCount,
			"absent_days":  absentCount,
		},
		"leave_detail": map[string]interface{}{
			"total_leave_days": totalLeaveDays,
			"over_leave_days":  totalOverLeave,
			"leaves":           leavesMap,
		},
	}, nil
}
// 🌟 Get Working Hours Statistic (อัปเดตโลจิกเหมือนฝั่ง Manager 100%)
func (r *StatRepo) GetWorkingHoursStatistic(userID string) (map[string]interface{}, error) {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := now.Month()

	// หยุดนับแค่วันนี้
	today := time.Date(currentYear, currentMonth, now.Day(), 0, 0, 0, 0, time.Local)

	offset := int(now.Weekday())
	startOfWeek := time.Date(currentYear, currentMonth, now.Day()-offset, 0, 0, 0, 0, time.Local)
	endOfWeek := startOfWeek.AddDate(0, 0, 6)

	startOfMonth := time.Date(currentYear, currentMonth, 1, 0, 0, 0, 0, time.Local)
	daysInMonth := startOfMonth.AddDate(0, 1, -1).Day()
	endOfMonth := time.Date(currentYear, currentMonth, daysInMonth, 0, 0, 0, 0, time.Local)

	var holidays []time.Time
	r.db.Table("company_holidays").Select("holiday_date").Pluck("holiday_date", &holidays)
	holidayMap := make(map[string]bool)
	for _, h := range holidays {
		holidayMap[h.Format("2006-01-02")] = true
	}

	workingDaysInWeek := 0
	for d := startOfWeek; !d.After(endOfWeek); d = d.AddDate(0, 0, 1) {
		if d.After(today) {
			continue // ไม่นับอนาคต
		}
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday && !holidayMap[d.Format("2006-01-02")] {
			workingDaysInWeek++
		}
	}

	workingDaysInMonth := 0
	for d := startOfMonth; !d.After(endOfMonth); d = d.AddDate(0, 0, 1) {
		if d.After(today) {
			continue // ไม่นับอนาคต
		}
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday && !holidayMap[d.Format("2006-01-02")] {
			workingDaysInMonth++
		}
	}

	thaiDays := map[time.Weekday]string{
		time.Sunday: "อา.", time.Monday: "จ.", time.Tuesday: "อ.",
		time.Wednesday: "พ.", time.Thursday: "พฤ.", time.Friday: "ศ.", time.Saturday: "ส.",
	}
	thaiMonths := map[time.Month]string{
		time.January: "ม.ค.", time.February: "ก.พ.", time.March: "มี.ค.",
		time.April: "เม.ย.", time.May: "พ.ค.", time.June: "มิ.ย.",
		time.July: "ก.ค.", time.August: "ส.ค.", time.September: "ก.ย.",
		time.October: "ต.ค.", time.November: "พ.ย.", time.December: "ธ.ค.",
	}

	weekKeys := []string{"อา.", "จ.", "อ.", "พ.", "พฤ.", "ศ.", "ส."}
	weekMap := make(map[string]float64)
	for _, k := range weekKeys {
		weekMap[k] = 0.0
	}

	monthKeys := []string{}
	monthMap := make(map[string]float64)
	for i := 1; i <= daysInMonth; i++ {
		k := strconv.Itoa(i)
		monthKeys = append(monthKeys, k)
		monthMap[k] = 0.0
	}

	yearKeys := []string{"ม.ค.", "ก.พ.", "มี.ค.", "เม.ย.", "พ.ค.", "มิ.ย.", "ก.ค.", "ส.ค.", "ก.ย.", "ต.ค.", "พ.ย.", "ธ.ค."}
	yearMap := make(map[string]float64)
	for _, k := range yearKeys {
		yearMap[k] = 0.0
	}

	totalMap := make(map[string]float64)
	var totalWorkingHour, weeklyWorkingHour, monthlyWorkingHour, yearlyWorkingHour float64

	type AttRecord struct {
		Date     time.Time
		CheckIn  string `gorm:"column:check_in"`
		CheckOut string `gorm:"column:check_out"`
	}
	var atts []AttRecord
	r.db.Table("attendance").
		Select("date, check_in, check_out").
		Where("user_id = ? AND check_in IS NOT NULL AND check_out IS NOT NULL", userID).
		Scan(&atts)

	for _, att := range atts {
		if att.Date.Weekday() == time.Saturday || att.Date.Weekday() == time.Sunday || holidayMap[att.Date.Format("2006-01-02")] {
			continue
		}

		inTime, errIn := time.Parse("15:04:05", att.CheckIn)
		outTime, errOut := time.Parse("15:04:05", att.CheckOut)
		if errIn != nil || errOut != nil {
			continue
		}

		duration := outTime.Sub(inTime).Hours()
		if duration < 0 {
			duration += 24
		}
		duration = math.Round(duration*100) / 100

		totalWorkingHour += duration
		buddhistYear := strconv.Itoa(att.Date.Year() + 543)[2:]
		totalMap[buddhistYear] += duration

		if att.Date.Year() == currentYear {
			yearlyWorkingHour += duration
			yearMap[thaiMonths[att.Date.Month()]] += duration
		}

		if att.Date.Year() == currentYear && att.Date.Month() == currentMonth {
			monthlyWorkingHour += duration
			dayStr := strconv.Itoa(att.Date.Day())
			monthMap[dayStr] += duration
		}

		attDateOnly := time.Date(att.Date.Year(), att.Date.Month(), att.Date.Day(), 0, 0, 0, 0, time.Local)
		if !attDateOnly.Before(startOfWeek) && !attDateOnly.After(endOfWeek) {
			weeklyWorkingHour += duration
			weekMap[thaiDays[att.Date.Weekday()]] += duration
		}
	}

	weeklyAverageHour, monthlyAverageHour, yearlyAverageHour, totalAverageHour := 0.0, 0.0, 0.0, 0.0
	if workingDaysInWeek > 0 {
		weeklyAverageHour = math.Round((weeklyWorkingHour/float64(workingDaysInWeek))*100) / 100
	}
	if workingDaysInMonth > 0 {
		monthlyAverageHour = math.Round((monthlyWorkingHour/float64(workingDaysInMonth))*100) / 100
	}
	currentMonthNum := int(currentMonth)
	if currentMonthNum > 0 {
		yearlyAverageHour = math.Round((yearlyWorkingHour/float64(currentMonthNum))*100) / 100
	}
	if len(totalMap) > 0 {
		totalAverageHour = math.Round((totalWorkingHour/float64(len(totalMap)))*100) / 100
	}

	// 🌟 ท่าไม้ตายเรียง JSON
	orderedJSON := func(keys []string, m map[string]float64) json.RawMessage {
		str := "{"
		for i, k := range keys {
			if i > 0 {
				str += ","
			}
			str += fmt.Sprintf(`"%s":%v`, k, m[k])
		}
		str += "}"
		return json.RawMessage(str)
	}

	totalKeys := []string{}
	for k := range totalMap {
		totalKeys = append(totalKeys, k)
	}
	sort.Strings(totalKeys)

	return map[string]interface{}{
		"total-working-hour":   math.Round(totalWorkingHour*100) / 100,
		"total-average-hour":   totalAverageHour,
		"weekly-working-hour":  math.Round(weeklyWorkingHour*100) / 100,
		"weekly-average-hour":  weeklyAverageHour,
		"monthly-working-hour": math.Round(monthlyWorkingHour*100) / 100,
		"monthly-average-hour": monthlyAverageHour,
		"yearly-working-hour":  math.Round(yearlyWorkingHour*100) / 100,
		"yearly-average-hour":  yearlyAverageHour,
		"total":                orderedJSON(totalKeys, totalMap),
		"week":                 orderedJSON(weekKeys, weekMap),
		"month":                orderedJSON(monthKeys, monthMap),
		"year":                 orderedJSON(yearKeys, yearMap),
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