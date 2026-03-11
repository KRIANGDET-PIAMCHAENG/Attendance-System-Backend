package repository

import (
	//"errors"
	"fmt"
	//"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

type AttendanceApprovalRepo struct {
	db *gorm.DB
}

func NewAttendanceApprovalRepo(db *gorm.DB) *AttendanceApprovalRepo {
	return &AttendanceApprovalRepo{db: db}
}

func (r *AttendanceApprovalRepo) GetPending(managerID string) ([]map[string]interface{}, error) {
	type Result struct {
		ID     int    `gorm:"column:id"`
		UserID string `gorm:"column:user_id"`
		Name   string `gorm:"column:fullname_thai"`
	}
	var rows []Result

	r.db.Table("attendance_requests ar").
		Select("DISTINCT ar.id, ar.user_id, ui.fullname_thai").
		Joins("JOIN user_info ui ON ar.user_id = ui.user_id").
		Joins("JOIN subordinate_manager_roles smr ON ar.user_id = smr.subordinate_id").
		Joins("JOIN user_roles ur ON smr.manager_role_id = ur.role_id").
		Where("ur.user_id = ? AND ar.status = 'pending'", managerID).
		Scan(&rows)

	var results []map[string]interface{}
	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"name":         row.Name,
			"attendanceId": fmt.Sprintf("REQ%012d", row.ID),
		})
	}
	if len(results) == 0 { return []map[string]interface{}{}, nil }
	return results, nil
}

func (r *AttendanceApprovalRepo) GetRecent(managerID, start, end string) ([]map[string]interface{}, error) {
	type Result struct {
		ID     int    `gorm:"column:id"`
		Name   string `gorm:"column:fullname_thai"`
		Status string `gorm:"column:status"`
	}
	var rows []Result

	query := r.db.Table("attendance_requests ar").
		Select("ar.id, ui.fullname_thai, ar.status").
		Joins("JOIN user_info ui ON ar.user_id = ui.user_id").
		Joins("JOIN subordinate_manager_roles smr ON ar.user_id = smr.subordinate_id").
		Joins("JOIN user_roles ur ON smr.manager_role_id = ur.role_id").
		Where("ur.user_id = ? AND ar.status != 'pending'", managerID)

	if start != "" && end != "" {
		query = query.Where("ar.date_from >= ? AND ar.date_from <= ?", start, end)
	}
	query.Order("ar.created_at DESC").Scan(&rows)

	var results []map[string]interface{}
	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"name":         row.Name,
			"status":       row.Status,
			"attendanceId": fmt.Sprintf("REQ%012d", row.ID),
		})
	}
	if len(results) == 0 { return []map[string]interface{}{}, nil }
	return results, nil
}

func (r *AttendanceApprovalRepo) GetFilterRange(managerID string) (map[string]interface{}, error) {
	var res struct {
		Start *time.Time `gorm:"column:min_date"`
		End   *time.Time `gorm:"column:max_date"`
	}
	r.db.Table("attendance_requests ar").
		Select("MIN(ar.date_from) as min_date, MAX(ar.date_to) as max_date").
		Joins("JOIN subordinate_manager_roles smr ON ar.user_id = smr.subordinate_id").
		Joins("JOIN user_roles ur ON smr.manager_role_id = ur.role_id").
		Where("ur.user_id = ?", managerID).
		Scan(&res)

	start, end := time.Now(), time.Now()
	if res.Start != nil { start = *res.Start }
	if res.End != nil { end = *res.End }
	return map[string]interface{}{
		"start": start.Format("2006-01-02T15:04:05.000Z"),
		"end":   end.Format("2006-01-02T15:04:05.000Z"),
	}, nil
}

func (r *AttendanceApprovalRepo) GetRequestDetail(managerID string, reqID int) (map[string]interface{}, error) {
    var req struct {
        UserID    string    `gorm:"column:user_id"`
        Name      string    `gorm:"column:fullname_thai"`
        InitRole  string    `gorm:"column:role_init"`
        Avatar    string    `gorm:"column:picture"`
        DateFrom  time.Time `gorm:"column:date_from"`
        DateTo    time.Time `gorm:"column:date_to"`
        StartTime string    `gorm:"column:start_time"`
        EndTime   string    `gorm:"column:end_time"`
        Remark    string    `gorm:"column:remark"`
        Status    string    `gorm:"column:status"`
    }
    r.db.Table("attendance_requests ar").
        Select("ar.*, ui.fullname_thai, ui.role_init, ui.picture").
        Joins("JOIN user_info ui ON ar.user_id = ui.user_id").
        Where("ar.id = ?", reqID).First(&req)

    files := []map[string]interface{}{}
    r.db.Table("attendance_request_attachments").Where("attendance_request_id = ?", reqID).
        Select("original_name as \"file-name\", file_path as \"file-url\", file_type as \"file-type\", file_size as \"file-size\"").Find(&files)
    baseURL := "http://20.194.9.179:3000/"
    
    for i := range files {
        if path, ok := files[i]["file-url"].(string); ok && !strings.HasPrefix(path, "http") {
            path = strings.ReplaceAll(path, "\\", "/")
            if path[0] == '/' {
                path = path[1:]
            }
            files[i]["file-url"] = baseURL + path
        }
    }

    var app struct {
        ApproverID  string    `gorm:"column:approver_id"`
        ApproveRole string    `gorm:"column:approve_role"`
        Reason      string    `gorm:"column:reason"`
        CreatedAt   time.Time `gorm:"column:created_at"`
    }
    r.db.Table("attendance_approvals").Where("attendance_request_id = ?", reqID).Limit(1).Find(&app)

    var approverName string
    if app.ApproverID != "" {
        r.db.Table("user_info").Where("user_id = ?", app.ApproverID).Select("fullname_thai").Scan(&approverName)
    }

    if app.ApproveRole == "" {
        r.db.Table("user_roles ur").Joins("JOIN role r ON ur.role_id = r.role_id").
            Where("ur.user_id = ? AND r.role_type = 'main'", managerID).Select("r.role_name").Limit(1).Scan(&app.ApproveRole)
    }

    var approveDateStr interface{} = "" 
    if !app.CreatedAt.IsZero() {
        // 🌟 เอา .UTC() ออก และตัด Z ทิ้ง
        approveDateStr = app.CreatedAt.Format("2006-01-02T15:04:05")
    }

    return map[string]interface{}{
        "request-detail": map[string]interface{}{
            // 🌟 เอา .UTC() ออก และตัด Z ทิ้ง
            "date-from":      req.DateFrom.Format("2006-01-02T15:04:05"),
            "date-to":        req.DateTo.Format("2006-01-02T15:04:05"),
            "time-start":     req.StartTime,
            "time-end":       req.EndTime,
            "remark":         req.Remark,
            "evidence-files": files,
        },
        "approve-detail": map[string]interface{}{
            "status":       req.Status,
            "approve-role": app.ApproveRole,
            "approver":     approverName,
            "reason":       app.Reason,
            "approve-date": approveDateStr,
        },
        "user-detail": map[string]interface{}{
            "avatar-url": req.Avatar,
            "name":       req.Name,
            "init-role":  req.InitRole,
        },
    }, nil
}

// 🌟 ฟังก์ชันอนุมัติ พร้อมแก้ตาราง Attendance
func (r *AttendanceApprovalRepo) ApproveRejectRequest(managerID string, reqID int, status, reason, signaturePath string) error {
	var manager struct { 
        Name string `gorm:"column:fullname_thai"` 
    }
	r.db.Table("user_info").Where("user_id = ?", managerID).Select("fullname_thai").First(&manager)
	var approveRole string
	r.db.Table("user_roles ur").Joins("JOIN role r ON ur.role_id = r.role_id").
		Where("ur.user_id = ? AND r.role_type = 'main'", managerID).Select("r.role_name").Limit(1).Scan(&approveRole)

	// ดึงข้อมูล Request เอาไว้ไปหยอดลงตาราง Attendance จริงๆ
	var req struct {
		UserID    string    `gorm:"column:user_id"`
		DateFrom  time.Time `gorm:"column:date_from"`
		DateTo    time.Time `gorm:"column:date_to"`
		StartTime string    `gorm:"column:start_time"`
		EndTime   string    `gorm:"column:end_time"`
	}
	r.db.Table("attendance_requests").Where("id = ?", reqID).First(&req)

	// Transaction 
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Update สถานะคำขอ
		if err := tx.Table("attendance_requests").Where("id = ?", reqID).Update("status", status).Error; err != nil { return err }

		// 2. บันทึกคนอนุมัติ
		var count int64
		tx.Table("attendance_approvals").Where("attendance_request_id = ?", reqID).Count(&count)
		if count > 0 {
			tx.Table("attendance_approvals").Where("attendance_request_id = ?", reqID).Updates(map[string]interface{}{
				"approver_id": managerID, // 🌟 เปลี่ยนมาเซฟ ID แทน
                "approve_role": approveRole, 
                "status": status, // 🌟 เพิ่มสถานะ
                "reason": reason, 
                "signature_path": signaturePath, 
                "created_at": time.Now(),
			})
		} else {
			tx.Table("attendance_approvals").Create(map[string]interface{}{
				"attendance_request_id": reqID, 
                "approver_id": managerID, // 🌟 เปลี่ยนมาเซฟ ID แทน
                "approve_role": approveRole, 
                "status": status, // 🌟 เพิ่มสถานะ
                "reason": reason, 
                "signature_path": signaturePath, 
                "created_at": time.Now(),
			})
		}

		// 🌟 3. [สำคัญมาก] ถ้ากด "approved" ให้แก้ตาราง attendance ด้วย!
		if status == "approved" {
			for d := req.DateFrom; !d.After(req.DateTo); d = d.AddDate(0, 0, 1) {
				dateStr := d.Format("2006-01-02")
				// หยอดข้อมูล (ถ้ามีอยู่แล้วให้ Update ทับ ถ้ายังไม่มีให้สร้างใหม่)
				var attCount int64
				tx.Table("attendance").Where("user_id = ? AND date = ?", req.UserID, dateStr).Count(&attCount)
				if attCount > 0 {
					tx.Table("attendance").Where("user_id = ? AND date = ?", req.UserID, dateStr).Updates(map[string]interface{}{
						"check_in": req.StartTime, "check_out": req.EndTime, "updated_at": time.Now(),
					})
				} else {
					tx.Table("attendance").Create(map[string]interface{}{
						"user_id": req.UserID, "date": dateStr, "check_in": req.StartTime, "check_out": req.EndTime, "created_at": time.Now(), "updated_at": time.Now(),
					})
				}
			}
		}
		return nil
	})
}