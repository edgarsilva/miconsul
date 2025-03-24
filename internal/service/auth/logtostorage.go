package auth

import (
	"github.com/gofiber/fiber/v2/middleware/session"
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
	value := storage.session.Get(key)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (storage *LogtoStorage) SetItem(key, value string) {
	storage.session.Set(key, value)
}

func (storage *LogtoStorage) Session(key, value string) *session.Session {
	return storage.session
}

func (storage *LogtoStorage) Save() {
	storage.session.Save()
}
