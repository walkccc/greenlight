package data

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
)

// Permissions slice holds the permission codes (e.g. "movies:read" and
// "movies:write") for a single user.
type Permissions []string

// Include checks if the Permissions slice holds a specific permission code.
func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}

type PermissionModelInterface interface {
	AddForUser(userId int64, codes ...string) error
	GetAllForUser(userID int64) (Permissions, error)
}

type PermissionModel struct {
	DB *sql.DB
}

func (m PermissionModel) AddForUser(userID int64, codes ...string) error {
	query := `
		INSERT INTO "UsersPermissions"
		SELECT $1, "Permissions".id
		FROM "Permissions"
		WHERE "Permissions".code = ANY($2)`
	args := []any{
		userID,
		pq.Array(codes),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	return err
}

func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
		SELECT "Permissions".code
		FROM "Permissions"
		INNER JOIN "UsersPermissions"
			ON ("UsersPermissions".permission_id = "Permissions".id)
		INNER JOIN "Users"
			ON ("UsersPermissions".user_id = "Users".id)
		WHERE "Users".id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions Permissions

	for rows.Next() {
		var permission string
		err := rows.Scan(&permission)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return permissions, nil
}
