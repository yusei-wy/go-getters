package example

import (
	"go/ast"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

//go:generate getters
type User struct {
	id        uuid.UUID
	name      string
	age       int
	birthday  time.Time
	children  []User
	createdAt time.Time
	updatedAt time.Time
}

type Ast struct {
	node *ast.Node
	num  int
}

func (a Ast) RandomNum() int {
	seed := time.Now().UnixNano()
	rand.Seed(seed)
	val := rand.Intn(100)

	return val
}
