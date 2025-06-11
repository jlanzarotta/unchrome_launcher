package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Construct the full path for the destination. Since the zip file we
		// are using has a subfolder, we need to remove the first directory as we
		// unzip it.
		fpath := filepath.Join(dest, removeFirstDir(f.Name))

		//log.Printf("fpath[%s]\n", fpath)

		// Prevent ZipSlip vulnerability
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			// Make directory
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
		} else {
			// Make parent directories
			if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			// Create the destination file
			dstFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer dstFile.Close()

			// Open the zip file content
			srcFile, err := f.Open()
			if err != nil {
				return err
			}
			defer srcFile.Close()

			// Copy the content
			if _, err := io.Copy(dstFile, srcFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func removeFirstDir(path string) string {
	// Clean path and split into parts
	cleaned := filepath.ToSlash(filepath.Clean(path))
	parts := strings.Split(cleaned, "/")

	if len(parts) <= 1 {
		return ""
	}

	// Join everything except the first element
	return filepath.Join(parts[1:]...)
}
