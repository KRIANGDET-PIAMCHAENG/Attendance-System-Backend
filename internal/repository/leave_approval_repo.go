package repository

import (
	"errors"
	"fmt"
	//"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type LeaveApprovalRepo struct {
	db *gorm.DB
}

func NewLeaveApprovalRepo(db *gorm.DB) *LeaveApprovalRepo {
	return &LeaveApprovalRepo{db: db}
}

// เช็คสิทธิ์ว่า Manager คนนี้ดูแล User คนนี้จริงไหม
func (r *LeaveApprovalRepo) checkPermission(managerID, targetUserID string) bool {
	if managerID == targetUserID {
		return true // ดูของตัวเองได้
	}
	var adminCount int64
	r.db.Table("user_roles ur").Joins("JOIN role r ON ur.role_id = r.role_id").
		Where("ur.user_id = ? AND r.role_type = ?", managerID, "admin").Count(&adminCount)
	if adminCount > 0 {
		return true
	}
	var subCount int64
	r.db.Table("subordinate_manager_roles smr").
		Joins("JOIN user_roles mr ON smr.manager_role_id = mr.role_id").
		Joins("JOIN role r_manager ON mr.role_id = r_manager.role_id").
		Where("mr.user_id = ? AND smr.subordinate_id = ? AND r_manager.role_type = ?", managerID, targetUserID, "main").
		Count(&subCount)
	return subCount > 0
}

// 1. GET /pending (แบบ Group ตาม User)
func (r *LeaveApprovalRepo) GetPendingSummary(managerID string) ([]map[string]interface{}, error) {
	type Result struct {
		UserID       string `gorm:"column:user_id"`
		Name         string `gorm:"column:name"`
		AvatarURL    string `gorm:"column:avatar_url"`
		RequestCount int    `gorm:"column:request_count"`
	}
	var rows []Result

	// ค้นหาคำขอที่ยัง pending และรวมกลุ่มตาม User
	r.db.Table("leave_requests lr").
		Select("lr.user_id, ui.fullname_thai as name, ui.picture as avatar_url, COUNT(lr.id) as request_count").
		Joins("JOIN user_info ui ON lr.user_id = ui.user_id").
		Joins("JOIN subordinate_manager_roles smr ON lr.user_id = smr.subordinate_id").
		Joins("JOIN user_roles ur ON smr.manager_role_id = ur.role_id").
		Where("ur.user_id = ? AND lr.status = 'pending'", managerID).
		Group("lr.user_id, ui.fullname_thai, ui.picture").
		Scan(&rows)

	var results []map[string]interface{}
	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"user-id":       row.UserID,
			"name":          row.Name,
			"request-count": row.RequestCount,
			"avatar-url":    row.AvatarURL,
		})
	}
	if len(results) == 0 {
		return []map[string]interface{}{}, nil
	}
	return results, nil
}

// 2. GET /recent
func (r *LeaveApprovalRepo) GetRecent(managerID, startDate, endDate string) ([]map[string]interface{}, error) {
	type Result struct {
		ID        int       `gorm:"column:id"`
		UserID    string    `gorm:"column:user_id"`
		Name      string    `gorm:"column:name"`
		LeaveType string    `gorm:"column:leave_type"`
		DateFrom  time.Time `gorm:"column:date_from"`
		Status    string    `gorm:"column:status"`
	}
	var rows []Result

	query := r.db.Table("leave_requests lr").
		Select("lr.id, lr.user_id, ui.fullname_thai as name, lr.leave_type, lr.date_from, lr.status").
		Joins("JOIN user_info ui ON lr.user_id = ui.user_id").
		Joins("JOIN subordinate_manager_roles smr ON lr.user_id = smr.subordinate_id").
		Joins("JOIN user_roles ur ON smr.manager_role_id = ur.role_id").
		Where("ur.user_id = ? AND lr.status != 'pending'", managerID)

	if startDate != "" && endDate != "" {
		query = query.Where("lr.date_from >= ? AND lr.date_from <= ?", startDate, endDate)
	}
	query.Order("lr.updated_at DESC").Scan(&rows)

	var results []map[string]interface{}
	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"user-id":    row.UserID,
			"name":       row.Name,
			"status":     row.Status,
			"request-id": fmt.Sprintf("LEV%012d", row.ID),
			"type":       row.LeaveType,
			"date-start": row.DateFrom.Format(time.RFC3339),
		})
	}
	if len(results) == 0 {
		return []map[string]interface{}{}, nil
	}
	return results, nil
}

// 3. GET /filter_range
func (r *LeaveApprovalRepo) GetFilterRange(managerID string) (map[string]interface{}, error) {
	var res struct {
		Start *time.Time `gorm:"column:min_date"`
		End   *time.Time `gorm:"column:max_date"`
	}
	r.db.Table("leave_requests lr").
		Select("MIN(lr.date_from) as min_date, MAX(lr.date_to) as max_date").
		Joins("JOIN subordinate_manager_roles smr ON lr.user_id = smr.subordinate_id").
		Joins("JOIN user_roles ur ON smr.manager_role_id = ur.role_id").
		Where("ur.user_id = ?", managerID).
		Scan(&res)

	start, end := time.Now(), time.Now()
	if res.Start != nil {
		start = *res.Start
	}
	if res.End != nil {
		end = *res.End
	}
	return map[string]interface{}{
		"start": start.Format("2006-01-02T15:04:05.000Z"),
		"end":   end.Format("2006-01-02T15:04:05.000Z"),
	}, nil
}

// 4. GET /user_detail
func (r *LeaveApprovalRepo) GetUserDetail(managerID, targetUserID string) (map[string]interface{}, error) {
	if !r.checkPermission(managerID, targetUserID) {
		return nil, errors.New("unauthorized")
	}

	// 1. ดึงข้อมูล User
	var user struct {
		Name      string `gorm:"column:fullname_thai"`
		InitRole  string `gorm:"column:role_init"`
		AvatarURL string `gorm:"column:picture"`
	}
	r.db.Table("user_info").Where("user_id = ?", targetUserID).Select("fullname_thai, role_init, picture").First(&user)

	// 2. ดึงข้อมูล Quota (อ้างอิงปีปัจจุบัน)
	currentYear := time.Now().Year()
	type Balance struct {
		LeaveType   string  `gorm:"column:name_en"`
		DaysAllowed float64 `gorm:"column:days_allowed"`
		DaysUsed    float64 `gorm:"column:days_used"`
	}
	var balances []Balance
	r.db.Table("leave_balances lb").
		Select("lt.name_en, lb.days_allowed, lb.days_used").
		Joins("JOIN leave_types lt ON lb.leave_type_id = lt.id").
		Where("lb.user_id = ? AND (lb.year = ? OR lb.year IS NULL)", targetUserID, currentYear).
		Scan(&balances)

	leaveInfo := make(map[string]interface{})
	allLeaveTypes := []string{"sick", "personal", "vacation", "maternity", "paternity", "parental"}
	for _, lt := range allLeaveTypes {
		leaveInfo[lt] = map[string]interface{}{"used_days": 0.0, "quota_days": 0.0} // Default
	}
	for _, b := range balances {
		leaveInfo[b.LeaveType] = map[string]interface{}{
			"used_days":  b.DaysUsed,
			"quota_days": b.DaysAllowed,
		}
	}

	// 3. ดึงรายการ Pending ของ User คนนี้
	type PendingReq struct {
		ID       int       `gorm:"column:id"`
		Type     string    `gorm:"column:leave_type"`
		DateFrom time.Time `gorm:"column:date_from"`
		DateTo   time.Time `gorm:"column:date_to"`
	}
	var pendings []PendingReq
	r.db.Table("leave_requests").
		Select("id, leave_type, date_from, date_to").
		Where("user_id = ? AND status = 'pending'", targetUserID).
		Scan(&pendings)

	var pendingList []map[string]interface{}
	for _, p := range pendings {
		pendingList = append(pendingList, map[string]interface{}{
			"request-id": fmt.Sprintf("LEV%012d", p.ID),
			"type":       p.Type,
			"date-from":  p.DateFrom.Format(time.RFC3339),
			"date-to":    p.DateTo.Format(time.RFC3339),
		})
	}
	if len(pendingList) == 0 {
		pendingList = []map[string]interface{}{}
	}

	return map[string]interface{}{
		"user-detail": map[string]interface{}{
			"name":       user.Name,
			"init-role":  user.InitRole,
			"avatar-url": user.AvatarURL,
		},
		"leave-info":   leaveInfo,
		"user-pending": pendingList,
	}, nil
}

// 5. GET /request_detail
func (r *LeaveApprovalRepo) GetRequestDetail(managerID string, reqID int) (map[string]interface{}, error) {
	// ใช้ Logic เดียวกับ PersonnelRepo.GetDetail ได้เลยครับ (ขออนุญาตดึงข้อมูลให้ครบตาม JSON)
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
		return nil, errors.New("request not found")
	}

	if !r.checkPermission(managerID, req.UserID) {
		return nil, errors.New("unauthorized")
	}

	files := []map[string]interface{}{}
	r.db.Table("leave_attachments").Where("leave_request_id = ?", reqID).
		Select("original_name as \"file-name\", file_path as \"file-url\", file_type as \"file-type\", file_size as \"file-size\"").
		Find(&files)

	baseURL := "http://20.194.9.179:3000/"
	for i := range files {
		if path, ok := files[i]["file-url"].(string); ok && !strings.HasPrefix(path, "http") {
			if path[0] == '/' {
				path = path[1:]
			}
			files[i]["file-url"] = baseURL + path
		}
	}

	var app struct {
		ApproverName string    `gorm:"column:approver_name"`
		ApproveRole  string    `gorm:"column:approve_role"`
		Reason       string    `gorm:"column:reason"`
		CreatedAt    time.Time `gorm:"column:created_at"`
	}
	r.db.Table("leave_approvals").Where("leave_request_id = ?", reqID).First(&app)

	if app.ApproveRole == "" {
		r.db.Table("subordinate_manager_roles smr").
			Select("r.role_name").
			Joins("JOIN role r ON smr.manager_role_id = r.role_id").
			Where("smr.subordinate_id = ? AND r.role_type = ?", req.UserID, "main").
			Limit(1).Scan(&app.ApproveRole)
	}

	finalStatus := req.Status
	if finalStatus == "pending" && req.DateFrom.Before(time.Now()) {
		finalStatus = "overdue"
	}
	var approveDateStr interface{} = nil
	if !app.CreatedAt.IsZero() {
		approveDateStr = app.CreatedAt.Format(time.RFC3339)
	}

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
			"status":       finalStatus,
			"approve-role": app.ApproveRole,
			"approver":     app.ApproverName,
			"reason":       app.Reason,
			"approve-date": approveDateStr,
		},
	}, nil
}

// 6. PUT /api/leave-approval/:id (อนุมัติ/ปฏิเสธ)
func (r *LeaveApprovalRepo) ApproveRejectRequest(managerID string, reqID int, status, reason, signaturePath string) error {
	// ดึงข้อมูลคนอนุมัติ
	var manager struct {
		Name string `gorm:"column:fullname_thai"`
	}
	r.db.Table("user_info").Where("user_id = ?", managerID).Select("fullname_thai").First(&manager)

	var approveRole string
	r.db.Table("user_roles ur").Joins("JOIN role r ON ur.role_id = r.role_id").
		Where("ur.user_id = ? AND r.role_type = 'main'", managerID).
		Select("r.role_name").Limit(1).Scan(&approveRole)

	// 🌟 บังคับใช้ Transaction เพื่อความชัวร์ 100% ว่า Update สำเร็จทุกตาราง!
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. อัปเดตตาราง leave_requests
		if err := tx.Table("leave_requests").Where("id = ?", reqID).Update("status", status).Error; err != nil {
			return err
		}

		// 2. บันทึกตาราง leave_approvals (ถ้ามีอยู่แล้วให้อัปเดต ถ้ายังไม่มีให้ Insert)
		var count int64
		tx.Table("leave_approvals").Where("leave_request_id = ?", reqID).Count(&count)
		
		if count > 0 {
			err := tx.Table("leave_approvals").Where("leave_request_id = ?", reqID).Updates(map[string]interface{}{
				"approver_name":  manager.Name,
				"approve_role":   approveRole,
				"reason":         reason,
				"signature_path": signaturePath,
				"created_at":     time.Now(),
			}).Error
			if err != nil { return err }
		} else {
			err := tx.Table("leave_approvals").Create(map[string]interface{}{
				"leave_request_id": reqID,
				"approver_name":    manager.Name,
				"approve_role":     approveRole,
				"reason":           reason,
				"signature_path":   signaturePath,
				"created_at":       time.Now(),
			}).Error
			if err != nil { return err }
		}

		return nil
	})
}