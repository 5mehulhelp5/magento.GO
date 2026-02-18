package entity

type AuthorizationRule struct {
	RuleID     uint    `gorm:"column:rule_id;primaryKey;autoIncrement"`
	RoleID     uint    `gorm:"column:role_id;not null;default:0"`
	ResourceID *string `gorm:"column:resource_id;type:varchar(255)"`
	Privileges *string `gorm:"column:privileges;type:varchar(20)"`
	Permission *string `gorm:"column:permission;type:varchar(10)"`
}

func (AuthorizationRule) TableName() string {
	return "authorization_rule"
}
