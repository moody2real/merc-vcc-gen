package session

import (
	"encoding/json"
	"os"
	"time"
)

type Session struct {
	Cookie  string    `json:"cookie"`
	XSRF    string    `json:"xsrf"`
	SavedAt time.Time `json:"savedAt"`
}

func Load(path string) (*Session, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (s *Session) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}
