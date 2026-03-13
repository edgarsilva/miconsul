package auth

import (
	"github.com/gofiber/fiber/v3/middleware/session"
)

type LogtoStorage struct {
	session *session.Session
}

func NewLogtoStorage(session *session.Session) *LogtoStorage {
	s := LogtoStorage{
		session: session,
	}

	return &s
}

func (storage *LogtoStorage) GetItem(key string) string {
	if storage == nil || storage.session == nil {
		return ""
	}

	value := storage.session.Get(key)
	if value == nil {
		return ""
	}

	valueStr, ok := value.(string)
	if !ok {
		return ""
	}

	return valueStr
}

func (storage *LogtoStorage) SetItem(key, value string) {
	if storage == nil || storage.session == nil {
		return
	}

	storage.session.Set(key, value)
}

func (storage *LogtoStorage) Save() error {
	if storage == nil || storage.session == nil {
		return nil
	}

	return storage.session.Save()
}
