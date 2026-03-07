package repository

import (
	"errors"
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
// 1. Get Pending
func (r *PersonnelRepo) GetPending(managerID, personnelID string) ([]map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) {
		return nil, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูลของพนักงานท่านนี้")
	}

	var results []map[string]interface{}
	type Result struct {
		ID        int; LeaveType string; DateFrom  time.Time
	}
	var rows []Result

	r.db.Table("leave_requests").
		Select("id, leave_type, date_from").
		Where("user_id = ? AND status = 'pending'", personnelID).
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

// 2. Get Recent
func (r *PersonnelRepo) GetRecent(managerID, personnelID, startDate, endDate string) ([]map[string]interface{}, error) {
	if !r.checkPermission(managerID, personnelID) {
		return nil, errors.New("unauthorized: ไม่มีสิทธิ์ดูข้อมูลของพนักงานท่านนี้")
	}

	var results []map[string]interface{}
	query := r.db.Table("leave_requests").
		Select("id, leave_type, date_from, status").
		Where("user_id = ? AND status != 'pending'", personnelID)

	if startDate != "" && endDate != "" {
		query = query.Where("date_from >= ? AND date_from <= ?", startDate, endDate)
	}

	type Result struct {
		ID int; LeaveType string; DateFrom time.Time; Status string
	}
	var rows []Result
	query.Scan(&rows)

	for _, row := range rows {
		results = append(results, map[string]interface{}{
			"id":         fmt.Sprintf("LEV%012d", row.ID),
			"leave-type": row.LeaveType,
			"date-start": row.DateFrom.Format(time.RFC3339),
			"status":     row.Status,
			"approved":   (row.Status == "approved"),
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
        UserID string `gorm:"column:user_id"`
        LeaveType string `gorm:"column:leave_type"`; DateFrom time.Time `gorm:"column:date_from"`
        DateTo time.Time `gorm:"column:date_to"`; FromDateMorning bool `gorm:"column:from_date_morning"`
        ToDateMorning bool `gorm:"column:to_date_morning"`; Remark string `gorm:"column:remark"`
        CreatedAt time.Time `gorm:"column:created_at"`; Status string `gorm:"column:status"`
    }
    if err := r.db.Table("leave_requests").Where("id = ?", reqID).First(&req).Error; err != nil {
        return nil, errors.New("ไม่พบใบคำขอนี้")
    }

    // 🛡️ เช็คสิทธิ์
    if !r.checkPermission(managerID, req.UserID) {
        return nil, errors.New("unauthorized: คุณไม่มีสิทธิ์ดูรายละเอียดใบคำขอของพนักงานท่านนี้")
    }

    // 🌟 [แก้ตรงนี้] ดึงข้อมูลไฟล์แนบมาเก็บไว้ก่อน
    var files []map[string]interface{}
    r.db.Table("leave_attachments").Where("leave_request_id = ?", reqID).
        Select("original_name as \"file-name\", file_path as \"file-url\", file_type as \"file-type\", file_size as \"file-size\"").
        Find(&files)

    // 🌟 [เพิ่มตรงนี้] วน Loop เติม Base URL ให้เป็น Link เต็ม
    // ⚠️ ถ้าขึ้น Production จริงๆ แนะนำให้ดึงจาก os.Getenv("BASE_URL") นะครับ
    baseURL := "http://20.194.9.179:3000/" 
    for i := range files {
        if path, ok := files[i]["file-url"].(string); ok && path != "" {
            // เช็คกันเหนียว ถ้าใน DB มันไม่ได้ขึ้นต้นด้วย http:// (แปลว่าเป็น Path สั้น) ให้เติม baseURL นำหน้า
            if len(path) < 4 || path[:4] != "http" {
                // ลบ / ด้านหน้าสุดออก (ถ้ามี) เพื่อไม่ให้ http://20.194.9.179:3000//uploads slash เบิ้ล
                if path[0] == '/' {
                    path = path[1:]
                }
                files[i]["file-url"] = baseURL + path
            }
        }
    }

    // (ดึงข้อมูลการอนุมัติ เหมือนเดิม)
    var app struct {
        ApproverName string `gorm:"column:approver_name"`; ApproveRole string `gorm:"column:approve_role"`
        Reason string `gorm:"column:reason"`; CreatedAt time.Time `gorm:"column:created_at"`
    }
    r.db.Table("leave_approvals").Where("leave_request_id = ?", reqID).First(&app)

    return map[string]interface{}{
        "request-detail": map[string]interface{}{
            "leave-type": req.LeaveType, "date-from": req.DateFrom.Format(time.RFC3339), "date-to": req.DateTo.Format(time.RFC3339),
            "from-date-morning": req.FromDateMorning, "to-date-morning": req.ToDateMorning,
            "remark": req.Remark, "evidence-files": files, "request-date": req.CreatedAt.Format(time.RFC3339),
        },
        "approve-detail": map[string]interface{}{
            "status": req.Status, "approve-role": app.ApproveRole, "approver": app.ApproverName,
            "reason": app.Reason, "approve-date": app.CreatedAt.Format(time.RFC3339),
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