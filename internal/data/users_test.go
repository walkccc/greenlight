package data

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/walkccc/greenlight/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

func TestIsAnonymous(t *testing.T) {
	user := AnonymousUser
	assert.True(t, user.IsAnonymous())
}

var plaintextPassword = "pa55word"

func TestSet(t *testing.T) {
	p := &password{}

	err := p.Set(plaintextPassword)
	assert.Nil(t, err)
	assert.Equal(t, plaintextPassword, *p.plaintext)
	assert.Nil(t, bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword)))
}

func TestMatches(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	p := &password{hash: hash}

	t.Run("MatchPassword", func(t *testing.T) {
		match, err := p.Matches(plaintextPassword)
		assert.Nil(t, err)
		assert.True(t, match)
	})

	t.Run("NotMatchPassword", func(t *testing.T) {
		match, err := p.Matches("wrongpassword")
		assert.Nil(t, err)
		assert.False(t, match)
	})

	t.Run("HashTooShort", func(t *testing.T) {
		invalidHash := []byte("invalidhash")
		err := bcrypt.CompareHashAndPassword(invalidHash, []byte(plaintextPassword))
		assert.Equal(t, bcrypt.ErrHashTooShort, err)
	})
}

func TestValidateEmail(t *testing.T) {
	t.Run("InvalidEmail", func(t *testing.T) {
		email := ""

		v := validator.New()
		ValidateEmail(v, email)
		assert.False(t, v.Valid())

		expectedErrors := map[string]string{
			"email": "must be provided",
		}
		for field, expectedMessage := range expectedErrors {
			actualMessage := v.Errors[field]
			assert.Equal(t, expectedMessage, actualMessage)
		}
	})

	t.Run("ValidEmail", func(t *testing.T) {
		email := "test@greenlight.com"

		v := validator.New()
		ValidateEmail(v, email)
		assert.True(t, v.Valid())
	})
}

func TestValidatePasswordPlaintext(t *testing.T) {
	t.Run("InvalidPassword", func(t *testing.T) {
		shortPassword := "pass"

		v := validator.New()
		ValidatePasswordPlaintext(v, shortPassword)
		assert.False(t, v.Valid())

		expectedErrors := map[string]string{
			"password": "must be at least 8 bytes long",
		}
		for field, expectedMessage := range expectedErrors {
			actualMessage := v.Errors[field]
			assert.Equal(t, expectedMessage, actualMessage)
		}
	})

	t.Run("ValidPassword", func(t *testing.T) {
		password := "pa55word"

		v := validator.New()
		ValidatePasswordPlaintext(v, password)
		assert.True(t, v.Valid())
	})
}

func TestValidateUser(t *testing.T) {
	t.Run("InvalidUser", func(t *testing.T) {
		shortPassword := "pass"
		hash, _ := bcrypt.GenerateFromPassword([]byte(shortPassword), 12)
		user := &User{
			Name:     "", // Invalid: empty name
			Email:    "", // Invalid: empty email
			Password: password{plaintext: &shortPassword, hash: hash},
		}

		v := validator.New()
		ValidateUser(v, user)
		assert.False(t, v.Valid())

		expectedErrors := map[string]string{
			"name":     "must be provided",
			"email":    "must be provided",
			"password": "must be at least 8 bytes long",
		}
		for field, expectedMessage := range expectedErrors {
			actualMessage := v.Errors[field]
			assert.Equal(t, expectedMessage, actualMessage)
		}
	})

	t.Run("ValidUser", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
		user := &User{
			Name:     "Test",
			Email:    "test@greenlight.com",
			Password: password{plaintext: &plaintextPassword, hash: hash},
		}

		v := validator.New()
		ValidateUser(v, user)
		assert.True(t, v.Valid())
	})
}

func TestUserlModel_Create(t *testing.T) {
	query := `
		INSERT INTO Users \(name, email, password_hash, activated\)
		VALUES \(\$1, \$2, \$3, \$4\)
		RETURNING id, created_at, version`
	passwordHash := make([]byte, 10)
	activated := false
	createdAt := time.Now()
	user := &User{
		ID:        1,
		CreatedAt: createdAt,
		Name:      "Jay",
		Email:     "jay@greenlight.com",
		Password:  password{hash: passwordHash},
		Activated: activated,
		Version:   1,
	}

	tests := []struct {
		name       string
		buildMock  func(mock sqlmock.Sqlmock)
		checkModel func(model UserModel)
	}{
		{
			name: "Success",
			buildMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(
					[]string{"id", "created_at", "version"}).
					AddRow(1, createdAt, 1)
				mock.ExpectQuery(query).
					WithArgs("Jay", "jay@greenlight.com", passwordHash, activated).
					WillReturnRows(rows)
			},
			checkModel: func(model UserModel) {
				err := model.Create(user)
				assert.Nil(t, err)
			},
		},
		{
			name: "DuplicateEmail",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs("Jay", "jay@greenlight.com", passwordHash, activated).
					WillReturnError(errors.New(`pq: duplicate key value violates unique constraint "users_email_key"`))
			},
			checkModel: func(model UserModel) {
				err := model.Create(user)
				assert.Equal(t, ErrDuplicateEmail, err)
			},
		},
		{
			name: "ErrConnDone",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs("Jay", "jay@greenlight.com", passwordHash, activated).
					WillReturnError(sql.ErrConnDone)
			},
			checkModel: func(model UserModel) {
				err := model.Create(user)
				assert.Equal(t, sql.ErrConnDone, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock := NewMock(t)
			model := UserModel{DB: db}
			defer model.DB.Close()
			test.buildMock(mock)
			test.checkModel(model)
		})
	}
}

func TestMUserModel_GetByEmail(t *testing.T) {
	query := `
		SELECT
			id,
			created_at,
			name,
			email,
			password_hash,
			activated,
			version
		FROM Users
		WHERE email = \$1`
	createdAt := time.Now()

	tests := []struct {
		name       string
		buildMock  func(mock sqlmock.Sqlmock)
		checkModel func(model UserModel)
	}{
		{
			name: "Success",
			buildMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.
					NewRows(
						[]string{
							"id",
							"created_at",
							"name",
							"email",
							"hash",
							"activated",
							"version",
						},
					).
					AddRow(1, createdAt, "Jay", "jay@greenlight.com", make([]byte, 10), false, 1)
				mock.ExpectQuery(query).WithArgs("jay@greenlight.com").WillReturnRows(rows)
			},
			checkModel: func(model UserModel) {
				user, err := model.GetByEmail("jay@greenlight.com")
				assert.NotNil(t, user)
				assert.Nil(t, err)
				assert.Equal(t, int64(1), user.ID)
				assert.Equal(t, createdAt, user.CreatedAt)
				assert.Equal(t, "Jay", user.Name)
				assert.Equal(t, "jay@greenlight.com", user.Email)
				assert.Equal(t, make([]byte, 10), user.Password.hash)
				assert.Equal(t, false, user.Activated)
				assert.Equal(t, 1, user.Version)
			},
		},
		{
			name: "ErrNoRows",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs("jay@greenlight.com").
					WillReturnError(sql.ErrNoRows)
			},
			checkModel: func(model UserModel) {
				user, err := model.GetByEmail("jay@greenlight.com")
				assert.Nil(t, user)
				assert.Equal(t, ErrRecordNotFound, err)
			},
		},
		{
			name: "ErrConnDone",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs("jay@greenlight.com").
					WillReturnError(sql.ErrConnDone)
			},
			checkModel: func(model UserModel) {
				user, err := model.GetByEmail("jay@greenlight.com")
				assert.Nil(t, user)
				assert.Equal(t, sql.ErrConnDone, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock := NewMock(t)
			model := UserModel{DB: db}
			defer model.DB.Close()
			test.buildMock(mock)
			test.checkModel(model)
		})
	}
}

func TestMUserModel_GetForToken(t *testing.T) {
	tokenScope := ScopeActivation
	tokenPlaintext := "TOKENPLAINTEXT"
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
		SELECT
			Users.id,
			Users.created_at,
			Users.name,
			Users.email,
			Users.password_hash,
			Users.activated,
			Users.version
		FROM Users
		INNER JOIN Tokens
			ON \(Users\.id = Tokens\.user_id\)
		WHERE
			Tokens\.hash = \$1
			AND Tokens\.scope = \$2
			AND Tokens\.expiry > \$3`
	createdAt := time.Now()

	tests := []struct {
		name       string
		buildMock  func(mock sqlmock.Sqlmock)
		checkModel func(model UserModel)
	}{
		{
			name: "Success",
			buildMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.
					NewRows(
						[]string{
							"id",
							"created_at",
							"name",
							"email",
							"hash",
							"activated",
							"version",
						},
					).
					AddRow(1, createdAt, "Jay", "jay@greenlight.com", make([]byte, 10), false, 1)
				mock.ExpectQuery(query).
					WithArgs(tokenHash[:], tokenScope, sqlmock.AnyArg()).
					WillReturnRows(rows)
			},
			checkModel: func(model UserModel) {
				user, err := model.GetForToken(tokenScope, tokenPlaintext)
				assert.NotNil(t, user)
				assert.Nil(t, err)
				assert.Equal(t, int64(1), user.ID)
				assert.Equal(t, createdAt, user.CreatedAt)
				assert.Equal(t, "Jay", user.Name)
				assert.Equal(t, "jay@greenlight.com", user.Email)
				assert.Equal(t, make([]byte, 10), user.Password.hash)
				assert.Equal(t, false, user.Activated)
				assert.Equal(t, 1, user.Version)
			},
		},
		{
			name: "ErrNoRows",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(tokenHash[:], tokenScope, sqlmock.AnyArg()).
					WillReturnError(sql.ErrNoRows)
			},
			checkModel: func(model UserModel) {
				user, err := model.GetForToken(tokenScope, tokenPlaintext)
				assert.Nil(t, user)
				assert.Equal(t, ErrRecordNotFound, err)
			},
		},
		{
			name: "ErrConnDone",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(tokenHash[:], tokenScope, sqlmock.AnyArg()).
					WillReturnError(sql.ErrConnDone)
			},
			checkModel: func(model UserModel) {
				user, err := model.GetForToken(tokenScope, tokenPlaintext)
				assert.Nil(t, user)
				assert.Equal(t, sql.ErrConnDone, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock := NewMock(t)
			model := UserModel{DB: db}
			defer model.DB.Close()
			test.buildMock(mock)
			test.checkModel(model)
		})
	}
}
