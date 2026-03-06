package server

import "time"

// CacheWrite writes a value to the Cache.
func (s *Server) CacheWrite(key string, src *[]byte, ttl time.Duration) error {
	if s.Cache == nil {
		return nil
	}

	return s.Cache.Write(key, src, ttl)
}

// CacheRead reads a cache value by key.
func (s *Server) CacheRead(key string, dst *[]byte) error {
	if s.Cache == nil {
		return nil
	}

	return s.Cache.Read(key, dst)
}
