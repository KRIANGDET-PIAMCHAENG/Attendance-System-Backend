package repository

import (
	"fmt"
	"time"
	"gorm.io/gorm"
)

type PersonnelRepo struct {
	db *gorm.DB
}

func NewPersonnelRepo(db *gorm.DB) *PersonnelRepo {
	return &PersonnelRepo{db: db}
}

// 1. Get Pending
func (r *PersonnelRepo) GetPending(personnelID string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	type Result struct {
		ID        int
		LeaveType string
		DateFrom  time.Time
	}
	var rows []Result

	err := r.db.Table("leave_requests").
		Select("id, leave_type, date_from").
		Where("user_id = ? AND status = 'pending'", personnelID).
		Scan(&rows).Error

	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"id":         fmt.Sprintf("LEV%012d", row.ID), // เติมเลขให้ครบ 12 หลัก
			"leave-type": row.LeaveType,
			"date-start": row.DateFrom.Format(time.RFC3339),
		})
	}
	return results, err
}

// 2. Get Recent
func (r *PersonnelRepo) GetRecent(personnelID string, startDate, endDate string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	query := r.db.Table("leave_requests").
		Select("id, leave_type, date_from, status").
		Where("user_id = ? AND status != 'pending'", personnelID)

	if startDate != "" && endDate != "" {
		query = query.Where("date_from >= ? AND date_from <= ?", startDate, endDate)
	}

	type Result struct {
		ID        int
		LeaveType string
		DateFrom  time.Time
		Status    string
	}
	var rows []Result
	err := query.Scan(&rows).Error

	for _, row := range rows {
		approved := false
		if row.Status == "approved" {
			approved = true
		}
		
		results = append(results, map[string]interface{}{
			"id":         fmt.Sprintf("LEV%012d", row.ID),
			"leave-type": row.LeaveType,
			"date-start": row.DateFrom.Format(time.RFC3339),
			"status":     row.Status,
			"approved":   approved, // ส่งไปทั้ง status และ approved กันเหนียว
		})
	}
	return results, err
}

// 3. Get Filter Range
func (r *PersonnelRepo) GetFilterRange(personnelID string) (time.Time, time.Time, error) {
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

// 4. Get Detail
func (r *PersonnelRepo) GetDetail(reqID int) (map[string]interface{}, error) {
	// 4.1 ดึงข้อมูลหลัก
	var req struct {
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
		return nil, err
	}

	// 4.2 ดึงไฟล์แนบ
	type Attachment struct {
		OriginalName string `gorm:"column:original_name"`
		FilePath     string `gorm:"column:file_path"`
		FileType     string `gorm:"column:file_type"`
		FileSize     int64  `gorm:"column:file_size"`
	}
	var atts []Attachment
	r.db.Table("leave_attachments").Where("leave_request_id = ?", reqID).Find(&atts)

	var files []map[string]interface{}
	for _, att := range atts {
		// ถ้า Type เป็น null จาก DB ให้ดึงจากนามสกุลไฟล์แทน
		fileType := att.FileType
		if fileType == "" {
			fileType = "unknown"
		}
		files = append(files, map[string]interface{}{
			"file-name": att.OriginalName,
			"file-url":  att.FilePath, // ถ้ามี base url เอามาต่อตรงนี้ได้เลยครับ
			"file-type": fileType,
			"file-size": att.FileSize,
		})
	}

	// 4.3 ดึงข้อมูลการอนุมัติ (ถ้ามี)
	var approval struct {
		ApproverName string    `gorm:"column:approver_name"`
		ApproveRole  string    `gorm:"column:approve_role"`
		Status       string    `gorm:"column:status"`
		Reason       string    `gorm:"column:reason"`
		CreatedAt    time.Time `gorm:"column:created_at"`
	}
	r.db.Table("leave_approvals").Where("leave_request_id = ?", reqID).First(&approval)

	// ประกอบร่าง
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
			"status":       req.Status, // ดึงสถานะปัจจุบันจากใบคำขอ
			"approve-role": approval.ApproveRole,
			"approver":     approval.ApproverName,
			"reason":       approval.Reason,
			"approve-date": approval.CreatedAt.Format(time.RFC3339),
		},
	}, nil
}

// 5. Get Users (ดึงรายชื่อ + Roles)
func (r *PersonnelRepo) GetUsers() ([]map[string]interface{}, error) {
	type UserRow struct {
		UserID       string `gorm:"column:user_id"`
		FullnameThai string `gorm:"column:fullname_thai"`
		FullnameEng  string `gorm:"column:fullname_eng"`
		Picture      string `gorm:"column:picture"`
		RoleInit     string `gorm:"column:role_init"`
		RoleID       string `gorm:"column:role_id"`
		RoleName     string `gorm:"column:role_name"`
		RoleColor    string `gorm:"column:role_color"`
	}

	var rows []UserRow
	// Join 3 ตาราง: user_info -> user_roles -> role
	err := r.db.Table("user_info u").
		Select("u.user_id, u.fullname_thai, u.fullname_eng, u.picture, u.role_init, r.role_id, r.role_name, r.role_color").
		Joins("LEFT JOIN user_roles ur ON u.user_id = ur.user_id").
		Joins("LEFT JOIN role r ON ur.role_id = r.role_id").
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	// Group ข้อมูล Role ใส่ Array ให้แต่ละ User
	userMap := make(map[string]map[string]interface{})
	for _, row := range rows {
		if _, exists := userMap[row.UserID]; !exists {
			userMap[row.UserID] = map[string]interface{}{
				"id":           row.UserID,
				"name-th":      row.FullnameThai,
				"name-en":      row.FullnameEng,
				"avatar-url":   row.Picture,
				"initial-role": row.RoleInit,
				"roles":        []map[string]string{},
			}
		}

		if row.RoleID != "" {
			roles := userMap[row.UserID]["roles"].([]map[string]string)
			roles = append(roles, map[string]string{
				"role-id":    row.RoleID,
				"role-name":  row.RoleName,
				"role-color": row.RoleColor,
			})
			userMap[row.UserID]["roles"] = roles
		}
	}

	var results []map[string]interface{}
	for _, v := range userMap {
		results = append(results, v)
	}

	return results, nil
}

// 6. Check Permission Level
func (r *PersonnelRepo) CheckApprovalPermission(requesterID string, targetID string) (int, error) {
	// 1. เช็คก่อนว่า Requester เป็น Admin หรือเปล่า
	var adminCount int64
	r.db.Table("user_roles ur").
		Joins("JOIN role r ON ur.role_id = r.role_id").
		Where("ur.user_id = ? AND r.role_type = ?", requesterID, "admin").
		Count(&adminCount)

	if adminCount > 0 {
		return 1, nil // เป็น Admin คืนค่า 1 ทันที
	}

	// 2. ถ้าไม่ใช่ Admin ให้เช็คว่า Requester เป็น Manager ของ Target ไหม
	// โดยเชื่อมตาราง subordinate_manager_roles เข้ากับ user_roles ของทั้งสองคน
	var managerCount int64
	r.db.Table("subordinate_manager_roles smr").
		Joins("JOIN user_roles mr ON smr.manager_role_id = mr.role_id"). // ฝั่งหัวหน้า (Requester)
		Joins("JOIN user_roles sr ON smr.subordinate_role_id = sr.role_id"). // ฝั่งลูกน้อง (Target)
		Where("mr.user_id = ? AND sr.user_id = ?", requesterID, targetID).
		Count(&managerCount)

	if managerCount > 0 {
		return 1, nil // มีสิทธิ์อนุมัติ (เป็นหัวหน้าโดยตรง)
	}

	return 0, nil // ไม่มีสิทธิ์เลย
}