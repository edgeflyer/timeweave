package models

import "time"

// UserStatus 用户状态枚举。
//
// 使用具名类型而不是直接用 int8，配合 const 让状态语义自解释，
// 后续在 service 层判断 if user.Status == UserStatusActive {} 更清晰。
type UserStatus int8

const (
	UserStatusActive   UserStatus = 0 // 正常
	UserStatusDisabled UserStatus = 1 // 禁用
)

// User 是 QuestOS 的账号实体。
//
// 字段说明：
//   - Email 用 191 字符：MySQL InnoDB 在 utf8mb4 下唯一索引最长 767 字节，
//     191 × 4 = 764，刚好卡在限制内，保证 uniqueIndex 能建上
//   - PasswordHash 用 255：bcrypt 输出只有 60 字符，留大点方便后续换哈希算法
//   - PasswordHash 加 json:"-"：**绝不能**让密码哈希通过 API 返回
//   - LastLoginAt 用指针：登录前为 nil
type User struct {
	BaseModel
	Email        string     `gorm:"size:191;uniqueIndex;not null" json:"email"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	Nickname     string     `gorm:"size:64;default:''" json:"nickname"`
	Status       UserStatus `gorm:"default:0;not null" json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// TableName 显式指定表名，不依赖 GORM 默认复数规则。
// 显式声明的好处：重命名 model 时不会悄悄改表名，迁移脚本更稳。
func (User) TableName() string {
	return "users"
}