package commands

import (
	"os"
	"runtime"
)

var emojiSupport = true
var EmojiEnabled = true

func init() {
	// everything that is not windows usually has emoji support
	if runtime.GOOS != "windows" {
		return
	}

	// check if we are running in the windows terminal
	// (windows terminal does not set this, but raw cmd or powershell do)
	if os.Getenv("SESSIONNAME") != "" {
		// dang it, no emojis for you :(
		emojiSupport = false
	}
}

func EmojiSupported() bool {
	return emojiSupport
}

// Emoji returns the given string (usually a emoji) if the current terminal
// (probably) supports it
func Emoji(e string) string {
	if emojiSupport && EmojiEnabled {
		return e
	}
	return ""
}
