package repo

import "github.com/pkg/errors"

type User struct {
	UserID string
	Notes  []Note
}

type Note struct {
	ServiceName string
	Login       string
	Password    string
}

var (
	ErrNotFound = errors.New("not found")
)

type NotesRepo interface {
	Set(userID string, note Note) error
	Get(userID, serviceName string) (Note, error)
	Del(userID, serviceName string) error
}
