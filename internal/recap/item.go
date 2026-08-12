package recap

import (
	"time"
)

type Item struct {
	ID          int64     `json:"id"`
	RecapID     int64     `json:"recap_id"`
	Date        string    `json:"date"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Balance     *float64  `json:"balance"`
	CreatedAt   time.Time `json:"created_at"`
	Category    *string   `json:"category"`
}
