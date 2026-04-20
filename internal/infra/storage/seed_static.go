package storage

import (
	"context"
	"io/fs"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
)

// ResolveStaticSeedDir returns the directory to scan for seed objects (typically repo-root `static/`).
// Empty means seeding is disabled or no directory was found.
func ResolveStaticSeedDir() string {
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("MINIO_SEED_STATIC"))); v == "0" || v == "false" || v == "no" {
		return ""
	}

	if d := strings.TrimSpace(os.Getenv("MINIO_STATIC_SEED_DIR")); d != "" {
		st, err := os.Stat(d)
		if err == nil && st.IsDir() {
			return filepath.Clean(d)
		}
		return ""
	}

	if wd, err := os.Getwd(); err == nil {
		p := filepath.Join(wd, "static")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return filepath.Clean(p)
		}
	}

	if exe, err := os.Executable(); err == nil {
		p := filepath.Join(filepath.Dir(exe), "static")
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			return filepath.Clean(p)
		}
	}

	return ""
}

// SeedStaticFromDir uploads files under root into the configured bucket using keys relative to root
// (e.g. static/posters/img1.jpg -> object key "posters/img1.jpg"). Skips hidden files and non-images.
// Objects that already exist are left unchanged.
func SeedStaticFromDir(ctx context.Context, svc Service, root string, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}
	ms, ok := svc.(*minioService)
	if !ok || root == "" {
		return nil
	}

	st, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			log.Debug("minio_static_seed_skip", "reason", "root_missing", "path", root)
			return nil
		}
		return err
	}
	if !st.IsDir() {
		return nil
	}

	var uploaded, skipped int
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		default:
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		objectName := filepath.ToSlash(rel)

		_, err = ms.client.StatObject(ctx, ms.bucket, objectName, minio.StatObjectOptions{})
		if err == nil {
			skipped++
			return nil
		}
		er := minio.ToErrorResponse(err)
		if er.Code != "NoSuchKey" && er.Code != "NotFound" {
			log.Warn("minio_static_seed_stat_failed", "object", objectName, "error", err)
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		fi, err := f.Stat()
		if err != nil {
			f.Close()
			return err
		}
		size := fi.Size()
		ct := mime.TypeByExtension(ext)
		if ct == "" {
			ct = "application/octet-stream"
		}
		if _, err := ms.UploadImage(ctx, objectName, f, size, ct); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		uploaded++
		log.Debug("minio_static_seed_uploaded", "object", objectName)
		return nil
	})
	if err != nil {
		return err
	}
	if uploaded > 0 || skipped > 0 {
		log.Info("minio_static_seed_done", "uploaded", uploaded, "skipped_existing", skipped, "root", root)
	}
	return nil
}
