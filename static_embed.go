package main

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static
var embeddedStatic embed.FS

func prepareStaticAssets() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	base := filepath.Join(filepath.Dir(exe), ".crm-static")
	if err := os.MkdirAll(base, 0o755); err != nil {
		return "", err
	}
	if err := fs.WalkDir(embeddedStatic, "static", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel := strings.TrimPrefix(p, "static")
		rel = strings.TrimPrefix(rel, "/")
		target := filepath.Join(base, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		src, err := embeddedStatic.Open(p)
		if err != nil {
			return err
		}
		defer src.Close()
		dst, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			_ = dst.Close()
			return err
		}
		return dst.Close()
	}); err != nil {
		return "", fmt.Errorf("prepare static assets: %w", err)
	}
	return base, nil
}
