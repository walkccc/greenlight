package data

import (
	"crypto/sha256"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/walkccc/greenlight/internal/validator"
)

func TestGenerateToken(t *testing.T) {
	token, err := generateToken(int64(1), time.Hour, ScopeActivation)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), token.UserID)
	assert.Equal(t, ScopeActivation, token.Scope)
	assert.True(t, token.Expiry.After(time.Now()))
	assert.True(t, token.Expiry.Before(time.Now().Add(time.Hour)))
	assert.Equal(t, 26, len(token.Plaintext))
	tokenHash := sha256.Sum256([]byte(token.Plaintext))
	assert.Equal(t, tokenHash[:], token.Hash)
}

func TestValidateTokenPlaintext(t *testing.T) {
	t.Run("InvalidTokenPlaintext", func(t *testing.T) {
		tokenPlaintext := ""

		v := validator.New()
		ValidateTokenPlaintext(v, tokenPlaintext)
		assert.False(t, v.Valid())

		expectedErrors := map[string]string{
			"token": "must be provided",
		}
		for field, expectedMessage := range expectedErrors {
			actualMessage := v.Errors[field]
			assert.Equal(t, expectedMessage, actualMessage)
		}
	})

	t.Run("ValidTokenPlaintext", func(t *testing.T) {
		tokenPlaintext := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

		v := validator.New()
		ValidateTokenPlaintext(v, tokenPlaintext)
		assert.True(t, v.Valid())
	})
}

func TestTokenModel_Create(t *testing.T) {
	query := `
		INSERT INTO Tokens \(hash, user_id, expiry, scope\)
		VALUES \(\$1, \$2, \$3, \$4\)`
	token := &Token{
		Hash:   []byte{1, 2},
		UserID: 1,
		Expiry: time.Now().Add(time.Hour),
		Scope:  ScopeActivation,
	}

	tests := []struct {
		name       string
		buildMock  func(mock sqlmock.Sqlmock)
		checkModel func(model TokenModel)
	}{
		{
			name: "ErrConnDone",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(token.Hash, token.UserID, token.Expiry, token.Scope).
					WillReturnError(sql.ErrConnDone)
			},
			checkModel: func(model TokenModel) {
				err := model.Create(token)
				assert.Equal(t, sql.ErrConnDone, err)
			},
		},
		{
			name: "Success",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(token.Hash, token.UserID, token.Expiry, token.Scope).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			checkModel: func(model TokenModel) {
				err := model.Create(token)
				assert.Nil(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock := NewMock(t)
			model := TokenModel{DB: db}
			defer model.DB.Close()
			test.buildMock(mock)
			test.checkModel(model)
		})
	}
}

func TestTokenModel_DeleteAllForUser(t *testing.T) {
	query := `
		DELETE FROM Tokens
		WHERE scope = \$1 AND user_id = \$2`
	userID := int64(1)

	tests := []struct {
		name       string
		buildMock  func(mock sqlmock.Sqlmock)
		checkModel func(model TokenModel)
	}{
		{
			name: "ErrConnDone",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(ScopeActivation, userID).
					WillReturnError(sql.ErrConnDone)
			},
			checkModel: func(model TokenModel) {
				err := model.DeleteAllForUser(ScopeActivation, userID)
				assert.Equal(t, sql.ErrConnDone, err)
			},
		},
		{
			name: "Success",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(ScopeActivation, userID).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			checkModel: func(model TokenModel) {
				err := model.DeleteAllForUser(ScopeActivation, userID)
				assert.Nil(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock := NewMock(t)
			model := TokenModel{DB: db}
			defer model.DB.Close()
			test.buildMock(mock)
			test.checkModel(model)
		})
	}
}
