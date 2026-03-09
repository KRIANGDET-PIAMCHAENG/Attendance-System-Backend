package repository

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	
	"math"
	"strconv"
	"strings"

	"encoding/json"
	"sort"          // 👈 เพิ่มอันนี้

	
)

type PersonnelRepo struct {
	db *gorm.DB
}

func NewPersonnelRepo(db *gorm.DB) *PersonnelRepo {
	return &PersonnelRepo{db: db}
}

func (r *PersonnelRepo) checkPermission(managerID, targetUserID string) bool {
	var adminCount int64
	r.db.Table("user_roles ur").Joins("JOIN role r ON ur.role_id = r.role_id").
		Where("ur.user_id = ? AND r.role_type = ?", managerID, "admin").Count(&adminCount)
	if adminCount > 0 { return true }

	var subCount int64
	r.db.Table("subordinate_manager_roles smr").
		Joins("JOIN user_roles mr ON smr.manager_role_id = mr.role_id").
		Joins("JOIN role r_manager ON mr.role_id = r_manager.role_id").
		// 🌟 ใช้ smr.subordinate_id ให้ตรงกับ Database จริง
		Where("mr.user_id = ? AND smr.subordinate_id = ? AND r_manager.role_type = ?", managerID, targetUserID, "main").
		Count(&subCount)
	return subCount > 0
}
// 1. Get Pending (ใบลาที่รออนุมัติ และ "ยังไม่เลยกำหนดเวลา")
func (r *PersonnelRepo) GetPending(managerID, personnelID string) ([]map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) {
		return nil, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูลของพนักงานท่านนี้")
	}

	// 🌟 ประกาศเป็น Array ว่าง ป้องกันการพ่น null ออกไปให้ Frontend
	results := []map[string]interface{}{}
	type Result struct {
		ID        int
		LeaveType string    `gorm:"column:leave_type"`
		DateFrom  time.Time `gorm:"column:date_from"`
	}
	var rows []Result

	// 🌟 [แก้ตรงนี้] เพิ่มเงื่อนไข AND date_from >= ? (ดึงเฉพาะคิวที่ยังไม่ถึงวันลา)
	r.db.Table("leave_requests").
		Select("id, leave_type, date_from").
		Where("user_id = ? AND status = 'pending' AND date_from >= ?", personnelID, time.Now()).
		Scan(&rows)

	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"id":         fmt.Sprintf("LEV%012d", row.ID),
			"leave-type": row.LeaveType,
			"date-start": row.DateFrom.Format(time.RFC3339),
		})
	}
	return results, nil
}

// 2. Get Recent (ประวัติใบลา + ใบลาที่รออนุมัติแต่ "เลยกำหนดเวลาแล้ว")
func (r *PersonnelRepo) GetRecent(managerID, personnelID, startDate, endDate string) ([]map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) {
		return nil, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูลของพนักงานท่านนี้")
	}

	results := []map[string]interface{}{}
	now := time.Now()

	// 🌟 [แก้ตรงนี้] ดึง status ไม่เท่ากับ pending หรือ (เป็น pending แต่ date_from ผ่านมาแล้ว)
	query := r.db.Table("leave_requests").
		Select("id, leave_type, date_from, status").
		Where("user_id = ? AND (status != 'pending' OR (status = 'pending' AND date_from < ?))", personnelID, now)

	if startDate != "" && endDate != "" {
		query = query.Where("date_from >= ? AND date_from <= ?", startDate, endDate)
	}

	// 🌟 เรียงจากล่าสุดไปเก่าสุดให้ด้วย เผื่อ Frontend ไม่ได้เรียง
	query = query.Order("date_from DESC")

	type Result struct {
		ID        int
		LeaveType string    `gorm:"column:leave_type"`
		DateFrom  time.Time `gorm:"column:date_from"`
		Status    string    `gorm:"column:status"`
	}
	var rows []Result
	query.Scan(&rows)

	for _, row := range rows {
		finalStatus := row.Status

		// 🌟 แปลงร่าง Status! ถ้าหลุดเข้ามาด้วยโควต้า pending แปลว่ามันเลยเวลาแล้ว ให้เปลี่ยนเป็น overdue
		if finalStatus == "pending" && row.DateFrom.Before(now) {
			finalStatus = "overdue"
		}

		results = append(results, map[string]interface{}{
			"id":         fmt.Sprintf("LEV%012d", row.ID),
			"leave-type": row.LeaveType,
			"date-start": row.DateFrom.Format(time.RFC3339),
			"status":     finalStatus,
			"approved":   (finalStatus == "approved"),
		})
	}
	return results, nil
}

// 3. Get Filter Range
func (r *PersonnelRepo) GetFilterRange(managerID, personnelID string) (time.Time, time.Time, error) {
	if !r.checkPermission(managerID, personnelID) {
		return time.Time{}, time.Time{}, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูล")
	}

	var res struct {
		Start time.Time `gorm:"column:start_date"`
		End   time.Time `gorm:"column:end_date"`
	}
	err := r.db.Table("leave_requests").
		Select("MIN(date_from) as start_date, MAX(date_to) as end_date").
		Where("user_id = ?", personnelID).
		Scan(&res).Error
	return res.Start, res.End, err
}

// 4. Get Detail (ตัดตัวอักษร LEV มาจาก Handler แล้ว)
func (r *PersonnelRepo) GetDetail(managerID string, reqID int) (map[string]interface{}, error) {
	// ดึง UserID ของใบคำขอนี้ออกมาก่อน เพื่อเช็คสิทธิ์
	var req struct {
		UserID          string    `gorm:"column:user_id"`
		LeaveType       string    `gorm:"column:leave_type"`
		DateFrom        time.Time `gorm:"column:date_from"`
		DateTo          time.Time `gorm:"column:date_to"`
		FromDateMorning bool      `gorm:"column:from_date_morning"`
		ToDateMorning   bool      `gorm:"column:to_date_morning"`
		Remark          string    `gorm:"column:remark"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		Status          string    `gorm:"column:status"`
	}
	if err := r.db.Table("leave_requests").Where("id = ?", reqID).First(&req).Error; err != nil {
		return nil, errors.New("ไม่พบใบคำขอนี้")
	}

	// 🛡️ เช็คสิทธิ์
	if !r.checkPermission(managerID, req.UserID) {
		return nil, errors.New("unauthorized: คุณไม่มีสิทธิ์ดูรายละเอียดใบคำขอของพนักงานท่านนี้")
	}

	// 🌟 [แก้ตรงนี้] บังคับเป็น Array ว่าง เพื่อไม่ให้ Frontend เจอค่า null กรณีไม่มีไฟล์แนบ
	files := []map[string]interface{}{}
	r.db.Table("leave_attachments").Where("leave_request_id = ?", reqID).
		Select("original_name as \"file-name\", file_path as \"file-url\", file_type as \"file-type\", file_size as \"file-size\"").
		Find(&files)

	// วน Loop เติม Base URL ให้เป็น Link เต็ม
	baseURL := "http://20.194.9.179:3000/"
	for i := range files {
		if path, ok := files[i]["file-url"].(string); ok && path != "" {
			if len(path) < 4 || path[:4] != "http" {
				if path[0] == '/' {
					path = path[1:]
				}
				files[i]["file-url"] = baseURL + path
			}
		}
	}

	// (ดึงข้อมูลการอนุมัติ)
	var app struct {
		ApproverName string    `gorm:"column:approver_name"`
		ApproveRole  string    `gorm:"column:approve_role"`
		Reason       string    `gorm:"column:reason"`
		CreatedAt    time.Time `gorm:"column:created_at"`
	}
	r.db.Table("leave_approvals").Where("leave_request_id = ?", reqID).First(&app)

	// ถ้ายังไม่มีคนอนุมัติ ให้ไปหาว่า "ตำแหน่งอะไร" ที่ต้องเป็นคนกด Approve (role_type = 'main')
	if app.ApproveRole == "" {
		var expectedRoleName string
		r.db.Table("subordinate_manager_roles smr").
			Select("r.role_name").
			Joins("JOIN role r ON smr.manager_role_id = r.role_id").
			Where("smr.subordinate_id = ? AND r.role_type = ?", req.UserID, "main").
			Limit(1).
			Scan(&expectedRoleName)

		app.ApproveRole = expectedRoleName
	}

	// 🌟 [NEW LOGIC] ตรรกะเช็คสถานะ overdue
	finalStatus := req.Status
	if finalStatus == "pending" && req.DateFrom.Before(time.Now()) {
		finalStatus = "overdue" // เปลี่ยนสถานะเป็นเลยกำหนด ถ้าถึงวันลาแล้วยังไม่มีใครอนุมัติ
	}

	// 🌟 ดักค่า 0001-01-01 ถ้ายังไม่มีประวัติการอนุมัติ ให้ส่งกลับเป็น null
	var approveDateStr interface{} = nil
	if !app.CreatedAt.IsZero() {
		approveDateStr = app.CreatedAt.Format(time.RFC3339)
	}

	// ประกอบร่าง JSON ส่งกลับ
	return map[string]interface{}{
		"request-detail": map[string]interface{}{
			"leave-type":        req.LeaveType,
			"date-from":         req.DateFrom.Format(time.RFC3339),
			"date-to":           req.DateTo.Format(time.RFC3339),
			"from-date-morning": req.FromDateMorning,
			"to-date-morning":   req.ToDateMorning,
			"remark":            req.Remark,
			"evidence-files":    files,
			"request-date":      req.CreatedAt.Format(time.RFC3339),
		},
		"approve-detail": map[string]interface{}{
			"status":       finalStatus,     // 👈 ยัดค่า overdue หรือสถานะจริงกลับไป
			"approve-role": app.ApproveRole,
			"approver":     app.ApproverName,
			"reason":       app.Reason,
			"approve-date": approveDateStr,  // 👈 จะกลายเป็น null ใน JSON ถ้ายังไม่มีคนอนุมัติ
		},
	}, nil
}

// 5. Get Users (หัวหน้าเห็นเฉพาะลูกน้อง หรือ Admin เห็นทุกคน)
func (r *PersonnelRepo) GetUsers(managerID string) ([]map[string]interface{}, error) {
    // เช็คว่าเป็น Admin หรือเปล่า
    var adminCount int64
    r.db.Table("user_roles ur").Joins("JOIN role r ON ur.role_id = r.role_id").
        Where("ur.user_id = ? AND r.role_type = ?", managerID, "admin").Count(&adminCount)
    isAdmin := adminCount > 0

    query := r.db.Table("user_info ui").
        Select("ui.user_id, ui.fullname_thai, ui.fullname_eng, ui.picture, ui.role_init, r.role_id, r.role_name, r.role_color").
        Joins("LEFT JOIN user_roles ur ON ui.user_id = ur.user_id").
        Joins("LEFT JOIN role r ON ur.role_id = r.role_id")

    // 🛡️ ถ้าไม่ใช่ Admin ให้ Filter เอามาแค่ "ลูกน้อง"
    if !isAdmin {
        // 🌟 [แก้ตรงนี้] ดึง subordinate_id ตรงๆ จากตาราง subordinate_manager_roles ไม่ต้องไป JOIN user_roles ฝั่งลูกน้องแล้ว
        subQuery := r.db.Table("subordinate_manager_roles smr").
            Select("smr.subordinate_id").
            Joins("JOIN user_roles mr ON smr.manager_role_id = mr.role_id").
            Where("mr.user_id = ?", managerID)
        
        query = query.Where("ui.user_id IN (?)", subQuery)
    }

    type UserRow struct {
        ID string `gorm:"column:user_id"`; NameTh string `gorm:"column:fullname_thai"`; NameEn string `gorm:"column:fullname_eng"`
        Pic string `gorm:"column:picture"`; InitRole string `gorm:"column:role_init"`; RoleID string; RoleName string; RoleColor string
    }
    var rows []UserRow
    query.Scan(&rows)

    // แปลงข้อมูลเป็น JSON Format
    userMap := make(map[string]map[string]interface{})
    for _, row := range rows {
        if _, exists := userMap[row.ID]; !exists {
            userMap[row.ID] = map[string]interface{}{
                "id": row.ID, "name-th": row.NameTh, "name-en": row.NameEn,
                "avatar-url": row.Pic, "initial-role": row.InitRole, "roles": []map[string]string{},
            }
        }
        if row.RoleID != "" {
            roles := userMap[row.ID]["roles"].([]map[string]string)
            roles = append(roles, map[string]string{
                "role-id": row.RoleID, "role-name": row.RoleName, "role-color": row.RoleColor,
            })
            userMap[row.ID]["roles"] = roles
        }
    }

    var results []map[string]interface{}
    for _, v := range userMap {
        results = append(results, v)
    }
    return results, nil
}


func (r *PersonnelRepo) CheckApprovalPermission(requesterID string, targetID string) (int, error) {
	var adminCount int64
	r.db.Table("user_roles ur").Joins("JOIN role r ON ur.role_id = r.role_id").
		Where("ur.user_id = ? AND r.role_type = ?", requesterID, "admin").Count(&adminCount)
	if adminCount > 0 { return 1, nil }

	var managerCount int64
	r.db.Table("subordinate_manager_roles smr").
		Joins("JOIN user_roles mr ON smr.manager_role_id = mr.role_id"). 
		Joins("JOIN role r_manager ON mr.role_id = r_manager.role_id"). 
		Where("mr.user_id = ? AND smr.subordinate_id = ? AND r_manager.role_type = ?", requesterID, targetID, "main").
		Count(&managerCount)

	if managerCount > 0 { return 1, nil }
	return 0, nil
}


// 7. Get Personnel Data (ดึงข้อมูลส่วนตัวพนักงานตาม ID)
func (r *PersonnelRepo) GetPersonnelData(managerID, personnelID string) (map[string]interface{}, error) {
	// 1. เช็คสิทธิ์ก่อนว่า Manager มีสิทธิ์ดูคนนี้ไหม? (เหมือนเส้นอื่นๆ)
	if !r.checkPermission(managerID, personnelID) {
		return nil, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูลของพนักงานท่านนี้")
	}

	// 2. ดึงข้อมูลพื้นฐานจากตาราง user_info
	var user struct {
		UserID       string `gorm:"column:user_id"`
		EmployeeID   string `gorm:"column:employee_id"`
		Email        string `gorm:"column:email"`
		FullnameEng  string `gorm:"column:fullname_eng"`
		FullnameThai string `gorm:"column:fullname_thai"`
		Gender       string `gorm:"column:gender"`
		Nationality  string `gorm:"column:nationality"`
		Phone        string `gorm:"column:phone"`
		Picture      string `gorm:"column:picture"`
		RoleInit     string `gorm:"column:role_init"`
	}

	if err := r.db.Table("user_info").Where("user_id = ?", personnelID).First(&user).Error; err != nil {
		return nil, errors.New("ไม่พบข้อมูลพนักงานในระบบ")
	}

	// 3. ดึงข้อมูล Roles (ตำแหน่งการทำงาน)
	type RoleData struct {
		RoleName  string `gorm:"column:role_name"`
		RoleColor string `gorm:"column:role_color"`
	}
	var roleRows []RoleData
	r.db.Table("user_roles ur").
		Select("r.role_name, r.role_color").
		Joins("JOIN role r ON ur.role_id = r.role_id").
		Where("ur.user_id = ?", personnelID).
		Scan(&roleRows)

	// แปลงผลลัพธ์ใส่เข้าไปใน Array ของ Roles
	var roles []map[string]string
	for _, rr := range roleRows {
		roles = append(roles, map[string]string{
			"role-name":  rr.RoleName,
			"role-color": rr.RoleColor,
		})
	}

	// เพิ่ม role_init เข้าไปใน List ด้วย (ถ้ามีข้อมูล) พร้อมกำหนดสีเทาเข้ม
	if user.RoleInit != "" {
		roles = append(roles, map[string]string{
			"role-name":  user.RoleInit,
			"role-color": "535353",
		})
	}

	// 4. ประกอบร่างเป็น JSON ตาม Format ที่ Frontend ต้องการเป๊ะๆ
	return map[string]interface{}{
		"user_id":       user.UserID,
		"employee_id":   user.EmployeeID,
		"email":         user.Email,
		"fullname_eng":  user.FullnameEng,
		"fullname_thai": user.FullnameThai,
		"gender":        user.Gender,
		"nationality":   user.Nationality,
		"phone":         user.Phone,
		"roles":         roles,
		"picture":       user.Picture,
	}, nil
}

// ==========================================
// 🌟 หมวด Statistic & Working Hours
// ==========================================

// ==========================================
// 🌟 หมวด Statistic & Working Hours
// ==========================================

// ประกาศ Struct เพื่อ "บังคับลำดับ" การแสดงผล JSON
type WeekStat struct {
	Mon float64 `json:"จ."`
	Tue float64 `json:"อ."`
	Wed float64 `json:"พ."`
	Thu float64 `json:"พฤ."`
	Fri float64 `json:"ศ."`
	Sat float64 `json:"ส."`
	Sun float64 `json:"อา."`
}

type MonthStat struct {
	D1 float64 `json:"1"`; D2 float64 `json:"2"`; D3 float64 `json:"3"`; D4 float64 `json:"4"`; D5 float64 `json:"5"`
	D6 float64 `json:"6"`; D7 float64 `json:"7"`; D8 float64 `json:"8"`; D9 float64 `json:"9"`; D10 float64 `json:"10"`
	D11 float64 `json:"11"`; D12 float64 `json:"12"`; D13 float64 `json:"13"`; D14 float64 `json:"14"`; D15 float64 `json:"15"`
	D16 float64 `json:"16"`; D17 float64 `json:"17"`; D18 float64 `json:"18"`; D19 float64 `json:"19"`; D20 float64 `json:"20"`
	D21 float64 `json:"21"`; D22 float64 `json:"22"`; D23 float64 `json:"23"`; D24 float64 `json:"24"`; D25 float64 `json:"25"`
	D26 float64 `json:"26"`; D27 float64 `json:"27"`; D28 float64 `json:"28"`; D29 float64 `json:"29"`; D30 float64 `json:"30"`; D31 float64 `json:"31"`
}

type YearStat struct {
	Jan float64 `json:"ม.ค."`
	Feb float64 `json:"ก.พ."`
	Mar float64 `json:"มี.ค."`
	Apr float64 `json:"เม.ย."`
	May float64 `json:"พ.ค."`
	Jun float64 `json:"มิ.ย."`
	Jul float64 `json:"ก.ค."`
	Aug float64 `json:"ส.ค."`
	Sep float64 `json:"ก.ย."`
	Oct float64 `json:"ต.ค."`
	Nov float64 `json:"พ.ย."`
	Dec float64 `json:"ธ.ค."`
}

// 🌟 [NEW] เพิ่มฟังก์ชันนี้ต่อท้าย Struct YearStat ทันที
// เพื่อบังคับให้ Go พ่น JSON ออกมาเป็นภาษาไทยและเรียงลำดับเป๊ะๆ (ทะลวงบั๊กสระอี/สระอิ)
func (y YearStat) MarshalJSON() ([]byte, error) {
	str := fmt.Sprintf(`{"ม.ค.":%v,"ก.พ.":%v,"มี.ค.":%v,"เม.ย.":%v,"พ.ค.":%v,"มิ.ย.":%v,"ก.ค.":%v,"ส.ค.":%v,"ก.ย.":%v,"ต.ค.":%v,"พ.ย.":%v,"ธ.ค.":%v}`,
		y.Jan, y.Feb, y.Mar, y.Apr, y.May, y.Jun, y.Jul, y.Aug, y.Sep, y.Oct, y.Nov, y.Dec)
	return []byte(str), nil
}

func (r *PersonnelRepo) GetWorkingHoursStatistic(managerID, personnelID string) (map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) {
		return nil, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูลของพนักงานท่านนี้")
	}

	now := time.Now()
	currentYear := now.Year()
	currentMonth := now.Month()

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
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday && !holidayMap[d.Format("2006-01-02")] {
			workingDaysInWeek++
		}
	}

	workingDaysInMonth := 0
	for d := startOfMonth; !d.After(endOfMonth); d = d.AddDate(0, 0, 1) {
		if d.Weekday() != time.Saturday && d.Weekday() != time.Sunday && !holidayMap[d.Format("2006-01-02")] {
			workingDaysInMonth++
		}
	}

	// 🌟 1. ประกาศลำดับ (Keys) ที่ต้องการให้แสดงใน JSON เป๊ะๆ
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

	// 🌟 2. ดึงข้อมูลและคำนวณตามเดิม
	type AttRecord struct {
		Date     time.Time
		CheckIn  string `gorm:"column:check_in"`
		CheckOut string `gorm:"column:check_out"`
	}
	var atts []AttRecord
	r.db.Table("attendance").
		Select("date, check_in, check_out").
		Where("user_id = ? AND check_in IS NOT NULL AND check_out IS NOT NULL", personnelID).
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
	yearlyAverageHour = math.Round((yearlyWorkingHour/12)*100) / 100
	if len(totalMap) > 0 {
		totalAverageHour = math.Round((totalWorkingHour/float64(len(totalMap)))*100) / 100
	}

	// 🌟 3. ท่าไม้ตาย! ฟังก์ชันจับเรียง JSON ให้ลำดับเป๊ะๆ
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

	// เรียงลำดับปีของ total (เช่น 66, 67, 68) ให้จากน้อยไปมาก
	totalKeys := []string{}
	for k := range totalMap {
		totalKeys = append(totalKeys, k)
	}
	sort.Strings(totalKeys)

	// 🌟 4. ประกอบร่าง JSON คืนค่า
	return map[string]interface{}{
		"total-working-hour":   math.Round(totalWorkingHour*100) / 100,
		"total-average-hour":   totalAverageHour,
		"weekly-working-hour":  math.Round(weeklyWorkingHour*100) / 100,
		"weekly-average-hour":  weeklyAverageHour,
		"monthly-working-hour": math.Round(monthlyWorkingHour*100) / 100,
		"monthly-average-hour": monthlyAverageHour,
		"yearly-working-hour":  math.Round(yearlyWorkingHour*100) / 100,
		"yearly-average-hour":  yearlyAverageHour,
		"total":                orderedJSON(totalKeys, totalMap), // 👈 ใช้ท่าไม้ตายตรงนี้
		"week":                 orderedJSON(weekKeys, weekMap),   // 👈 ใช้ท่าไม้ตายตรงนี้
		"month":                orderedJSON(monthKeys, monthMap), // 👈 ใช้ท่าไม้ตายตรงนี้
		"year":                 orderedJSON(yearKeys, yearMap),   // 👈 ใช้ท่าไม้ตายตรงนี้
	}, nil
}

// 10. Get Filter Range (ใช้ร่วมกันได้ทั้ง สถิติ และ ประวัติเข้างาน)
func (r *PersonnelRepo) GetManagerStatFilterRange(managerID, personnelID string) (map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) { return nil, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูล") }
	var bounds struct { MinDate *time.Time `gorm:"column:min_date"`; MaxDate *time.Time `gorm:"column:max_date"` }
	r.db.Table("attendance").Select("MIN(date) as min_date, MAX(date) as max_date").Where("user_id = ?", personnelID).Scan(&bounds)
	
	start, end := time.Now(), time.Now()
	if bounds.MinDate != nil { start = *bounds.MinDate }
	if bounds.MaxDate != nil { end = *bounds.MaxDate }
	return map[string]interface{}{ "start": start.Format("2006-01-02T15:04:05.000Z"), "end": end.Format("2006-01-02T15:04:05.000Z") }, nil
}

// ==========================================
// 🌟 หมวด คำขอแก้ไขเวลาเข้างาน (Attendance Requests)
// ==========================================

// 11. Get Pending Attendance Requests
func (r *PersonnelRepo) GetAttReqPending(managerID, personnelID string) ([]map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) { return nil, errors.New("unauthorized") }
	type Result struct { ID int; DateFrom time.Time `gorm:"column:date_from"`; DateTo time.Time `gorm:"column:date_to"` }
	var rows []Result
	r.db.Table("attendance_requests").Select("id, date_from, date_to").Where("user_id = ? AND status = 'pending'", personnelID).Scan(&rows)
	
	var results []map[string]interface{}
	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"id": fmt.Sprintf("REQ%012d", row.ID), "date-start": row.DateFrom.Format(time.RFC3339), "date-end": row.DateTo.Format(time.RFC3339),
		})
	}
	if len(results) == 0 { return []map[string]interface{}{}, nil }
	return results, nil
}

// 12. Get Recent Attendance Requests
func (r *PersonnelRepo) GetAttReqRecent(managerID, personnelID, startDate, endDate string) ([]map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) { return nil, errors.New("unauthorized") }
	query := r.db.Table("attendance_requests").Select("id, date_from, date_to, status").Where("user_id = ? AND status != 'pending'", personnelID)
	if startDate != "" && endDate != "" { query = query.Where("date_from >= ? AND date_from <= ?", startDate, endDate) }
	
	type Result struct { ID int; DateFrom time.Time; DateTo time.Time; Status string }
	var rows []Result
	query.Scan(&rows)
	
	var results []map[string]interface{}
	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"id": fmt.Sprintf("REQ%012d", row.ID), "date-start": row.DateFrom.Format(time.RFC3339),
			"date-end": row.DateTo.Format(time.RFC3339), "status": row.Status,
		})
	}
	if len(results) == 0 { return []map[string]interface{}{}, nil }
	return results, nil
}

// 13. Get Attendance Request Filter Range
func (r *PersonnelRepo) GetAttReqFilterRange(managerID, personnelID string) (map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) { return nil, errors.New("unauthorized") }
	var res struct { Start *time.Time `gorm:"column:start_date"`; End *time.Time `gorm:"column:end_date"` }
	r.db.Table("attendance_requests").Select("MIN(date_from) as start_date, MAX(date_to) as end_date").Where("user_id = ?", personnelID).Scan(&res)
	start, end := time.Now(), time.Now()
	if res.Start != nil { start = *res.Start }; if res.End != nil { end = *res.End }
	return map[string]interface{}{ "start": start.Format("2006-01-02T15:04:05.000Z"), "end": end.Format("2006-01-02T15:04:05.000Z") }, nil
}

// 14. Get Attendance Request Detail (ตัด REQ มาจาก Handler แล้ว)
func (r *PersonnelRepo) GetAttReqDetail(managerID string, reqID int) (map[string]interface{}, error) {
	var req struct {
		UserID string; DateFrom time.Time; DateTo time.Time; StartTime string; EndTime string; Remark string; Status string; CreatedAt time.Time
	}
	if err := r.db.Table("attendance_requests").Where("id = ?", reqID).First(&req).Error; err != nil { return nil, errors.New("ไม่พบใบคำขอนี้") }
	if !r.checkPermission(managerID, req.UserID) { return nil, errors.New("unauthorized") }

	files := []map[string]interface{}{}
	r.db.Table("attendance_request_attachments").Where("attendance_request_id = ?", reqID).
		Select("original_name as \"file-name\", file_path as \"file-url\", file_type as \"file-type\", file_size as \"file-size\"").Find(&files)
	baseURL := "http://20.194.9.179:3000/" 
	for i := range files {
		if path, ok := files[i]["file-url"].(string); ok && path != "" && !strings.HasPrefix(path, "http") {
			if path[0] == '/' { path = path[1:] }; files[i]["file-url"] = baseURL + path
		}
	}

	var app struct { ApproverName string; ApproveRole string; Reason string; CreatedAt time.Time }
	r.db.Table("attendance_approvals").Where("attendance_request_id = ?", reqID).First(&app)

	if app.ApproveRole == "" { // หาคนอนุมัติล่วงหน้าถ้ายัง Pending
		r.db.Table("subordinate_manager_roles smr").Select("r.role_name").Joins("JOIN role r ON smr.manager_role_id = r.role_id").
			Where("smr.subordinate_id = ? AND r.role_type = ?", req.UserID, "main").Limit(1).Scan(&app.ApproveRole)
	}

	return map[string]interface{}{
		"request-detail": map[string]interface{}{
			"date-from": req.DateFrom.Format(time.RFC3339), "date-to": req.DateTo.Format(time.RFC3339),
			"time-start": req.StartTime[:5], "time-end": req.EndTime[:5], "remark": req.Remark, "evidence-files": files,
		},
		"approve-detail": map[string]interface{}{
			"status": req.Status, "approve-role": app.ApproveRole, "approver": app.ApproverName,
			"reason": app.Reason, "approve-date": app.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// ==========================================
// 🌟 หมวด ประวัติการเข้างาน (Attendance History)
// ==========================================

// 15. Get Attendance History
func (r *PersonnelRepo) GetAttendanceHistory(managerID, personnelID, startDate, endDate string) ([]map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) { return nil, errors.New("unauthorized") }
	
	type AttRec struct { Date time.Time; CheckIn *string `gorm:"column:check_in"`; CheckOut *string `gorm:"column:check_out"` }
	var records []AttRec
	query := r.db.Table("attendance").Select("date, check_in::text, check_out::text").Where("user_id = ?", personnelID)
	if startDate != "" && endDate != "" { query = query.Where("date >= ? AND date <= ?", startDate, endDate) }
	query.Order("date DESC").Scan(&records)

	// ดึงข้อมูลการลาที่อนุมัติแล้วมาเพื่อเช็คสถานะ leavePeriod
	type LeaveReq struct { DateFrom time.Time; DateTo time.Time; FromMorn bool `gorm:"column:from_date_morning"`; ToMorn bool `gorm:"column:to_date_morning"` }
	var leaves []LeaveReq
	leaveQ := r.db.Table("leave_requests").Where("user_id = ? AND status = 'approved'", personnelID)
	if startDate != "" && endDate != "" { leaveQ = leaveQ.Where("date_from <= ? AND date_to >= ?", endDate, startDate) }
	leaveQ.Scan(&leaves)

	thaiDOW := []string{"อาทิตย์", "จันทร์", "อังคาร", "พุธ", "พฤหัสบดี", "ศุกร์", "เสาร์"}
	var results []map[string]interface{}

	for _, rec := range records {
		cIn, cOut := interface{}(nil), interface{}(nil)
		if rec.CheckIn != nil { cIn = (*rec.CheckIn)[:5] }
		if rec.CheckOut != nil { cOut = (*rec.CheckOut)[:5] }

		leavePeriod := "NONE"
		for _, lv := range leaves {
			if (rec.Date.Equal(lv.DateFrom) || rec.Date.After(lv.DateFrom)) && (rec.Date.Equal(lv.DateTo) || rec.Date.Before(lv.DateTo)) {
				isStart := rec.Date.Equal(lv.DateFrom)
				isEnd := rec.Date.Equal(lv.DateTo)
				if isStart && isEnd {
					if lv.FromMorn && !lv.ToMorn { leavePeriod = "FULL_DAY" } else if lv.FromMorn && lv.ToMorn { leavePeriod = "MORNING" } else if !lv.FromMorn && !lv.ToMorn { leavePeriod = "AFTERNOON" } else { leavePeriod = "FULL_DAY" }
				} else if isStart {
					if lv.FromMorn { leavePeriod = "FULL_DAY" } else { leavePeriod = "AFTERNOON" }
				} else if isEnd {
					if !lv.ToMorn { leavePeriod = "FULL_DAY" } else { leavePeriod = "MORNING" }
				} else { leavePeriod = "FULL_DAY" }
				break
			}
		}

		results = append(results, map[string]interface{}{
			"date": rec.Date.Format("2006-01-02"), "dow": thaiDOW[int(rec.Date.Weekday())],
			"checkIn": cIn, "checkOut": cOut, "leavePeriod": leavePeriod,
		})
	}
	if len(results) == 0 { return []map[string]interface{}{}, nil }
	return results, nil
}

func (r *PersonnelRepo) GetPersonnelStatistic(managerID, personnelID string, targetYear int) (map[string]interface{}, error) {
	// 1. 🛡️ เช็คสิทธิ์ (ถ้าลูกพี่มีฟังก์ชันนี้แล้ว)
	if !r.checkPermission(managerID, personnelID) {
		return nil, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูลของพนักงานท่านนี้")
	}

	// 2. 📅 โหลด Config ปีงบประมาณ (Budget Year)
	var byConfig struct {
		Month int `json:"month"`
		Day   int `json:"day"`
	}
	byConfig.Month = 1 // Default 1 ม.ค.
	byConfig.Day = 1

	var rawBY string
	r.db.Table("system_configs").Where("config_key = 'budget_year'").Select("config_value").Scan(&rawBY)
	if rawBY != "" {
		json.Unmarshal([]byte(rawBY), &byConfig)
	}

	// 🎯 คำนวณวันเริ่มต้น และ วันสิ้นสุด ปีงบประมาณ
	startYear := targetYear
	if byConfig.Month > 6 {
		startYear = targetYear - 1 // ถ้าเริ่ม ต.ค. ปีงบ 2026 ต้องเริ่ม 1 ต.ค. 2025
	}
	startDate := time.Date(startYear, time.Month(byConfig.Month), byConfig.Day, 0, 0, 0, 0, time.Local)
	endDate := startDate.AddDate(1, 0, 0).AddDate(0, 0, -1)

	// 3. ⏱️ โหลด Config เวลาเข้างาน
	var attConfig struct {
		CheckInTime struct {
			Hour   int `json:"hour"`
			Minute int `json:"minute"`
		} `json:"check-in-time"`
	}
	attConfig.CheckInTime.Hour = 8 // Default 08:30
	attConfig.CheckInTime.Minute = 30

	var rawAtt string
	r.db.Table("system_configs").Where("config_key = 'attendance_time'").Select("config_value").Scan(&rawAtt)
	if rawAtt != "" {
		json.Unmarshal([]byte(rawAtt), &attConfig)
	}
	checkInLimit, _ := time.Parse("15:04:05", fmt.Sprintf("%02d:%02d:00", attConfig.CheckInTime.Hour, attConfig.CheckInTime.Minute))

	// 4. 🏖️ โหลดวันหยุดบริษัท/ราชการ
	var holidays []time.Time
	r.db.Table("company_holidays").
		Select("holiday_date").
		Where("holiday_date >= ? AND holiday_date <= ?", startDate, endDate).
		Pluck("holiday_date", &holidays)

	holidayMap := make(map[string]bool)
	for _, h := range holidays {
		holidayMap[h.Format("2006-01-02")] = true
	}

	// 5. 💼 คำนวณวันทำงานทั้งหมด (ตัดเสาร์-อาทิตย์ และวันหยุดออก)
	totalWorkDays := 0
	workDayMap := make(map[string]bool)
	now := time.Now()

	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue // ตัดเสาร์อาทิตย์
		}
		dateStr := d.Format("2006-01-02")
		if holidayMap[dateStr] {
			continue // ตัดวันหยุดราชการ
		}
		totalWorkDays++

		// มาร์กไว้ว่าเป็น "วันทำงานในอดีตจนถึงปัจจุบัน" (เพื่อใช้คำนวณการขาดงาน/มาสาย)
		if !d.After(now) {
			workDayMap[dateStr] = true
		}
	}

	// 6. 🤒 โหลดข้อมูลการลา (ที่อนุมัติแล้วเท่านั้น)
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
		Where("lr.user_id = ? AND lr.status = 'approved' AND lr.date_from <= ? AND lr.date_to >= ?", personnelID, endDate, startDate).
		Scan(&leaves)

	leaveUsedPerType := make(map[string]float64)
	leaveDaysMap := make(map[string]float64)

	// ลูประบุว่าวันไหนลาไปบ้าง (คำนวณครึ่งวัน/เต็มวัน)
	for _, l := range leaves {
		for d := l.DateFrom; !d.After(l.DateTo); d = d.AddDate(0, 0, 1) {
			if d.Before(startDate) || d.After(endDate) {
				continue // ตัดวันที่อยู่นอกปีงบประมาณ
			}
			dateStr := d.Format("2006-01-02")
			if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday || holidayMap[dateStr] {
				continue // ลาคร่อมวันหยุด ไม่นับวันนั้น
			}

			dayVal := 1.0
			if d.Format("2006-01-02") == l.DateFrom.Format("2006-01-02") && !l.FromDateMorning {
				dayVal -= 0.5 // เริ่มลาตอนบ่าย หักออกครึ่งวัน
			}
			if d.Format("2006-01-02") == l.DateTo.Format("2006-01-02") && l.ToDateMorning {
				dayVal -= 0.5 // สิ้นสุดลาตอนเช้า หักออกครึ่งวัน
			}

			leaveUsedPerType[l.LeaveType] += dayVal

			if !d.After(now) {
				leaveDaysMap[dateStr] += dayVal
			}
		}
	}

	// 7. ⏰ โหลดข้อมูลการสแกนเข้างาน
	type AttRecord struct {
		Date    time.Time
		CheckIn string `gorm:"column:check_in"`
	}
	var atts []AttRecord
	r.db.Table("attendance").
		Select("date, check_in").
		Where("user_id = ? AND date >= ? AND date <= ?", personnelID, startDate, now).
		Scan(&atts)

	attMap := make(map[string]bool)
	onTimeCount := 0
	lateCount := 0

	for _, a := range atts {
		dateStr := a.Date.Format("2006-01-02")
		if !workDayMap[dateStr] {
			continue // สแกนในวันหยุด ไม่นับ
		}
		attMap[dateStr] = true

		if a.CheckIn != "" {
			t, err := time.Parse("15:04:05", a.CheckIn)
			if err == nil {
				if t.After(checkInLimit) {
					lateCount++
				} else {
					onTimeCount++
				}
			}
		}
	}

	// 8. ❌ คำนวณวันขาดงาน (Absent)
	absentCount := 0
	for dateStr, isWorkDay := range workDayMap {
		if !isWorkDay {
			continue
		}
		if attMap[dateStr] {
			continue // สแกนเข้างานแล้ว ไม่ขาด
		}
		if leaveDaysMap[dateStr] >= 1.0 {
			continue // ลาเต็มวันแล้ว ไม่ขาด
		}

		// 💡 ดักบั๊ก UX: ถ้าเป็น "วันนี้" และยังไม่ถึงเที่ยง อย่าเพิ่งนับว่าขาด 
		// (อาจจะแค่สายหนักมาก หรือระบบสแกนดีเลย์)
		if dateStr == now.Format("2006-01-02") && now.Hour() < 12 {
			continue
		}

		absentCount++
	}

	// 9. 📊 โหลดโควต้าวันลา
	type Balance struct {
		LeaveType   string  `gorm:"column:name_en"`
		DaysAllowed float64 `gorm:"column:days_allowed"`
	}
	var balances []Balance
	r.db.Table("leave_balances lb").
		Select("lt.name_en, lb.days_allowed").
		Joins("JOIN leave_types lt ON lb.leave_type_id = lt.id").
		// บางปีค่า year อาจเป็น NULL (จาก db เก่า) เลยดักไว้ให้ด้วยครับ
		Where("lb.user_id = ? AND (lb.year = ? OR lb.year IS NULL)", personnelID, targetYear).
		Scan(&balances)

	quotaMap := make(map[string]float64)
	for _, b := range balances {
		quotaMap[b.LeaveType] = b.DaysAllowed
	}

	// 10. 🧩 ประกอบร่าง JSON ข้อมูล Leave ให้ตรงตาม Format
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

	// 11. 🚀 คืนค่ากลับให้ Handler
	return map[string]interface{}{
		"total_work_days":  totalWorkDays,
		// ลบวันลาที่ใช้ไปทั้งหมด ออกจากวันที่ต้องมาทำงาน = "วันที่ต้องทำงานจริง"
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