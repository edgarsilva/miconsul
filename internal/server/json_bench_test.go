package server

import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"testing"
	"time"

	"miconsul/internal/models"

	"github.com/gofiber/fiber/v3"
)

// Payloads mirror real JSON traffic: GET /api/auth/protected (models.User),
// the admin debug runtime map, and the POST /api/auth/signin body.
var (
	benchUser = struct{ User models.User }{User: models.User{
		ID: 42, UID: "user_d3k0a1b2c3d4e5f6g7h8", Name: "Ada Lovelace", Email: "ada@example.com",
		Theme: "dark", Phone: "+52 55 1234 5678", Timezone: "America/Mexico_City", Role: "admin",
		ModelBase: models.ModelBase{CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	benchMap = fiber.Map{
		"uptime_seconds": 86400, "goroutines": 42, "alloc_bytes": uint64(12_345_678),
		"total_alloc_bytes": uint64(987_654_321), "sys_bytes": uint64(45_678_901), "heap_objects": uint64(123_456),
	}
	benchSignin = []byte(`{"email":"ada@example.com","password":"correct horse battery staple","remember_me":true}`)
)

type benchSigninBody struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
}

func BenchmarkJSONEncodeUser(b *testing.B) {
	b.Run("v1", func(b *testing.B) {
		for b.Loop() {
			_, _ = jsonv1.Marshal(benchUser)
		}
	})
	b.Run("v2", func(b *testing.B) {
		for b.Loop() {
			_, _ = marshalJSON(benchUser)
		}
	})
}

func BenchmarkJSONEncodeMap(b *testing.B) {
	b.Run("v1", func(b *testing.B) {
		for b.Loop() {
			_, _ = jsonv1.Marshal(benchMap)
		}
	})
	b.Run("v2", func(b *testing.B) {
		for b.Loop() {
			_, _ = marshalJSON(benchMap)
		}
	})
	b.Run("v2-nondeterministic", func(b *testing.B) {
		for b.Loop() {
			_, _ = json.Marshal(benchMap)
		}
	})
}

func BenchmarkJSONDecodeSignin(b *testing.B) {
	b.Run("v1", func(b *testing.B) {
		for b.Loop() {
			var in benchSigninBody
			_ = jsonv1.Unmarshal(benchSignin, &in)
		}
	})
	b.Run("v2", func(b *testing.B) {
		for b.Loop() {
			var in benchSigninBody
			_ = unmarshalJSON(benchSignin, &in)
		}
	})
}
