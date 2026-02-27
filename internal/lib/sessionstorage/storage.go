package sessionstorage

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type SessionStorage struct {
	*badger.DB
}

func New() *SessionStorage {
	cacheOpts := badger.DefaultOptions(os.Getenv("SESSION_DB_PATH"))
	sessiondb, err := badger.Open(cacheOpts)
	if err != nil {
		log.Fatal(err)
	}

	return &SessionStorage{
		DB: sessiondb,
	}
}

func (s SessionStorage) Get(key string) ([]byte, error) {
	var dst []byte
	err := s.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			dst = val
			return nil
		})

		return err
	})

	if errors.Is(err, badger.ErrKeyNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return dst, nil
}

func (s SessionStorage) Set(key string, val []byte, exp time.Duration) error {
	err := s.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), val).WithTTL(exp)
		err := txn.SetEntry(entry)
		return err
	})

	return err
}

func (s SessionStorage) GetWithContext(_ context.Context, key string) ([]byte, error) {
	return s.Get(key)
}

func (s SessionStorage) SetWithContext(_ context.Context, key string, val []byte, exp time.Duration) error {
	return s.Set(key, val, exp)
}

func (s SessionStorage) Delete(key string) error {
	err := s.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		return err
	})

	return err
}

func (s SessionStorage) DeleteWithContext(_ context.Context, key string) error {
	return s.Delete(key)
}

// Reset resets the storage and delete all keys.
func (s SessionStorage) Reset() error {
	// Drop all keys from the database
	err := s.DropAll()
	if err != nil {
		return err
	}

	return nil
}

func (s SessionStorage) ResetWithContext(_ context.Context) error {
	return s.Reset()
}

func (s SessionStorage) Close() error {
	err := s.DB.Close()
	if err != nil {
		return err
	}

	return nil
}
