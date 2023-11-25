package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/reznik99/go-cryptobackup/internal"
	log "github.com/sirupsen/logrus"
)

type Config struct {
	Directories     []string `json:"directories"`
	BackupDirectory string   `json:"backup_directory"`
	DateBackups     bool     `json:"date_backups"`
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

func ParseFlags() (string, string, bool, bool) {
	var configPath = flag.String("conf", "", "Absolute path to config file for backup/restore instructions")
	var passphrase = flag.String("pass", "", "Passphrase to encrypt backups")
	var decrypt = flag.Bool("restore", false, "Passphrase to encrypt backups")
	var verbose = flag.Bool("v", false, "Passphrase to encrypt backups")
	flag.Parse()

	if *configPath == "" || *passphrase == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	return *configPath, *passphrase, *verbose, *decrypt
}

func main() {
	// Init logger
	log.SetFormatter(&log.TextFormatter{})
	log.SetOutput(os.Stdout)

	// Parse Flags and Configuration
	log.Info("📝 Parsing CLI flags...")
	var configPath, passphrase, verbose, decrypt = ParseFlags()
	log.Info("📝 Parsing config...")
	var config = ParseConfig(configPath)

	if verbose {
		log.SetLevel(log.DebugLevel)
		log.Debug("📝 Verbose logging enbabled...")
	}

	if decrypt {
		DoDecrypt(config, passphrase)
	} else {
		DoEncrypt(config, passphrase)
	}

}

func DoEncrypt(config Config, passphrase string) {
	// Derive encryption keys
	log.Info("🔑 Deriving encryption keys...")
	encKey, salt, err := internal.DeriveKey(passphrase, nil, 32)
	if err != nil {
		log.Fatalf("Unable to derive ENC key: %s", err)
	}

	encrypter, err := internal.NewEncrypter(encKey, internal.Encrypt)
	if err != nil {
		log.Fatal(err)
	}

	// Create backup directory
	log.Printf("📂 Creating backup directory...")
	backupDir := config.BackupDirectory
	if config.DateBackups {
		backupDir = filepath.Join(config.BackupDirectory, fmt.Sprintf("%d", time.Now().UnixMilli()))
	}
	if err := internal.CreateIfNotExists(backupDir, 0755); err != nil {
		log.Fatal(fmt.Errorf("unable to create general backup directory: %s", err))
	}

	// Write info file
	err = internal.WriteInfoFile(&internal.InfoFile{Salt: salt}, backupDir)
	if err != nil {
		log.Fatal(err)
	}

	var bytesRead = int64(0)
	var start = time.Now()
	for _, sourceDir := range config.Directories {
		targetDir := filepath.Join(backupDir, filepath.Base(sourceDir))
		if err := internal.CreateIfNotExists(targetDir, 0755); err != nil {
			log.Errorf("Unable to create backup directory %q: %s", targetDir, err)
			continue
		}

		log.WithField("Source", sourceDir).Infof("🔐 Backing up")
		read, err := internal.CopyDirectory(sourceDir, targetDir, encrypter)
		if err != nil {
			log.Errorf("Failed to backup %q: %s", sourceDir, err)
			continue
		}
		bytesRead += read
	}

	// Log Statistics
	bytesPerSeconds := (bytesRead / time.Since(start).Milliseconds()) * 1000
	statistics := log.Fields{
		"Read":  internal.Formatter.Sprint(internal.ByteCountBinary(bytesRead)),
		"Speed": internal.Formatter.Sprintf("%s/s", internal.ByteCountBinary(bytesPerSeconds)),
		"Time":  internal.Formatter.Sprint(time.Since(start).Truncate(time.Millisecond * 10)),
	}
	log.WithField("Dir", backupDir).Info("✅ Encrypt/Backup completed")
	log.WithFields(statistics).Info("💡 Backup Statistics")
	log.Infof("Key: %X", encKey)
}

func DoDecrypt(config Config, passphrase string) {
	// Get salt for KDF from backup info file
	backupInfo, err := internal.ReadInfoFile(config.BackupDirectory)
	if err != nil {
		log.Fatal(err)
	}

	// Derive encryption keys
	log.Info("🔑 Deriving encryption keys...")
	encKey, _, err := internal.DeriveKey(passphrase, backupInfo.Salt, 32)
	if err != nil {
		log.Fatalf("Unable to derive ENC key: %s", err)
	}

	encrypter, err := internal.NewEncrypter(encKey, internal.Decrypt)
	if err != nil {
		log.Fatal(err)
	}

	entries, err := os.ReadDir(config.BackupDirectory)
	if err != nil {
		log.Fatal(err)
	}

	restoreDir := filepath.Join(filepath.Dir(config.BackupDirectory), "bktool-restored")

	var bytesRead = int64(0)
	var start = time.Now()
	for _, entry := range entries {
		if entry.Name() == internal.ToolInfoFile {
			continue
		}
		sourceDir := filepath.Join(config.BackupDirectory, entry.Name())
		targetDir := filepath.Join(restoreDir, entry.Name())
		if err := internal.CreateIfNotExists(targetDir, 0755); err != nil {
			log.Errorf("Unable to create backup directory %q: %s", targetDir, err)
			continue
		}

		log.WithField("Source", sourceDir).Infof("🔐 Restoring")
		read, err := internal.CopyDirectory(sourceDir, targetDir, encrypter)
		if err != nil {
			log.Errorf("Failed to backup %q: %s", sourceDir, err)
			continue
		}
		bytesRead += read
	}

	// Log Statistics
	bytesPerSeconds := (bytesRead / time.Since(start).Milliseconds()) * 1000
	statistics := log.Fields{
		"Read":  internal.Formatter.Sprint(internal.ByteCountBinary(bytesRead)),
		"Speed": internal.Formatter.Sprintf("%s/s", internal.ByteCountBinary(bytesPerSeconds)),
		"Time":  internal.Formatter.Sprint(time.Since(start).Truncate(time.Millisecond * 10)),
	}
	log.WithField("Dir", restoreDir).Info("✅ Decrypt/Restore completed")
	log.WithFields(statistics).Info("💡 Backup Statistics")
	log.Infof("Key: %X", encKey)
}
