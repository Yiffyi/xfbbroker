package data

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID                   uint
	CreatedAt            time.Time
	UpdatedAt            time.Time
	LastTransCheckedAt   time.Time
	LastBalanceCheckedAt time.Time

	Name               string
	OpenId             string
	SessionId          string `gorm:"index:idx_session_id,unique"`
	YmUserId           string `gorm:"index:idx_ym_user_id,unique"`
	LastTransSerial    int
	AutoTopupThreshold float64
	EnableAutoTopup    bool
	EnableTransNotify  bool
}

func (db *DB) SelectUserFromSessionId(sessionId string) (*User, error) {
	var user User
	err := db.Where("session_id = ?", sessionId).First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("user with sessionId=%s not found", sessionId)
	}
	return &user, err
}

func (db *DB) SelectUserFromYmUserId(ymUserId string) (*User, error) {
	var user User
	err := db.Where("ym_user_id = ?", ymUserId).First(&user).Error
	return &user, err
}

func (db *DB) SelectUsersForBalanceCheck(batchSize int) ([]User, error) {
	var users []User
	err := db.Limit(batchSize).Where("enable_auto_topup = true").Order("last_balance_checked_at ASC").Find(&users).Error
	if err != nil {
		return nil, err
	}

	err = db.Model(&users).Update("last_balance_checked_at", time.Now()).Error
	return users, err
}

func (db *DB) SelectUsersForTransCheck(batchSize int) ([]User, error) {
	var users []User
	err := db.Limit(batchSize).Where("enable_trans_notify = true").Order("last_trans_checked_at ASC").Find(&users).Error
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, nil
	}

	err = db.Model(&users).Update("last_trans_checked_at", time.Now()).Error
	return users, err
}

func (db *DB) CreateUser(name, openId, sessionId, ymUserId string) (*User, error) {
	u := User{
		Name:               name,
		OpenId:             openId,
		SessionId:          sessionId,
		YmUserId:           ymUserId,
		LastTransSerial:    0,
		AutoTopupThreshold: 100,
		EnableAutoTopup:    false,
		EnableTransNotify:  false,
	}
	err := db.Create(&u).Error
	return &u, err
}

func (db *DB) UpdateUserSessionId(u *User, sessionId string) error {
	return db.Model(u).Update("session_id", sessionId).Error
}

func (db *DB) UpdateUserTransSerial(u *User, serial int) error {
	return db.Model(u).Where("last_trans_serial <= ?", serial).Update("last_trans_serial", serial).Error
}

func (db *DB) UpdateUserLoopEnabled(u *User) error {
	return db.Model(u).Update("enable_auto_topup", u.EnableAutoTopup).Update("enable_trans_notify", u.EnableTransNotify).Error
}

type NotificationChannel struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time

	UserID uint
	User   User

	Name        string
	Enabled     bool
	ChannelType string
	Params      string
}

func (db *DB) SelectNotificationChannelForUser(userId uint) ([]NotificationChannel, error) {
	var cs []NotificationChannel
	err := db.Where("user_id = ?", userId).Find(&cs).Error
	return cs, err
}

func (db *DB) CreateNotificationChannel(userId uint, name, channelType, params string, enabled bool) (*NotificationChannel, error) {
	c := NotificationChannel{
		UserID:      userId,
		Name:        name,
		Enabled:     enabled,
		ChannelType: channelType,
		Params:      params,
	}
	err := db.Create(&c).Error
	return &c, err
}
