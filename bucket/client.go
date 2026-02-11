// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package bucket provides a client for interacting with cloud storage buckets
// (specifically DigitalOcean Spaces) to list and download CS2 demo files.
// It handles XML parsing of bucket listings and filtering demos by competitive tier.
package bucket

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// ListBucketResult represents the XML response from an S3-compatible bucket listing.
// It contains metadata about the bucket and its contents.
type ListBucketResult struct {
	XMLName        xml.Name        `xml:"ListBucketResult"`
	Name           string          `xml:"Name"`           // Bucket name
	Prefix         string          `xml:"Prefix"`         // Prefix filter used in the request
	MaxKeys        int             `xml:"MaxKeys"`        // Maximum number of keys returned
	Delimiter      string          `xml:"Delimiter"`      // Delimiter used for grouping (usually "/")
	IsTruncated    bool            `xml:"IsTruncated"`    // Whether results are truncated
	Marker         string          `xml:"Marker"`         // Marker for pagination
	CommonPrefixes []CommonPrefix  `xml:"CommonPrefixes"` // Virtual "folders" in the bucket
	Contents       []BucketContent `xml:"Contents"`       // Actual files in the bucket
}

// CommonPrefix represents a virtual folder/directory in the bucket listing.
type CommonPrefix struct {
	Prefix string `xml:"Prefix"` // The folder path prefix
}

// BucketContent represents a single file/object in the bucket.
type BucketContent struct {
	Key          string `xml:"Key"`          // Full path/key of the object
	LastModified string `xml:"LastModified"` // ISO 8601 timestamp of last modification
	ETag         string `xml:"ETag"`         // Entity tag (hash) for the object
	Size         int64  `xml:"Size"`         // Size in bytes
	StorageClass string `xml:"StorageClass"` // Storage class (e.g., STANDARD)
}

// Client provides methods for interacting with an S3-compatible bucket.
type Client struct {
	BaseURL string // Base URL of the bucket (e.g., https://bucket.nyc3.digitaloceanspaces.com/)
}

// NewClient creates a new bucket client with the specified base URL.
func NewClient(baseURL string) *Client {
	return &Client{BaseURL: baseURL}
}

// ListFolders returns all virtual folders (common prefixes) under the given prefix.
// This is used to enumerate combine event folders in the bucket.
func (c *Client) ListFolders(prefix string) ([]string, error) {
	result, err := c.listBucket(prefix)
	if err != nil {
		return nil, err
	}

	folders := make([]string, 0, len(result.CommonPrefixes))
	for _, cp := range result.CommonPrefixes {
		folders = append(folders, cp.Prefix)
	}
	return folders, nil
}

// ListFiles returns all files under the given prefix.
func (c *Client) ListFiles(prefix string) ([]BucketContent, error) {
	result, err := c.listBucket(prefix)
	if err != nil {
		return nil, err
	}
	return result.Contents, nil
}

// ListFilesByTier returns files filtered by competitive tier.
// It looks for files with names starting with "combine-{tier}".
func (c *Client) ListFilesByTier(prefix, tier string) ([]BucketContent, error) {
	files, err := c.ListFiles(prefix)
	if err != nil {
		return nil, err
	}

	tierPrefix := "combine-" + tier
	filtered := make([]BucketContent, 0)
	for _, f := range files {
		parts := strings.Split(f.Key, "/")
		filename := parts[len(parts)-1]
		if strings.HasPrefix(filename, tierPrefix) {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

// GetAllDemosByTier retrieves all demo files for a specific tier across all combine folders.
// It iterates through each combine folder and collects matching demos.
func (c *Client) GetAllDemosByTier(combinesPrefix, tier string) ([]BucketContent, error) {
	folders, err := c.ListFolders(combinesPrefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list combine folders: %w", err)
	}

	var allDemos []BucketContent
	for _, folder := range folders {
		demos, err := c.ListFilesByTier(folder, tier)
		if err != nil {
			return nil, fmt.Errorf("failed to list files in %s: %w", folder, err)
		}
		allDemos = append(allDemos, demos...)
	}

	return allDemos, nil
}

// GetAllDemos retrieves all demo files under the given prefix (including subfolders)
// without filtering by tier. Used when tier is "all" or when filenames don't follow
// the combine-{tier} naming convention.
func (c *Client) GetAllDemos(prefix string) ([]BucketContent, error) {
	// First try listing files directly at this prefix
	result, err := c.listBucket(prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list bucket at %s: %w", prefix, err)
	}

	var allDemos []BucketContent

	// Collect any demo files at this level
	for _, f := range result.Contents {
		if isDemoFile(f.Key) {
			allDemos = append(allDemos, f)
		}
	}

	// Recurse into subfolders
	for _, cp := range result.CommonPrefixes {
		subDemos, err := c.GetAllDemos(cp.Prefix)
		if err != nil {
			return nil, fmt.Errorf("failed to list files in %s: %w", cp.Prefix, err)
		}
		allDemos = append(allDemos, subDemos...)
	}

	return allDemos, nil
}

// isDemoFile checks if a key looks like a demo file (.dem or .dem.zip).
func isDemoFile(key string) bool {
	lower := strings.ToLower(key)
	return strings.HasSuffix(lower, ".dem") ||
		strings.HasSuffix(lower, ".dem.zip") ||
		strings.HasSuffix(lower, ".dem.gz")
}

// ParseTierFromKey extracts the competitive tier from a demo file key.
// For old format (combine-{tier}-...), it returns the tier name.
// For new format (s19-M01-TeamA-vs-TeamB-...) or unrecognized formats, it returns "".
func ParseTierFromKey(key string) string {
	filename := path.Base(key)
	if strings.HasPrefix(filename, "combine-") {
		// Old format: combine-contender-mid7272-0_de_mirage-...
		rest := strings.TrimPrefix(filename, "combine-")
		idx := strings.Index(rest, "-")
		if idx > 0 {
			return rest[:idx]
		}
	}
	return ""
}

// ParseTeamsFromKey extracts team names from a demo file key with the new naming format.
// For new format (s19-M01-TeamA-vs-TeamB-mid...), it returns (TeamA, TeamB, true).
// For old format or unrecognized formats, it returns ("", "", false).
func ParseTeamsFromKey(key string) (team1, team2 string, ok bool) {
	filename := path.Base(key)
	// New format: s19-M01-TeamA-vs-TeamB-mid7712-0_de_mirage-...
	// Find "-vs-" separator
	vsIdx := strings.Index(filename, "-vs-")
	if vsIdx < 0 {
		return "", "", false
	}

	// Extract team1: everything between the second "-" and "-vs-"
	// e.g., "s19-M01-TiltedTogglers-vs-..." -> find prefix before -vs-
	prefix := filename[:vsIdx]
	// Find the team name by looking for the pattern after "s##-M##-" or similar prefix
	// Strategy: find the part after the second hyphen-separated segment that looks like a season/match prefix
	parts := strings.SplitN(prefix, "-", 3)
	if len(parts) < 3 {
		return "", "", false
	}
	team1 = parts[2] // Everything after "s19-M01-"

	// Extract team2: everything between "-vs-" and "-mid"
	after := filename[vsIdx+4:] // skip "-vs-"
	midIdx := strings.Index(after, "-mid")
	if midIdx < 0 {
		return "", "", false
	}
	team2 = after[:midIdx]

	if team1 == "" || team2 == "" {
		return "", "", false
	}

	return team1, team2, true
}

// GetDemosByTeam retrieves all demo files under the given prefix whose filename
// contains the specified team name. Works with both old format (combine-...) and
// new format (s19-M01-TeamA-vs-TeamB-...).
func (c *Client) GetDemosByTeam(prefix, teamName string) ([]BucketContent, error) {
	allDemos, err := c.GetAllDemos(prefix)
	if err != nil {
		return nil, err
	}

	lowerTeam := strings.ToLower(teamName)
	var filtered []BucketContent
	for _, f := range allDemos {
		filename := strings.ToLower(path.Base(f.Key))
		if strings.Contains(filename, lowerTeam) {
			filtered = append(filtered, f)
		}
	}
	return filtered, nil
}

// GetDownloadURL constructs the full download URL for a given object key.
func (c *Client) GetDownloadURL(key string) string {
	return c.BaseURL + key
}

// listBucket performs the actual HTTP request to list bucket contents.
// It uses the delimiter "/" to enable folder-like navigation.
// It handles pagination automatically when results are truncated.
func (c *Client) listBucket(prefix string) (*ListBucketResult, error) {
	var combined ListBucketResult
	marker := ""

	for {
		params := url.Values{}
		params.Set("delimiter", "/")
		params.Set("prefix", prefix)
		if marker != "" {
			params.Set("marker", marker)
		}

		reqURL := c.BaseURL + "?" + params.Encode()

		resp, err := http.Get(reqURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch bucket listing: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		var result ListBucketResult
		if err := xml.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("failed to parse XML response: %w", err)
		}

		combined.Name = result.Name
		combined.Prefix = result.Prefix
		combined.MaxKeys = result.MaxKeys
		combined.Delimiter = result.Delimiter
		combined.CommonPrefixes = append(combined.CommonPrefixes, result.CommonPrefixes...)
		combined.Contents = append(combined.Contents, result.Contents...)

		if !result.IsTruncated {
			break
		}

		// Use the last key as the marker for the next page
		if len(result.Contents) > 0 {
			marker = result.Contents[len(result.Contents)-1].Key
		} else if result.Marker != "" {
			marker = result.Marker
		} else {
			break
		}
	}

	return &combined, nil
}
