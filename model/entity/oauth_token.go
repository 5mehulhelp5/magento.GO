package entity

import "time"

type OauthToken struct {
	EntityID   uint      `gorm:"column:entity_id;primaryKey;autoIncrement"`
	ConsumerID *uint     `gorm:"column:consumer_id"`
	AdminID    *uint     `gorm:"column:admin_id"`
	CustomerID *uint     `gorm:"column:customer_id"`
	Type       string    `gorm:"column:type;type:varchar(16);not null"`
	Token      string    `gorm:"column:token;type:varchar(32);not null;uniqueIndex"`
	Secret     string    `gorm:"column:secret;type:varchar(128);not null"`
	Revoked    uint16    `gorm:"column:revoked;not null;default:0"`
	Authorized uint16    `gorm:"column:authorized;not null;default:0"`
	UserType   *int      `gorm:"column:user_type"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (OauthToken) TableName() string {
	return "oauth_token"
}
