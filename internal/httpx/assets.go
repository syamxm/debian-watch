package httpx

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"net/http"
)

func assetVersion(fsys fs.FS) (string, error) {
	digest := sha256.New()

	err := fs.WalkDir(fsys, "static", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		file, err := fsys.Open(name)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := io.WriteString(digest, name); err != nil {
			return err
		}
		_, err = io.Copy(digest, file)
		return err
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil))[:12], nil
}

func staticHandler(fsys fs.FS, version string) http.Handler {
	fileServer := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("v") == version {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=60, must-revalidate")
		}
		fileServer.ServeHTTP(w, r)
	})
}
