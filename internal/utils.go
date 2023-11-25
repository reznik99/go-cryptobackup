package internal

import (
	"fmt"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var Formatter = message.NewPrinter(language.English)

// ByteCountBinary returns stringified Bytesize such as '1.2 KiB' or '2.5 MiB'
func ByteCountBinary(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
