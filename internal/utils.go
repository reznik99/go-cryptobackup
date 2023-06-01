package internal

import (
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var Formatter = message.NewPrinter(language.English)

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

func CreateBackupDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("unable to get user home directory: %s", err)
	}

	var backupDir = fmt.Sprintf("%s/backup-%s", homeDir, time.Now().Format("2006-01-02T15:04:05-0700"))
	err = CreateIfNotExists(backupDir, 0755)
	if err != nil {
		return "", fmt.Errorf("unable to create general backup directory: %s", err)
	}
	log.Infof("Backup directory: %s", backupDir)

	return backupDir, nil
}
