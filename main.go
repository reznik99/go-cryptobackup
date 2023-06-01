package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/reznik99/go-cryptobackup/internal"
	log "github.com/sirupsen/logrus"
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

	return config
}

func ParseFlags() (string, string, bool) {
	var configPath = flag.String("conf", "", "Absolute path to config file for backup/restore instructions")
	var passphrase = flag.String("pass", "", "Passphrase to encrypt backups")
	var verbose = flag.Bool("v", false, "Passphrase to encrypt backups")
	flag.Parse()

	if *configPath == "" || *passphrase == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	return *configPath, *passphrase, *verbose
}

func main() {
	// Init logger
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	// Parse Flags and Configuration
	log.Println("Parsing CLI flags...")
	var configPath, passphrase, verbose = ParseFlags()
	log.Println("Parsing config...")
	var config = ParseConfig(configPath)

	if verbose {
		log.SetLevel(log.DebugLevel)
		log.Debug("Verbose logging enbabled...")
	}

	// Derive encryption keys
	log.Println("Deriving encryption keys...")
	encKey, err := internal.DeriveKey(passphrase)
	if err != nil {
		log.Fatalf("Unable to derive ENC key: %s", err)
	}
	macKey, err := internal.DeriveKey(passphrase)
	if err != nil {
		log.Fatalf("Unable to derive MAC key: %s", err)
	}

	// Create backup directory
	log.Println("Creating backup directory...")
	backupDir, err := internal.CreateBackupDirectory()
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Handle Decrypting (and maybe restoring the backup)

	// Iterate over directories and encrypt/backup
	var start = time.Now()
	var bytesRead = int64(1)
	for _, dir := range config.Directories {
		tokens := strings.Split(dir, "/")
		targetDir := fmt.Sprintf("%s/%s", backupDir, tokens[len(tokens)-1])
		err = internal.CreateIfNotExists(targetDir, 0755)
		if err != nil {
			log.Errorf("Unable to create backup directory: %s", err)
			continue
		}
		log.Infof("Backing up: %s ...", dir)
		read, err := internal.CopyDirectory(dir, targetDir, encKey, macKey)
		if err != nil {
			log.Errorf("Failed to backup %s: %s", dir, err)
			continue
		}
		bytesRead += read
	}

	// TODO: Save metadata to an backup.info file to allow decryption/restore

	// Log Statistics
	var formattedRead = internal.ByteCountBinary(bytesRead)
	logFields := log.Fields{
		"Read":  internal.Formatter.Sprint(formattedRead),
		"Speed": internal.Formatter.Sprintf("%s/s", internal.ByteCountBinary((bytesRead/time.Since(start).Milliseconds())*1000)),
		"Time":  internal.Formatter.Sprint(time.Since(start).Truncate(time.Millisecond * 10)),
	}
	log.Info("Backup completed")
	log.WithFields(logFields).Info("Backup Statistics: ")
}
