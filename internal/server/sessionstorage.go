package server

import (
	"github.com/gofiber/fiber/v2/middleware/session"
)

type SessionStorage struct {
	session *session.Session
}

func NewLogtoStorage(session *session.Session) *SessionStorage {
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
}

func (storage *SessionStorage) Session(key, value string) *session.Session {
	return storage.session
}

func (storage *SessionStorage) Save() {
	storage.session.Save()
}
