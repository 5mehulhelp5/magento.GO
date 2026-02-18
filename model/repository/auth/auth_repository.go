package auth

import (
	"gorm.io/gorm"

	entity "magento.GO/model/entity"
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

// FindActiveToken returns a non-revoked access token by its token string.
func (r *AuthRepository) FindActiveToken(token string) (*entity.OauthToken, error) {
	var t entity.OauthToken
	err := r.db.Where("token = ? AND type = 'access' AND revoked = 0", token).First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// FindUserRole returns the role assignment (role_type='U') for a given admin user ID.
func (r *AuthRepository) FindUserRole(adminID uint) (*entity.AuthorizationRole, error) {
	var role entity.AuthorizationRole
	err := r.db.Where("user_id = ? AND role_type = 'U'", adminID).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// FindGroupRole returns the group role (role_type='G') by role ID.
func (r *AuthRepository) FindGroupRole(roleID uint) (*entity.AuthorizationRole, error) {
	var role entity.AuthorizationRole
	err := r.db.Where("role_id = ? AND role_type = 'G'", roleID).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// FindAllowedResources returns all allowed ACL resource IDs for a given role ID.
func (r *AuthRepository) FindAllowedResources(roleID uint) ([]string, error) {
	var rules []entity.AuthorizationRule
	if err := r.db.Where("role_id = ? AND permission = 'allow'", roleID).Find(&rules).Error; err != nil {
		return nil, err
	}
	resources := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.ResourceID != nil {
			resources = append(resources, *rule.ResourceID)
		}
	}
	return resources, nil
}
