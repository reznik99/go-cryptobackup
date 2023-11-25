package internal

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func CopyDirectory(scrDir, dest string, encKey []byte, macKey []byte) (int64, error) {
	var totalRead = int64(0)
	entries, err := os.ReadDir(scrDir)
	if err != nil {
		return totalRead, err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			log.Error(err)
			continue
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := CreateIfNotExists(destPath, 0755); err != nil {
				log.Error(err)
				continue
			}
			read, err := CopyDirectory(sourcePath, destPath, encKey, macKey)
			if err != nil {
				log.Error(err)
				continue
			}
			totalRead += read
		case os.ModeSymlink:
			if err := CopySymLink(sourcePath, destPath); err != nil {
				log.Error(err)
				continue
			}
		default:
			_, err := Copy(sourcePath, destPath, encKey, macKey)
			if err != nil {
				log.Error(err)
				continue
			}
			totalRead += fileInfo.Size()
		}

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			log.Error(fmt.Errorf("failed to get raw syscall.Stat_t data for %q", sourcePath))
			continue
		}
		// Copy File ownership
		if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
			log.Error(err)
			continue
		}

		// Copy File permissions if not a symlink
		fInfo, err := entry.Info()
		if err != nil {
			log.Error(err)
			continue
		}

		isSymlink := fInfo.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, fInfo.Mode()); err != nil {
				log.Error(err)
				continue
			}
		}
	}
	return totalRead, nil
}

func Copy(srcFile, dstFile string, encKey []byte, macKey []byte) (*StreamMeta, error) {
	out, err := os.Create(dstFile)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	in, err := os.Open(srcFile)
	if err != nil {
		return nil, err
	}
	defer in.Close()

	inEncrypt, err := NewStreamEncrypter(encKey, macKey, in)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(out, inEncrypt)
	if err != nil {
		return nil, err
	}

	inStats, err := in.Stat()
	if err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{
		"name": inStats.Name(),
		"size": Formatter.Sprint(ByteCountBinary(inStats.Size())),
	}).Debug("- Copied file")

	meta := inEncrypt.Meta()

	return &meta, nil
}

func Exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func CreateIfNotExists(dir string, perm os.FileMode) error {
	if Exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory %q: %s", dir, err)
	}

	return nil
}

func CopySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}
