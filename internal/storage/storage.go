package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Local struct{ Root string }

type Store interface {
	Save(context.Context, io.Reader, string, int64) (string, int64, error)
	Open(string) (*os.File, error)
	Remove(string) error
}

func (s Local) Save(ctx context.Context, src io.Reader, name string, max int64) (string, int64, error) {
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	ext := strings.ToLower(filepath.Ext(name))
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", 0, err
	}
	key := hex.EncodeToString(raw) + ext
	if err := os.MkdirAll(s.Root, 0o750); err != nil {
		return "", 0, err
	}
	path := filepath.Join(s.Root, key)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(src, max+1))
	if err != nil {
		_ = os.Remove(path)
		return "", 0, err
	}
	if n > max {
		_ = os.Remove(path)
		return "", 0, fmt.Errorf("file too large")
	}
	return key, n, nil
}

func (s Local) Open(key string) (*os.File, error) {
	if key == "" || filepath.Base(key) != key || strings.Contains(key, "..") {
		return nil, os.ErrInvalid
	}
	return os.Open(filepath.Join(s.Root, key))
}

func (s Local) Remove(key string) error {
	if key == "" || filepath.Base(key) != key {
		return os.ErrInvalid
	}
	return os.Remove(filepath.Join(s.Root, key))
}
