package common

import (
	"github.com/gofiber/fiber/v2/middleware/session"
)

type SessionStorage struct {
	session *session.Session
}

func NewSessionStorage(session *session.Session) *SessionStorage {
	s := SessionStorage{
		session: session,
	}

	return &s
}

func (storage *SessionStorage) GetItem(key string) string {
	value := storage.session.Get(key)
	if value == nil {
		return ""
	}
	return value.(string)
}

func (storage *SessionStorage) SetItem(key, value string) {
	storage.session.Set(key, value)

	storage.session.Save()
}
