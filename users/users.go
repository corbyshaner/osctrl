package users

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/jinzhu/gorm/dialects/postgres"
	"golang.org/x/crypto/bcrypt"
)

// AdminUser to hold all users
type AdminUser struct {
	gorm.Model
	Username      string `gorm:"index"`
	Email         string
	Fullname      string
	PassHash      string
	APIToken      string
	TokenExpire   time.Time
	Admin         bool
	CSRFToken     string
	Permissions   postgres.Jsonb
	LastIPAddress string
	LastUserAgent string
	LastAccess    time.Time
	LastTokenUse  time.Time
}

// EnvPermissions to hold permissions for environments
type EnvPermissions map[string]bool

// UserPermissions to abstract the permissions for a user
type UserPermissions struct {
	Environments EnvPermissions `json:"environments"`
	Query        bool           `json:"query"`
	Carve        bool           `json:"carve"`
}

// TokenClaims to hold user claims when using JWT
type TokenClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// UserManager have all users of the system
type UserManager struct {
	DB *gorm.DB
}

// CreateUserManager to initialize the users struct and tables
func CreateUserManager(backend *gorm.DB) *UserManager {
	var u *UserManager
	u = &UserManager{DB: backend}
	// table admin_users
	if err := backend.AutoMigrate(AdminUser{}).Error; err != nil {
		log.Fatalf("Failed to AutoMigrate table (admin_users): %v", err)
	}
	return u
}

// HashTextWithSalt to hash text before store it
func (m *UserManager) HashTextWithSalt(text string) (string, error) {
	saltedBytes := []byte(text)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hash := string(hashedBytes)
	return hash, nil
}

// HashPasswordWithSalt to hash a password before store it
func (m *UserManager) HashPasswordWithSalt(password string) (string, error) {
	return m.HashTextWithSalt(password)
}

// CheckLoginCredentials to check provided login credentials by matching hashes
func (m *UserManager) CheckLoginCredentials(username, password string) (bool, AdminUser) {
	// Retrieve user
	user, err := m.Get(username)
	if err != nil {
		return false, AdminUser{}
	}
	// Check for hash matching
	p := []byte(password)
	existing := []byte(user.PassHash)
	err = bcrypt.CompareHashAndPassword(existing, p)
	if err != nil {
		return false, AdminUser{}
	}
	return true, user
}

// CreateToken to create a new JWT token for a given user
func (m *UserManager) CreateToken(username string, expireHours int, jwtSecret string) (string, time.Time, error) {
	expirationTime := time.Now().Add(time.Hour * time.Duration(expireHours))
	// Create the JWT claims, which includes the username, level and expiry time
	claims := &TokenClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
			Issuer:    "admin",
		},
	}
	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", time.Now(), err
	}
	return tokenString, expirationTime, nil
}

// CheckToken to verify if a token used is valid
func (m *UserManager) CheckToken(jwtSecret, tokenStr string) (TokenClaims, bool) {
	claims := &TokenClaims{}
	tkn, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil {
		log.Printf("Error %v", err)
		return *claims, false
	}
	if !tkn.Valid {
		log.Println("Not valid")
		return *claims, false
	}
	return *claims, true
}

// Get user by username
func (m *UserManager) Get(username string) (AdminUser, error) {
	var user AdminUser
	if err := m.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return user, err
	}
	return user, nil
}

// Create new user
func (m *UserManager) Create(user AdminUser) error {
	if m.DB.NewRecord(user) {
		if err := m.DB.Create(&user).Error; err != nil {
			return fmt.Errorf("Create AdminUser %v", err)
		}
	} else {
		return fmt.Errorf("db.NewRecord did not return true")
	}
	return nil
}

// New empty user
func (m *UserManager) New(username, password, email, fullname string, admin bool) (AdminUser, error) {
	if !m.Exists(username) {
		passhash, err := m.HashPasswordWithSalt(password)
		if err != nil {
			return AdminUser{}, err
		}
		permsRaw, err := json.Marshal(m.GenPermissions([]string{}, admin))
		if err != nil {
			permsRaw = []byte("{}")
		}
		return AdminUser{
			Username:    username,
			PassHash:    passhash,
			Admin:       admin,
			Permissions: postgres.Jsonb{RawMessage: permsRaw},
			Email:       email,
			Fullname:    fullname,
		}, nil
	}
	return AdminUser{}, fmt.Errorf("%s already exists", username)
}

// Exists checks if user exists
func (m *UserManager) Exists(username string) bool {
	var results int
	m.DB.Model(&AdminUser{}).Where("username = ?", username).Count(&results)
	return (results > 0)
}

// ExistsGet checks if user exists and returns the user
func (m *UserManager) ExistsGet(username string) (bool, AdminUser) {
	user, err := m.Get(username)
	if err != nil {
		return false, AdminUser{}
	}
	return true, user
}

// IsAdmin checks if user is an admin
func (m *UserManager) IsAdmin(username string) bool {
	var results int
	m.DB.Model(&AdminUser{}).Where("username = ? AND admin = ?", username, true).Count(&results)
	return (results > 0)
}

// ChangeAdmin to modify the admin setting for a user
func (m *UserManager) ChangeAdmin(username string, admin bool) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	if admin != user.Admin {
		if err := m.DB.Model(&user).Updates(map[string]interface{}{"admin": admin}).Error; err != nil {
			return err
		}
	}
	return nil
}

// GenPermissions to generate the struct with empty permissions
func (m *UserManager) GenPermissions(environments []string, level bool) UserPermissions {
	envs := make(EnvPermissions)
	for _, e := range environments {
		envs[e] = level
	}
	perms := UserPermissions{
		Environments: envs,
		Query:        level,
		Carve:        level,
	}
	return perms
}

// ChangePermissions for setting user permissions by username
func (m *UserManager) ChangePermissions(username string, permissions UserPermissions) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	rawPerms, err := json.Marshal(permissions)
	if err != nil {
		return err
	}
	if err := m.DB.Model(&user).Update("permissions", postgres.Jsonb{RawMessage: rawPerms}).Error; err != nil {
		return fmt.Errorf("Update %v", err)
	}
	return nil
}

// GetPermissions to extract permissions by username
func (m *UserManager) GetPermissions(username string) (UserPermissions, error) {
	var perms UserPermissions
	user, err := m.Get(username)
	if err != nil {
		return perms, fmt.Errorf("error getting user %v", err)
	}
	if err := json.Unmarshal(user.Permissions.RawMessage, &perms); err != nil {
		return perms, fmt.Errorf("error parsing permissions %v", err)
	}
	return perms, nil
}

// ConvertPermissions to convert from stored Jsonb to struct
func (m *UserManager) ConvertPermissions(raw json.RawMessage) (UserPermissions, error) {
	var perms UserPermissions
	if err := json.Unmarshal(raw, &perms); err != nil {
		return perms, fmt.Errorf("error parsing permissions %v", err)
	}
	return perms, nil
}

// All get all users
func (m *UserManager) All() ([]AdminUser, error) {
	var users []AdminUser
	if err := m.DB.Find(&users).Error; err != nil {
		return users, err
	}
	return users, nil
}

// Delete user by username
func (m *UserManager) Delete(username string) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	if err := m.DB.Unscoped().Delete(&user).Error; err != nil {
		return fmt.Errorf("Delete %v", err)
	}
	return nil
}

// ChangePassword for user by username
func (m *UserManager) ChangePassword(username, password string) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	passhash, err := m.HashPasswordWithSalt(password)
	if err != nil {
		return err
	}
	if passhash != user.PassHash {
		if err := m.DB.Model(&user).Update("pass_hash", passhash).Error; err != nil {
			return fmt.Errorf("Update %v", err)
		}
	}
	return nil
}

// UpdateToken for user by username
func (m *UserManager) UpdateToken(username, token string, exp time.Time) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	if token != user.APIToken {
		if err := m.DB.Model(&user).Updates(
			AdminUser{
				APIToken:    token,
				TokenExpire: exp,
			}).Error; err != nil {
			return fmt.Errorf("Update %v", err)
		}
	}
	return nil
}

// ChangeEmail for user by username
func (m *UserManager) ChangeEmail(username, email string) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	if email != user.Email {
		if err := m.DB.Model(&user).Update("email", email).Error; err != nil {
			return fmt.Errorf("Update %v", err)
		}
	}
	return nil
}

// ChangeFullname for user by username
func (m *UserManager) ChangeFullname(username, fullname string) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	if fullname != user.Fullname {
		if err := m.DB.Model(&user).Update("fullname", fullname).Error; err != nil {
			return fmt.Errorf("Update %v", err)
		}
	}
	return nil
}

// UpdateMetadata updates IP, User Agent and Last Access for a given user
func (m *UserManager) UpdateMetadata(ipaddress, useragent, username, csrftoken string) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	if err := m.DB.Model(&user).Updates(
		AdminUser{
			LastIPAddress: ipaddress,
			LastUserAgent: useragent,
			CSRFToken:     csrftoken,
			LastAccess:    time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("Update %v", err)
	}
	return nil
}

// UpdateTokenIPAddress updates IP and Last Access for a user's token
func (m *UserManager) UpdateTokenIPAddress(ipaddress, username string) error {
	user, err := m.Get(username)
	if err != nil {
		return fmt.Errorf("error getting user %v", err)
	}
	if err := m.DB.Model(&user).Updates(
		AdminUser{
			LastIPAddress: ipaddress,
			LastTokenUse:  time.Now(),
		}).Error; err != nil {
		return fmt.Errorf("Update %v", err)
	}
	return nil
}
