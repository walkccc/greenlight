package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"time"

	"github.com/walkccc/greenlight/internal/validator"
)

// Constants for the token scope.
const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

// Token holds the data for an individual token.
type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	// Use the Read() function from the crypto/rand package to fill the byte slice
	// with random bytes from your OS's CSPRNG (Cryptographically Secure
	// Pseudo-Random Number Generator).
	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	// Encode the byte slice to a base-32-encoded string and assign it to the
	// token Plaintext field. This will be the token string that we send to the
	// user in their welcome email. They will look similar to this:
	// Y3QMGX3PJ3WLRL2YRTQGQ6KRHU.
	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)

	// Generate a SHA-256 hash of the plaintext token string. This will be the
	// value stored in the db. Note that the sha256.Sum256() function returns an
	// ARRAY of length 32, so to make it easier to work with we convert it to a
	// slice using the [:] operator befor storing it.
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]
	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModelInterface interface {
	New(userID int64, ttl time.Duration, scope string) (*Token, error)
	Create(token *Token) error
	DeleteAllForUser(scope string, userID int64) error
}

type TokenModel struct {
	DB *sql.DB
}

// New creates a new Token struct and then inserts the data in the tokens table.
func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Create(token)
	return token, err
}

func (m TokenModel) Create(token *Token) error {
	query := `
		INSERT INTO "Tokens" (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)`
	args := []any{
		token.Hash,
		token.UserID,
		token.Expiry,
		token.Scope,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

// DeleteAllForUsers deletes all tokens for a specific user and scope.
func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
		DELETE FROM "Tokens"
		WHERE scope = $1 AND user_id = $2`
	args := []any{
		scope,
		userID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}
