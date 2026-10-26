package url

import "time"

type ShortURL struct {
	ID          string
	Code        string
	OriginalURL string
	CreatedAt   time.Time
	Status      string
	TotalClicks int64
	LastClicked *time.Time
}
