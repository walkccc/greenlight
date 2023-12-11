package data

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestPermissionsInclude(t *testing.T) {
	permissions := Permissions{"movies:read"}
	assert.True(t, permissions.Include("movies:read"))
	assert.False(t, permissions.Include("movies:write"))
}

func TestPermissionMovel_AddForUser(t *testing.T) {
	query := `
		INSERT INTO UsersPermissions
		SELECT \$1, permissions\.id
		FROM Permissions
		WHERE Permissions.code = ANY\(\$2\)`

	tests := []struct {
		name       string
		buildMock  func(mock sqlmock.Sqlmock)
		checkModel func(model PermissionModel)
	}{
		{
			name: "ErrConnDone",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(1, pq.Array([]string{"movies:read", "movies:write"})).
					WillReturnError(sql.ErrConnDone)
			},
			checkModel: func(model PermissionModel) {
				err := model.AddForUser(1, "movies:read", "movies:write")
				assert.Equal(t, sql.ErrConnDone, err)
			},
		},
		{
			name: "Success",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(query).
					WithArgs(1, pq.Array([]string{"movies:read", "movies:write"})).
					WillReturnResult(sqlmock.NewResult(1, 2))
			},
			checkModel: func(model PermissionModel) {
				err := model.AddForUser(1, "movies:read", "movies:write")
				assert.Nil(t, err)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock := NewMock(t)
			model := PermissionModel{DB: db}
			defer model.DB.Close()
			test.buildMock(mock)
			test.checkModel(model)
		})
	}
}

func TestPermissionMovel_GetAllForUser(t *testing.T) {
	query := `
		SELECT Permissions\.code
		FROM Permissions
		INNER JOIN UsersPermissions
			ON \(UsersPermissions\.permission_id = Permissions\.id\)
		INNER JOIN Users
			ON \(UsersPermissions\.user_id = Users\.id\)
		WHERE Users\.id = \$1`

	tests := []struct {
		name       string
		buildMock  func(mock sqlmock.Sqlmock)
		checkModel func(model PermissionModel)
	}{
		{
			name: "ErrConnDone",
			buildMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},
			checkModel: func(model PermissionModel) {
				permissions, err := model.GetAllForUser(1)
				assert.Nil(t, permissions)
				assert.Equal(t, sql.ErrConnDone, err)
			},
		},
		{
			name: "Success",
			buildMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"permission"}).
					AddRow("movies:read").
					AddRow("movies:write")
				mock.ExpectQuery(query).
					WithArgs(1).
					WillReturnRows(rows)
			},
			checkModel: func(model PermissionModel) {
				permissions, err := model.GetAllForUser(1)
				assert.Nil(t, err)
				assert.Equal(t, permissions, Permissions{"movies:read", "movies:write"})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db, mock := NewMock(t)
			model := PermissionModel{DB: db}
			defer model.DB.Close()
			test.buildMock(mock)
			test.checkModel(model)
		})
	}
}
