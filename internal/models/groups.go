package models

import (
	"database/sql"
)

type Group struct {
	ID   int
	Name string
}

type GroupModel struct {
	DB *sql.DB
}

func (m *GroupModel) AddUserToGroup(userId int, groupId int) error {
	stmt := "INSERT INTO user_groups (user_id, group_id) VALUES (?, ?);"

	_, err := m.DB.Exec(stmt, userId, groupId)
	if err != nil {
		return err
	}

	return nil
}
