package models

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/joncalhoun/lenslocked/rand"
)

const (
	MinBytesPerToken = 32 // Big enough for attackers to guess a correct session token
)

// Session struct
type Session struct {
	ID     int
	UserID int
	// Token is only set when creating a new session
	// and won't be stored in the database
	Token     string
	TokenHash string
}

// Session service struct
type SessionService struct {
	DB            *sql.DB
	BytesPerToken int
}

// Create a new session for the given user
func (ss *SessionService) Create(userID int) (*Session, error) {
	// Create session token
	bytesPerToken := ss.BytesPerToken
	if bytesPerToken < MinBytesPerToken {
		bytesPerToken = MinBytesPerToken
	}
	token, err := rand.String(bytesPerToken)
	if err != nil {
		return nil, fmt.Errorf("Error creating session token: %w", err)
	}

	//Create/update a session and store it in database
	session := Session{
		UserID:    userID,
		Token:     token,
		TokenHash: ss.hash(token),
	}

	row := ss.DB.QueryRow(`
		INSERT INTO sessions (user_id, token_hash)
		VALUES ($1, $2) ON CONFLICT (user_id) DO
		UPDATE
		SET token_hash = $2
		RETURNING id;
	`, session.UserID, session.TokenHash)
	err = row.Scan(&session.ID)
	if err != nil {
		return nil, fmt.Errorf("Error creating session: %w", err)
	}
	return &session, nil
}

// Query for the user with given session token
func (ss *SessionService) User(token string) (*User, error) {
	tokenHash := ss.hash(token)
	var user User
	row := ss.DB.QueryRow(`
		SELECT 
			u.id,
			u.email,
			u.password_hash
		FROM sessions AS s
		JOIN users AS u
			ON s.user_id = u.id
		WHERE s.token_hash = $1;
	`, tokenHash)
	err := row.Scan(&user.ID, &user.Email, &user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("Erroring finding user: %w", err)
	}

	return &user, nil
}

// Delete a session
func (ss *SessionService) Delete(token string) error {
	tokenHash := ss.hash(token)
	_, err := ss.DB.Exec(`
		DELETE FROM sessions
		WHERE token_hash = $1
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("Error deleting session %w", err)
	}
	return nil
}

// Hash a given session token and return the hash
func (ss *SessionService) hash(token string) string {
	tokenHash := sha256.Sum256([]byte(token)) // This is an array not slice
	return base64.URLEncoding.EncodeToString(tokenHash[:])
}
