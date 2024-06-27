// Code generated. DO NOT EDIT.
package example

import (
	"time"

	"github.com/google/uuid"
)

func (n User) Id() uuid.UUID {
	return n.id
}

func (n User) Name() string {
	return n.name
}

func (n User) Age() int {
	return n.age
}

func (n User) Birthday() time.Time {
	return n.birthday
}

func (n User) Children() []User {
	return n.children
}

func (n User) CreatedAt() time.Time {
	return n.createdAt
}

func (n User) UpdatedAt() time.Time {
	return n.updatedAt
}
