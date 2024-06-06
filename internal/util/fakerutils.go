package util

func PravatarURL(id string) string {
	return "https://i.pravatar.cc/150?u=" + id
}

func DicebearShapeAvatarURL(id string) string {
	return "https://api.dicebear.com/8.x/shapes/svg?seed=" + id
}
