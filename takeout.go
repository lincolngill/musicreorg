/*
Takeout extracts music from a google takeout zip file and:
1) Corrects MP3 ID tags - Especially the Alburm Artist. To correctly group tracks.
2) Organises files into Album Artist -> Album Name -> Track File directory structure.
It is intended to prepare the music for Volumio.
*/
package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"strings"

	"github.com/dhowden/tag"
)

var version string = "0.1.0"

// Error type to enable error type check when a file already exists
type fileExistsError struct {
	fpath string
}

func (e *fileExistsError) Error() string {
	return fmt.Sprintf("File exists: %s", e.fpath)
}

func main() {
	log.Printf("Takeout %v\n", version)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to determine home directory. %v\n", err)
	}
	var (
		inputFileName string = path.Join(homeDir, "Downloads", "takeout-20200813T083635Z-002.zip")
		outputDir     string = path.Join(homeDir, "Music", "takeout")
		tmpDir        string = path.Join(outputDir, "tmp")
	)

	// Make the outputDir and tmpDir if they don't exist.
	for _, d := range []string{outputDir, tmpDir} {
		if _, err := os.Stat(d); os.IsNotExist(err) {
			log.Printf("Creating directory: %s\n", d)
			os.MkdirAll(d, 0755)
		}
	}

	zipReadCloser, err := zip.OpenReader(inputFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer zipReadCloser.Close()

	// ReadCloser is a struct containing a Reader
	// A Reader is a struct containing File which is []*File of the file names in the zip file.
	for _, file := range zipReadCloser.Reader.File {
		if strings.ToLower(path.Ext(file.FileHeader.Name)) == ".mp3" {
			fmt.Printf("   Upzipping %s... ", file.FileHeader.Name)
			fpath, err := unzip(file, tmpDir)
			if err != nil {
				if _, ok := err.(*fileExistsError); ok {
					fmt.Printf("SKIPPED - %v\n", err)
				} else {
					fmt.Printf("ERROR - %v\n", err)
					continue
				}
			} else {
				fmt.Println("Done")
			}
			fmt.Println(fpath)
			m, err := metadata(fpath)
			if err != nil {
				log.Fatal(err)
			}
			disc, discTotal := m.Disc()
			track, trackTotal := m.Track()
			fmt.Printf("        Format: %s\n", m.Format())
			fmt.Printf("         Title: %s\n", m.Title())
			fmt.Printf("          Disc: %v of %v\n", disc, discTotal)
			fmt.Printf("         Track: %v of %v\n", track, trackTotal)
			fmt.Printf("         Album: %s\n", m.Album())
			fmt.Printf("        Artist: %s\n", m.Artist())
			fmt.Printf("   AlbumArtist: %s\n", m.AlbumArtist())
		}
	}

	log.Println("Done")
}

// unzip unzips a single file from a zip file and writes it to destDir.
// Returns the destination file path and err.
// Returns a fileExistsError in err, if the destination files already exists and does not overwrite the file.
func unzip(file *zip.File, destDir string) (destPath string, err error) {
	destPath = path.Join(destDir, path.Base(file.FileHeader.Name))
	if fileInfo, err := os.Stat(destPath); err == nil && fileInfo.Mode().IsRegular() {
		return destPath, &fileExistsError{destPath}
	}
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		return
	}
	readCloser, err := file.Open()
	if err != nil {
		return
	}
	_, err = io.Copy(destFile, readCloser)
	destFile.Close()
	readCloser.Close()
	return
}

// metadata opens a media file and returns the metadata tags
func metadata(fpath string) (m tag.Metadata, err error) {
	f, err := os.Open(fpath)
	if err != nil {
		return nil, err
	}
	return tag.ReadFrom(f)
}
