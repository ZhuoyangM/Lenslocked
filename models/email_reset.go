package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/joncalhoun/lenslocked/rand"
)

type EmailReset struct {
	ID     int
	UserID int
	// Token is only set when a EmailReset is being created.
	Token     string
	TokenHash string
	ExpiresAt time.Time
}

type EmailResetService struct {
	DB *sql.DB
	// BytesPerToken is used to determine how many bytes to use when generating
	// each password reset token. If this value is not set or is less than the
	// MinBytesPerToken const it will be ignored and MinBytesPerToken will be
	// used.
	BytesPerToken int
	// Duration is the amount of time that a PasswordReset is valid for.
	// Defaults to DefaultResetDuration
	Duration time.Duration
}

// Create a new email_reset record
func (service *EmailResetService) Create(userID int) (*EmailReset, error) {

	bytesPerToken := service.BytesPerToken
	if bytesPerToken == 0 {
		bytesPerToken = MinBytesPerToken
	}

	token, err := rand.String(bytesPerToken)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}

	duration := service.Duration
	if duration == 0 {
		duration = DefaultResetDuration
	}

	emailReset := EmailReset{
		UserID:    userID,
		Token:     token,
		TokenHash: service.hash(token),
		ExpiresAt: time.Now().Add(duration),
	}

	row := service.DB.QueryRow(`
		INSERT INTO email_resets (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3) ON CONFLICT (user_id) DO
		UPDATE
		SET token_hash = $2, expires_at = $3
		RETURNING id;`, emailReset.UserID, emailReset.TokenHash, emailReset.ExpiresAt)
	err = row.Scan(&emailReset.ID)
	if err != nil {
		return nil, fmt.Errorf("create: %w", err)
	}
	return &emailReset, nil
}

// We are going to consume a token and return the user associated with it, or return an error if the token wasn't valid for any reason.
func (service *EmailResetService) Consume(token string) (*User, error) {
	tokenHash := service.hash(token)
	var user User
	var emailReset EmailReset
	row := service.DB.QueryRow(`
		SELECT 
			e.id,
			e.expires_at,
			u.id,
			u.password_hash
		FROM email_resets AS e
		JOIN users AS u
			ON u.id = e.user_id
		WHERE e.token_hash = $1;`, tokenHash)
	err := row.Scan(
		&emailReset.ID, &emailReset.ExpiresAt,
		&user.ID, &user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	if time.Now().After(emailReset.ExpiresAt) {
		return nil, fmt.Errorf("token expired: %v", token)
	}

	err = service.delete(emailReset.ID)
	if err != nil {
		return nil, fmt.Errorf("consume: %w", err)
	}
	return &user, nil

}

func (service *EmailResetService) delete(id int) error {
	_, err := service.DB.Exec(`
		DELETE FROM email_resets
		WHERE id = $1;`, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

func (service *EmailResetService) hash(token string) string {
	tokenHash := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(tokenHash[:])
}
