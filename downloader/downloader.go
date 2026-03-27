// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package downloader handles downloading and extracting CS2 demo files from URLs.
// It supports caching (skipping already downloaded files) and automatic zip extraction.
package downloader

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Downloader manages downloading and extracting demo files to a local directory.
type Downloader struct {
	OutputDir string // Directory where downloaded and extracted files are stored
}

// NewDownloader creates a new Downloader with the specified output directory.
func NewDownloader(outputDir string) *Downloader {
	return &Downloader{OutputDir: outputDir}
}

// DownloadResult contains information about a completed download operation.
type DownloadResult struct {
	URL      string // Original URL that was downloaded
	ZipPath  string // Local path to the downloaded zip file
	DemoPath string // Local path to the extracted .dem file
	Error    error  // Any error that occurred during download
}

// Download fetches a file from the given URL and saves it to the output directory.
// If the file already exists locally, it skips the download (caching behavior).
func (d *Downloader) Download(url string) (*DownloadResult, error) {
	if err := os.MkdirAll(d.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]
	zipPath := filepath.Join(d.OutputDir, filename)
	if info, err := os.Stat(zipPath); err == nil {
		log.Printf("    Zip already exists: %s (%.2f MB)", filename, float64(info.Size())/(1024*1024))
		return &DownloadResult{
			URL:     url,
			ZipPath: zipPath,
		}, nil
	}

	log.Printf("    Downloading: %s", filename)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}
	if resp.ContentLength > 0 {
		log.Printf("    Size: %.2f MB", float64(resp.ContentLength)/(1024*1024))
	}
	out, err := os.Create(zipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file %s: %w", zipPath, err)
	}
	defer out.Close()
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write file %s: %w", zipPath, err)
	}

	log.Printf("    Downloaded: %.2f MB", float64(written)/(1024*1024))

	return &DownloadResult{
		URL:     url,
		ZipPath: zipPath,
	}, nil
}

// Extract opens a zip file and extracts the first .dem file found.
// If the demo is already extracted, it returns the existing path.
func (d *Downloader) Extract(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to open zip file %s: %w", zipPath, err)
	}
	defer r.Close()

	log.Printf("    Zip contains %d files", len(r.File))
	for _, f := range r.File {
		log.Printf("    Found in zip: %s (%.2f MB)", f.Name, float64(f.UncompressedSize64)/(1024*1024))

		if strings.HasSuffix(f.Name, ".dem") {
			demoPath := filepath.Join(d.OutputDir, filepath.Base(f.Name))
			if info, err := os.Stat(demoPath); err == nil {
				log.Printf("    Demo already extracted: %s (%.2f MB)", filepath.Base(demoPath), float64(info.Size())/(1024*1024))
				return demoPath, nil
			}

			log.Printf("    Extracting: %s", filepath.Base(f.Name))
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("failed to open file in zip: %w", err)
			}

			out, err := os.Create(demoPath)
			if err != nil {
				rc.Close()
				return "", fmt.Errorf("failed to create demo file: %w", err)
			}

			written, err := io.Copy(out, rc)
			rc.Close()
			out.Close()

			if err != nil {
				return "", fmt.Errorf("failed to extract demo file: %w", err)
			}

			log.Printf("    Extracted: %s (%.2f MB)", filepath.Base(demoPath), float64(written)/(1024*1024))
			return demoPath, nil
		}
	}

	return "", fmt.Errorf("no .dem file found in %s", zipPath)
}

// DownloadAndExtract is a convenience method that downloads and extracts in one call.
// Returns the path to the extracted .dem file.
func (d *Downloader) DownloadAndExtract(url string) (string, error) {
	result, err := d.Download(url)
	if err != nil {
		return "", err
	}

	demoPath, err := d.Extract(result.ZipPath)
	if err != nil {
		return "", err
	}

	return demoPath, nil
}

// DownloadDem downloads a .dem file directly (not zipped) from a URL.
// If the file already exists locally, it skips the download.
func (d *Downloader) DownloadDem(url string) (string, error) {
	if err := os.MkdirAll(d.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	parts := strings.Split(url, "/")
	filename := parts[len(parts)-1]
	demoPath := filepath.Join(d.OutputDir, filename)

	// Check if already downloaded
	if info, err := os.Stat(demoPath); err == nil {
		log.Printf("    Demo already exists: %s (%.2f MB)", filename, float64(info.Size())/(1024*1024))
		return demoPath, nil
	}

	log.Printf("    Downloading: %s", filename)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}

	if resp.ContentLength > 0 {
		log.Printf("    Size: %.2f MB", float64(resp.ContentLength)/(1024*1024))
	}

	out, err := os.Create(demoPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file %s: %w", demoPath, err)
	}
	defer out.Close()

	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", demoPath, err)
	}

	log.Printf("    Downloaded: %.2f MB", float64(written)/(1024*1024))
	return demoPath, nil
}
