// Package emoji provides helper functions for emoji handling
package emoji

import "slices"

// IsEmoji returns whether an emoji is a Discord default emoji.
func IsEmoji(str string) bool {
	return slices.Contains(emojis, str)
}
