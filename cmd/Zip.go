package cmd

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unchrome_launcher/constants"

	"github.com/bodgit/sevenzip"
	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

func unzip(src string, dest string) error {
	if strings.HasSuffix(src, ".7z") {
    	r, err := sevenzip.OpenReader(src)
    	if err != nil {
			log.Fatalf("%s: %v\n",
				color.RedString(constants.FATAL_NORMAL_CASE), err)
    	    return err
    	}
    	defer r.Close()

		bar := progressbar.NewOptions(
			len(r.File),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription("unzipping"))

    	for _, f := range r.File {
			bar.Add(1)
			// Construct the full path for the destination. Since the zip file we
			// are using has a subfolder, we need to remove the first directory as we
			// unzip it.
			if removeFirstDir(f.Name) == constants.EMPTY {
				continue
			}

			outPath := filepath.Join(dest, removeFirstDir(f.Name))

			// Prevent ZipSlip vulnerability.
			if !strings.HasPrefix(filepath.Clean(outPath), filepath.Clean(dest)+string(os.PathSeparator)) {
				return fmt.Errorf("illegal file path: %s", outPath)
			}

    	    // Create parent directories
    	    if f.FileInfo().IsDir() {
    	        os.MkdirAll(outPath, f.Mode())
    	        continue
    	    } else {
    	        os.MkdirAll(filepath.Dir(outPath), 0755)
    	    }

    	    // Open file inside archive
    	    rc, err := f.Open()
    	    if err != nil {
    	        fmt.Println("Failed to open file in archive:", err)
    	        continue
    	    }

    	    // Create file on disk
    	    outFile, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
    	    if err != nil {
    	        fmt.Println("Failed to create output file:", err)
    	        rc.Close()
    	        continue
    	    }

    	    // Copy file contents
    	    _, err = io.Copy(outFile, rc)
    	    if err != nil {
    	        fmt.Println("Failed to extract:", f.Name, err)
    	    }

    	    rc.Close()
    	    outFile.Close()
    	}

    	fmt.Println("Extraction complete.")
	} else {
		r, err := zip.OpenReader(src)
		if err != nil {
			return err
		}
		defer r.Close()

		bar := progressbar.NewOptions(
			len(r.File),
			progressbar.OptionShowCount(),
			progressbar.OptionSetDescription("unzipping"))

		for _, f := range r.File {
			bar.Add(1)
			// Construct the full path for the destination. Since the zip file we
			// are using has a subfolder, we need to remove the first directory as we
			// unzip it.
			if removeFirstDir(f.Name) == constants.EMPTY {
				continue
			}

			fpath := filepath.Join(dest, removeFirstDir(f.Name))
			//log.Printf("fpath[%s]\n", fpath)

			// Prevent ZipSlip vulnerability.
			if !strings.HasPrefix(filepath.Clean(fpath), filepath.Clean(dest)+string(os.PathSeparator)) {
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

    	fmt.Println("Extraction complete.")
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
