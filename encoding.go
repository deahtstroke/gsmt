package gsmt

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
)

func Hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func ReadFileContent(fs fs.FS, path string) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}

	bytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
