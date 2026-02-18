package entity

import "time"

type AdminUser struct {
	UserID    uint      `gorm:"column:user_id;primaryKey;autoIncrement"`
	Firstname *string   `gorm:"column:firstname;type:varchar(32)"`
	Lastname  *string   `gorm:"column:lastname;type:varchar(32)"`
	Email     *string   `gorm:"column:email;type:varchar(128)"`
	Username  *string   `gorm:"column:username;type:varchar(40);uniqueIndex"`
	IsActive  int16     `gorm:"column:is_active;not null;default:1"`
	Created   time.Time `gorm:"column:created;autoCreateTime"`
	Modified  time.Time `gorm:"column:modified;autoUpdateTime"`
}

func (AdminUser) TableName() string {
	return "admin_user"
}
