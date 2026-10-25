package domain

// NameOrderKey folds only ASCII capitals. UTF-8 byte ordering of valid
// Unicode then agrees with codepoint order and PostgreSQL COLLATE C.
func NameOrderKey(name string) string {
	key := []byte(name)
	for i, b := range key {
		if b >= 'A' && b <= 'Z' {
			key[i] = b + ('a' - 'A')
		}
	}
	return string(key)
}
