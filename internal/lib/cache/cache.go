package cache

import (
	"log"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
)

type Cache struct {
	*badger.DB
}

func New() (cacheRef *Cache, shutdownFn func()) {
	cacheOpts := badger.DefaultOptions(os.Getenv("CACHE_DB_PATH"))
	cachedb, err := badger.Open(cacheOpts)
	if err != nil {
		log.Fatal(err)
	}

	return &Cache{DB: cachedb}, func() {
		cachedb.Close()
	}
}

// Write writes a value to the Cache
// backed by BadgerDB
func (c *Cache) Write(key string, src *[]byte, ttl time.Duration) error {
	err := c.DB.Update(func(txn *badger.Txn) error {
		entry := badger.NewEntry([]byte(key), *src).WithTTL(ttl)
		err := txn.SetEntry(entry)
		return err
	})

	return err
}

// Read reads a cache value by key
func (c *Cache) Read(key string, dst *[]byte) error {
	err := c.DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			*dst = val
			return nil
		})

		return err
	})

	return err
}
