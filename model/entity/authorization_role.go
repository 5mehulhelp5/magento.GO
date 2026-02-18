package entity

type AuthorizationRole struct {
	RoleID    uint   `gorm:"column:role_id;primaryKey;autoIncrement"`
	ParentID  uint   `gorm:"column:parent_id;not null;default:0"`
	TreeLevel uint16 `gorm:"column:tree_level;not null;default:0"`
	SortOrder uint16 `gorm:"column:sort_order;not null;default:0"`
	RoleType  string `gorm:"column:role_type;type:varchar(1);not null;default:'0'"`
	UserID    uint   `gorm:"column:user_id;not null;default:0"`
	UserType  string `gorm:"column:user_type;type:varchar(16)"`
	RoleName  string `gorm:"column:role_name;type:varchar(50)"`
}

func (AuthorizationRole) TableName() string {
	return "authorization_role"
}
