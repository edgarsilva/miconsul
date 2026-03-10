package avatar

import "testing"

func TestAvatarURLs(t *testing.T) {
	id := "usr_123"

	t.Run("PravatarURL", func(t *testing.T) {
		want := "https://i.pravatar.cc/150?u=" + id
		if got := PravatarURL(id); got != want {
			t.Fatalf("PravatarURL() = %q, want %q", got, want)
		}
	})

	t.Run("DicebearShapeAvatarURL", func(t *testing.T) {
		want := "https://api.dicebear.com/8.x/shapes/svg?seed=" + id
		if got := DicebearShapeAvatarURL(id); got != want {
			t.Fatalf("DicebearShapeAvatarURL() = %q, want %q", got, want)
		}
	})

	t.Run("DicebearFunEmojiAvatarURL", func(t *testing.T) {
		want := "https://api.dicebear.com/8.x/fun-emoji/svg?seed=" + id
		if got := DicebearFunEmojiAvatarURL(id); got != want {
			t.Fatalf("DicebearFunEmojiAvatarURL() = %q, want %q", got, want)
		}
	})

	t.Run("DicebearAvatarURL", func(t *testing.T) {
		want := "https://api.dicebear.com/8.x/avataaars-neutral/svg?seed=" + id
		if got := DicebearAvatarURL(id); got != want {
			t.Fatalf("DicebearAvatarURL() = %q, want %q", got, want)
		}
	})

	t.Run("DicebearIconAvatarURL", func(t *testing.T) {
		want := "https://api.dicebear.com/8.x/icons/svg?seed=" + id
		if got := DicebearIconAvatarURL(id); got != want {
			t.Fatalf("DicebearIconAvatarURL() = %q, want %q", got, want)
		}
	})
}
