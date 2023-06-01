package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/reznik99/go-cryptobackup/internal"
	log "github.com/sirupsen/logrus"

	"golang.org/x/crypto/pbkdf2"
)

type Config struct {
	Directories []string `json:"directories"`
}

func ParseConfig(configPath string) Config {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Unable to read config at: %s. Error: %s", configPath, err)
	}

	var config = Config{}
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		log.Fatalf("Unable to parse config: %s", err)
	}
	log.Println("Parsed config...")

	return config
}

func ParseFlags() (string, string) {
	var configPath = flag.String("config", "", "Absolute path to config file for backup/restore instructions")
	var passphrase = flag.String("passphrase", "", "Passphrase to encrypt backups")
	flag.Parse()

	if *configPath == "" || *passphrase == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	log.Println("Parsed flags...")
	return *configPath, *passphrase
}

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	// Parse Flags and Configuration
	var configPath, passphrase = ParseFlags()
	var config = ParseConfig(configPath)

	// Parse passphrase (if any) and generate encryption keys
	// TODO: Temporary IV
	// TODO: Temporary enc and mac keys are same
	var iv = []byte{0x52, 0x84, 0xf3, 0x22, 0x01, 0xff, 0x4f, 0x4a}
	encKey := pbkdf2.Key([]byte(passphrase), iv, 4096, 32, sha256.New)

	// Iterate over directories and backup/encrypt
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Unable to get user home directory: %s", err)
	}

	var backupDir = fmt.Sprintf("%s/backup-%s", homeDir, time.Now().Format("2006-01-02T15:04:05-0700"))
	err = internal.CreateIfNotExists(backupDir, 0755)
	if err != nil {
		log.Fatalf("Unable to create general backup directory: %s", err)
	}
	log.Infof("Backup directory: %s", backupDir)

	var start = time.Now()
	var bytesRead = int64(0)
	for _, dir := range config.Directories {
		tokens := strings.Split(dir, "/")
		targetDir := fmt.Sprintf("%s/%s", backupDir, tokens[len(tokens)-1])
		err = internal.CreateIfNotExists(targetDir, 0755)
		if err != nil {
			log.Errorf("Unable to create backup directory: %s", err)
			continue
		}
		read, err := internal.CopyDirectory(dir, targetDir, encKey, encKey)
		if err != nil {
			log.Errorf("Failed to backup %s: %s", dir, err)
			continue
		}
		bytesRead += read
	}

	// TODO: Save metadata to an backup.info file to allow decryption/restore

	// Log stats
	log.Info("Backup completed")
	var formattedRead = internal.ByteCountBinary(bytesRead)
	log.WithFields(log.Fields{
		"Read":  internal.Formatter.Sprint(formattedRead),
		"Speed": internal.Formatter.Sprintf("%s/s", internal.ByteCountBinary((bytesRead/time.Since(start).Milliseconds())*1000)),
		"Time":  internal.Formatter.Sprint(time.Since(start).Truncate(time.Millisecond * 10)),
	}).Info("Backup Statistics: ")
}
