package recap

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("recap not found")

type Recap struct {
	ID        int64      `json:"id"`
	Name      string     `json:"name"`
	Status    string     `json:"status"`
	BankName  string     `json:"bank_name"`
	Period    string     `json:"period"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}

type CreateInput struct {
	Name     string
	BankName string
	Period   string
}
