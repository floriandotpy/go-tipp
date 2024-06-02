package models

import (
	"database/sql"
	"errors"
)

type Group struct {
	ID     int
	Name   string
	Invite string
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

func (m *GroupModel) All() ([]Group, error) {
	stmt := "SELECT id, name, invite FROM `groups` ORDER BY id ASC"
	rows, err := m.DB.Query(stmt)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var groups []Group

	for rows.Next() {
		var group Group
		err = rows.Scan(&group.ID, &group.Name, &group.Invite)
		if err != nil {
			return nil, err
		}

		groups = append(groups, group)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

func (m *GroupModel) AllForUser(userId int) ([]Group, error) {
	stmt := `SELECT 
    g.id AS group_id,
    g.name AS group_name 
FROM 
    ` + "`" + `groups` + "`" + ` g
JOIN 
    user_groups ug ON g.id = ug.group_id 
WHERE 
    ug.user_id = ?
ORDER BY 
    group_id ASC;`
	rows, err := m.DB.Query(stmt, userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var groups []Group

	for rows.Next() {
		var group Group
		err = rows.Scan(&group.ID, &group.Name)
		if err != nil {
			return nil, err
		}

		groups = append(groups, group)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return groups, nil
}

func (m *GroupModel) GetByInvite(invite string) (Group, error) {
	stmt := "SELECT id, name, invite FROM `groups` WHERE invite = ? ORDER BY id ASC"

	var group Group
	err := m.DB.QueryRow(stmt, invite).Scan(&group.ID, &group.Name, &group.Invite)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Group{}, ErrInvalidInvite
		} else {
			return Group{}, err
		}
	}

	return group, nil
}
