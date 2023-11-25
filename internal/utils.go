package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const ToolInfoFile = "bktool-backup.info"

var Formatter = message.NewPrinter(language.English)

type InfoFile struct {
	Salt []byte `json:"salt"`
}

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

func ParseInfoFile(path string) (*InfoFile, error) {
	infoFile, err := os.ReadFile(filepath.Join(path, ToolInfoFile))
	if err != nil {
		return nil, err
	}

	backupInfo := &InfoFile{}
	err = json.Unmarshal(infoFile, backupInfo)
	if err != nil {
		return nil, err
	}

	return backupInfo, nil
}

func MarshalInfoFile(info *InfoFile) ([]byte, error) {
	return json.Marshal(info)
}
