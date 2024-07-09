package userservice

import (
	"context"
	"database/sql"
)

func (m *DBModel) addUserPermission(tx *sql.Tx, ctx context.Context, id int, permissions ...Permission) error {
	// Add the permissions to the user
	for _, p := range permissions {
		_, err := tx.ExecContext(ctx, "INSERT INTO user_permissions (user_id, permission) VALUES ($1, $2)", id, p)
		if err != nil {
			return err
		}
	}

	return nil
}

// func (m *PermissionModel) get(ctx context.Context, id int) (*Permissions, error) {
// 	query := `
// 		SELECT permission
// 		FROM user_permissions
// 		WHERE user_id = $1`

// 	rows, err := m.db.QueryContext(ctx, query, id)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer rows.Close()

// 	var permissions Permissions
// 	for rows.Next() {
// 		var permission Permission

// 		err := rows.Scan(&permission)
// 		if err != nil {
// 			return nil, err
// 		}

// 		permissions = append(permissions, permission)
// 	}

// 	if err = rows.Err(); err != nil {
// 		return nil, err
// 	}

// 	return &permissions, nil
// }
