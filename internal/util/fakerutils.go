package util

func PravatarURL(id string) string {
	return "https://i.pravatar.cc/150?u=" + id
}

func DicebearShapeAvatarURL(id string) string {
	return "https://api.dicebear.com/8.x/shapes/svg?seed=" + id
}

func DicebearFunEmojiAvatarURL(id string) string {
	return "https://api.dicebear.com/8.x/fun-emoji/svg?seed=" + id
}

func DicebearAvatarURL(id string) string {
	return "https://api.dicebear.com/8.x/avataaars-neutral/svg?seed=" + id
}

func DicebearIconAvatarURL(id string) string {
	return " https://api.dicebear.com/8.x/icons/svg?seed=" + id
}
