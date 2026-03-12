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
func (r *LeaveApprovalRepo) GetPendingSummary(managerID string) ([]map[string]interface{}, error) {
    type Result struct {
        UserID       string `gorm:"column:user_id"`
        Name         string `gorm:"column:name"`
        AvatarURL    string `gorm:"column:avatar_url"`
        RequestCount int    `gorm:"column:request_count"`
    }
    var rows []Result

    // 🌟 CROSS JOIN เพื่ออ่านค่า allow-retroactive จาก JSON ในตาราง config
    sql := `SELECT lr.user_id, ui.fullname_thai as name, ui.picture as avatar_url, COUNT(DISTINCT lr.id) as request_count
            FROM leave_requests lr
            JOIN user_info ui ON lr.user_id = ui.user_id
            JOIN subordinate_manager_roles smr ON lr.user_id = smr.subordinate_id
            JOIN user_roles ur ON smr.manager_role_id = ur.role_id
            CROSS JOIN (SELECT config_value FROM system_configs WHERE config_key = 'leave_config') sc
            WHERE ur.user_id = ? 
              AND lr.status = 'pending' 
              AND (
                  CAST(sc.config_value->lr.leave_type->>'allow-retroactive' AS BOOLEAN) = true
                  OR lr.date_from >= CURRENT_TIMESTAMP
              )
            GROUP BY lr.user_id, ui.fullname_thai, ui.picture`

    r.db.Raw(sql, managerID).Scan(&rows)

    var results []map[string]interface{}
    for _, row := range rows {
        results = append(results, map[string]interface{}{
            "user-id":       row.UserID,
            "name":          row.Name,
            "request-count": row.RequestCount,
            "avatar-url":    row.AvatarURL,
        })
    }
    if len(results) == 0 { return []map[string]interface{}{}, nil }
    return results, nil
}

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

    // 🌟 ใช้ DISTINCT ON (lr.id) เพื่อบังคับให้ 1 ID ออกมาแค่ 1 แถวเท่านั้น
    // และต้องเพิ่ม lr.id เข้าไปใน ORDER BY ตัวแรกสุดตามกฎของ Postgres
    query := `SELECT DISTINCT ON (lr.id) lr.id, lr.user_id, ui.fullname_thai as name, lr.leave_type, lr.date_from, 
                CASE 
                    WHEN lr.status = 'pending' 
                         AND lr.date_from < CURRENT_TIMESTAMP 
                         AND CAST(sc.config_value->lr.leave_type->>'allow-retroactive' AS BOOLEAN) = false 
                    THEN 'overdue' 
                    ELSE lr.status 
                END as status
              FROM leave_requests lr
              JOIN user_info ui ON lr.user_id = ui.user_id
              JOIN subordinate_manager_roles smr ON lr.user_id = smr.subordinate_id
              JOIN user_roles ur ON smr.manager_role_id = ur.role_id
              CROSS JOIN (SELECT config_value FROM system_configs WHERE config_key = 'leave_config') sc
              WHERE ur.user_id = ? 
                AND (
                    lr.status != 'pending' 
                    OR (
                        lr.status = 'pending' 
                        AND lr.date_from < CURRENT_TIMESTAMP 
                        AND CAST(sc.config_value->lr.leave_type->>'allow-retroactive' AS BOOLEAN) = false
                    )
                )`

    args := []interface{}{managerID}
    if startDate != "" && endDate != "" {
        query += ` AND lr.date_from >= ? AND lr.date_from <= ?`
        args = append(args, startDate, endDate)
    }
    
    // 🌟 ต้องเอา lr.id ขึ้นก่อน แล้วค่อยตามด้วย CreatedAt เพื่อเรียงลำดับเวลา
    query += ` ORDER BY lr.id, lr.created_at DESC`

    r.db.Raw(query, args...).Scan(&rows)

    var results []map[string]interface{}
    for _, row := range rows {
        results = append(results, map[string]interface{}{
            "user-id":    row.UserID,
            "name":       row.Name,
            "status":     row.Status,
            "request-id": fmt.Sprintf("LEV%012d", row.ID),
            "type":       row.LeaveType,
            "date-start": row.DateFrom.Format("2006-01-02T15:04:05"), 
        })
    }
    if len(results) == 0 { return []map[string]interface{}{}, nil }
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

// 4. GET /user_detail (เวอร์ชันอัปเกรด Logic และ Format เวลา)
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
        leaveInfo[lt] = map[string]interface{}{"used_days": 0.0, "quota_days": 0.0}
    }
    for _, b := range balances {
        if _, ok := leaveInfo[b.LeaveType]; ok {
            leaveInfo[b.LeaveType] = map[string]interface{}{
                "used_days":  b.DaysUsed,
                "quota_days": b.DaysAllowed,
            }
        }
    }

    // 🌟 3. ดึงรายการ Pending ของ User คนนี้ (อัปเกรด Logic Retroactive)
    type PendingReq struct {
        ID       int       `gorm:"column:id"`
        Type     string    `gorm:"column:leave_type"`
        DateFrom time.Time `gorm:"column:date_from"`
        DateTo   time.Time `gorm:"column:date_to"`
    }
    var pendings []PendingReq
    
    // 🌟 ใช้ CROSS JOIN เช็ค config ย้อนหลัง เพื่อให้รายการตรงกับหน้าจอหลักของบอส
    sql := `SELECT lr.id, lr.leave_type, lr.date_from, lr.date_to
            FROM leave_requests lr
            CROSS JOIN (SELECT config_value FROM system_configs WHERE config_key = 'leave_config') sc
            WHERE lr.user_id = $1 
              AND lr.status = 'pending' 
              AND (
                  CAST(sc.config_value->lr.leave_type->>'allow-retroactive' AS BOOLEAN) = true
                  OR lr.date_from >= CURRENT_TIMESTAMP
              )
            ORDER BY lr.date_from DESC`

    r.db.Raw(sql, targetUserID).Scan(&pendings)

    var pendingList []map[string]interface{}
    for _, p := range pendings {
        pendingList = append(pendingList, map[string]interface{}{
            "request-id": fmt.Sprintf("LEV%012d", p.ID),
            "type":       p.Type,
            // 🌟 แก้ Format เวลา ตัด +07:00 ออก เพื่อความเนียนกริ๊บ
            "date-from":  p.DateFrom.Format("2006-01-02T15:04:05"),
            "date-to":    p.DateTo.Format("2006-01-02T15:04:05"),
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
// 🌟 แก้ Repo: เพิ่ม user-detail ตามที่ Flutter ต้องการ
func (r *LeaveApprovalRepo) GetRequestDetail(managerID string, reqID int, baseURL string) (map[string]interface{}, error) {
    var req struct {
        UserID           string    `gorm:"column:user_id"`
        LeaveType        string    `gorm:"column:leave_type"`
        DateFrom         time.Time `gorm:"column:date_from"`
        DateTo           time.Time `gorm:"column:date_to"`
        FromDateMorning  bool      `gorm:"column:from_date_morning"`
        ToDateMorning    bool      `gorm:"column:to_date_morning"`
        Remark           string    `gorm:"column:remark"`
        CreatedAt        time.Time `gorm:"column:created_at"`
        Status           string    `gorm:"column:status"`
        AllowRetroactive bool      `gorm:"column:allow_retroactive"`
    }

    // 1. ดึงข้อมูลใบลา
    err := r.db.Table("leave_requests lr").
        Select(`
            lr.*, 
            CAST(sc.config_value->lr.leave_type->>'allow-retroactive' AS BOOLEAN) as allow_retroactive
        `).
        Joins("CROSS JOIN (SELECT config_value FROM system_configs WHERE config_key = 'leave_config') sc").
        Where("lr.id = ?", reqID).
        First(&req).Error

    if err != nil {
        return nil, errors.New("request not found")
    }

    if !r.checkPermission(managerID, req.UserID) {
        return nil, errors.New("unauthorized")
    }

    // 2. ดึงไฟล์แนบ
    files := []map[string]interface{}{}
    r.db.Table("leave_attachments").Where("leave_request_id = ?", reqID).
        Select("original_name as \"file-name\", file_path as \"file-url\", file_type as \"file-type\", file_size as \"file-size\"").
        Find(&files)

    for i := range files {
        if path, ok := files[i]["file-url"].(string); ok && !strings.HasPrefix(path, "http") {
            path = strings.ReplaceAll(path, "\\", "/")
            if path[0] == '/' {
                path = path[1:]
            }
            files[i]["file-url"] = baseURL + path
        }
    }

    // 3. ดึงข้อมูลผู้อนุมัติ
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

    // 🌟 4. [NEW] ดึงข้อมูลผู้ขอลา (User Detail)
    var userInfo struct {
        Name      string `gorm:"column:fullname_thai"`
        AvatarURL string `gorm:"column:profile_picture"` // ⚠️ ถ้าใน DB ลูกพี่ใช้ชื่ออื่น (เช่น avatar_url) ให้แก้ตรงนี้นะครับ
    }
    r.db.Table("user_info").Select("fullname_thai, profile_picture").Where("user_id = ?", req.UserID).Scan(&userInfo)

    var initRole string
    r.db.Table("user_roles ur").
        Select("r.role_name").
        Joins("JOIN role r ON ur.role_id = r.role_id").
        Where("ur.user_id = ? AND r.role_type = 'main'", req.UserID).
        Limit(1).Scan(&initRole)

    // 5. ตรรกะเช็คสถานะ Overdue
    finalStatus := req.Status
    if finalStatus == "pending" && req.DateFrom.Before(time.Now()) && !req.AllowRetroactive {
        finalStatus = "overdue"
    }
    
    var approveDateStr interface{} = ""
    if !app.CreatedAt.IsZero() {
        approveDateStr = app.CreatedAt.Format("2006-01-02T15:04:05")
    }

    // 🌟 6. ประกอบร่าง JSON คืนให้ Frontend
    return map[string]interface{}{
        "user-detail": map[string]interface{}{
            "avatar-url": userInfo.AvatarURL,
            "name":       userInfo.Name,
            "init-role":  initRole,
        },
        "request-detail": map[string]interface{}{
            "leave-type":        req.LeaveType,
            "date-from":         req.DateFrom.Format("2006-01-02T15:04:05"),
            "date-to":           req.DateTo.Format("2006-01-02T15:04:05"),
            "from-date-morning": req.FromDateMorning,
            "to-date-morning":   req.ToDateMorning,
            "remark":            req.Remark,
            "evidence-files":    files,
            "request-date":      req.CreatedAt.Format("2006-01-02T15:04:05"),
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

// CalculateLeaveDays ใช้คำนวณจำนวนวันลาจริง โดยคิดเรื่อง ครึ่งเช้า/ครึ่งบ่าย
func CalculateLeaveDays(from, to time.Time, fromAM, toAM bool) float64 {
    // คำนวณวันเบื้องต้น (รวมวันเริ่มและวันจบ)
    days := float64(to.Sub(from).Hours()/24) + 1

    // ถ้าวันแรกไม่ได้ลาช่วงเช้า (เริ่มลาช่วงบ่าย) หักออก 0.5 วัน
    if !fromAM {
        days -= 0.5
    }
    // ถ้าวันสุดท้ายลาแค่ช่วงเช้า (กลับมาทำงานช่วงบ่าย) หักออก 0.5 วัน
    if toAM {
        days -= 0.5
    }

    if days < 0 { return 0 }
    return days
}

func (r *LeaveApprovalRepo) ApproveRejectRequest(managerID string, reqID int, status, reason, signaturePath string) error {
    var manager struct {
        Name string `gorm:"column:fullname_thai"`
    }
    r.db.Table("user_info").Where("user_id = ?", managerID).Select("fullname_thai").First(&manager)

    var approveRole string
    r.db.Table("user_roles ur").Joins("JOIN role r ON ur.role_id = r.role_id").
        Where("ur.user_id = ? AND r.role_type = 'main'", managerID).
        Select("r.role_name").Limit(1).Scan(&approveRole)

    return r.db.Transaction(func(tx *gorm.DB) error {
        // 0. ดึงข้อมูลใบลาเดิมมาดูก่อน (เพื่อเอาไปคำนวณวันและประเภทลา)
        var oldReq struct {
            UserID          string    `gorm:"column:user_id"`
            LeaveType       string    `gorm:"column:leave_type"`
            DateFrom         time.Time `gorm:"column:date_from"`
            DateTo           time.Time `gorm:"column:date_to"`
            FromDateMorning bool      `gorm:"column:from_date_morning"`
            ToDateMorning   bool      `gorm:"column:to_date_morning"`
            Status          string    `gorm:"column:status"`
        }
        if err := tx.Table("leave_requests").Where("id = ?", reqID).First(&oldReq).Error; err != nil {
            return err
        }

        // 🌟 ป้องกันการกดย้ำ: ถ้าอนุมัติไปแล้ว (approved) จะไม่บวกยอด used ซ้ำอีก
        if oldReq.Status == "approved" && status == "approved" {
            return errors.New("คำขอนี้ถูกอนุมัติไปเรียบร้อยแล้ว")
        }

        // 1. อัปเดตตาราง leave_requests
        if err := tx.Table("leave_requests").Where("id = ?", reqID).Update("status", status).Error; err != nil {
            return err
        }

        // 2. บันทึก/อัปเดตตาราง leave_approvals
        var count int64
        tx.Table("leave_approvals").Where("leave_request_id = ?", reqID).Count(&count)
        approvalData := map[string]interface{}{
            "leave_request_id": reqID,
            "approver_name":    manager.Name,
            "approve_role":     approveRole,
            "status":           status,
            "reason":           reason,
            "signature_path":   signaturePath,
            "created_at":       time.Now(),
        }
        if count > 0 {
            if err := tx.Table("leave_approvals").Where("leave_request_id = ?", reqID).Updates(approvalData).Error; err != nil { return err }
        } else {
            if err := tx.Table("leave_approvals").Create(approvalData).Error; err != nil { return err }
        }

        // 🌟 3. [Logic หัวใจสำคัญ] ถ้าบสกด "approved" ให้ไปตัดยอดวันลา!
        if status == "approved" {
            // ก. หา Type ID
            var typeID int
            tx.Table("leave_types").Where("name_en = ?", oldReq.LeaveType).Select("id").Scan(&typeID)

            // ข. คำนวณจำนวนวันที่ลาจริง
            leaveDays := CalculateLeaveDays(oldReq.DateFrom, oldReq.DateTo, oldReq.FromDateMorning, oldReq.ToDateMorning)

            // ค. หาปีงบประมาณ
            budgetYear, _ := GetBudgetYear(tx, oldReq.DateFrom)

            // ง. อัปเดตตาราง leave_balances (บวกวันลาสะสมเพิ่มเข้าไป)
            err := tx.Table("leave_balances").
                Where("user_id = ? AND leave_type_id = ? AND year = ?", oldReq.UserID, typeID, budgetYear).
                UpdateColumn("days_used", gorm.Expr("days_used + ?", leaveDays)).Error
            if err != nil {
                return errors.New("ไม่พบข้อมูลโควตาลาของพนักงานรายนี้ในปีงบประมาณปัจจุบัน")
            }
        }

        return nil
    })
}