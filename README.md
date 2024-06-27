# go-getters

Automatic Getter Generation for Go Structs.

This tool automatically generates getter methods for Go structs annotated with //go:generate getters.

## Features:
- Generates getter methods for struct fields
- Output filename follows the format filename_getters.go

## Install

```
$ go install github.com/yusei-wy/go-getters
```

## Usage:

1. Annotate struct fields with //go:generate getters comment tag.
  ```go
  //go:generate getters
  type MyStruct struct {
      field1 string
      field2 int
  }
  ```
1. Run go generate in the directory containing the struct definition.

## TODO:

- [ ] Support fields with return types of func and struct.
- [ ] Allow generating getters with reference types.

## Example:

```go
package example

import (
	"go/ast"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

//go:generate getters
type User struct {
	id       uuid.UUID
	name     string
	age      int
	birthday time.Time
	children []User
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
```

Running go generate will generate the following file:

```go
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
```

## Disclaimer:

This tool is provided as-is and is not intended for production use. It is still under development and may contain bugs or limitations.

Feel free to contribute or provide feedback!
