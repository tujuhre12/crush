package update

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/crush/internal/version"
)

const updateCheckInterval = 24 * time.Hour

// LastCheckInfo stores information about the last update check.
type LastCheckInfo struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
	ReleaseURL    string    `json:"release_url"`
	Available     bool      `json:"available"`
}

// ShouldCheckForUpdate determines if we should check for updates based on the last check time.
func ShouldCheckForUpdate(dataDir string) bool {
	info, err := loadLastCheckInfo(dataDir)
	if err != nil {
		// If we can't load the info, we should check.
		return true
	}
	
	return time.Since(info.CheckedAt) > updateCheckInterval
}

// SaveLastCheckInfo saves information about the last update check.
func SaveLastCheckInfo(dataDir string, info *UpdateInfo) error {
	lastCheck := LastCheckInfo{
		CheckedAt:     time.Now(),
		LatestVersion: info.LatestVersion,
		ReleaseURL:    info.ReleaseURL,
		Available:     info.Available,
	}
	
	data, err := json.MarshalIndent(lastCheck, "", "  ")
	if err != nil {
		return err
	}
	
	path := filepath.Join(dataDir, "last-update-check.json")
	return os.WriteFile(path, data, 0o644)
}

// GetLastCheckInfo returns information about the last update check.
func GetLastCheckInfo(dataDir string) (*LastCheckInfo, error) {
	return loadLastCheckInfo(dataDir)
}

func loadLastCheckInfo(dataDir string) (*LastCheckInfo, error) {
	path := filepath.Join(dataDir, "last-update-check.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var info LastCheckInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	
	return &info, nil
}

// CheckForUpdateAsync performs an update check in the background and returns immediately.
// If an update is available, it returns the update info through the channel.
func CheckForUpdateAsync(ctx context.Context, dataDir string) <-chan *UpdateInfo {
	ch := make(chan *UpdateInfo, 1)
	
	go func() {
		defer close(ch)
		
		// Check if we should perform the check.
		if !ShouldCheckForUpdate(dataDir) {
			// Even if we shouldn't check, show notification if forced
			if os.Getenv("CRUSH_FORCE_UPDATE_NOTIFICATION") == "1" {
				lastInfo, err := loadLastCheckInfo(dataDir)
				if err == nil && lastInfo.Available {
					ch <- &UpdateInfo{
						CurrentVersion: version.Version,
						LatestVersion:  lastInfo.LatestVersion,
						ReleaseURL:     lastInfo.ReleaseURL,
						Available:      true,
					}
				}
			}
			return
		}
		
		// Perform the check.
		info, err := CheckForUpdate(ctx)
		if err != nil {
			// Log error but don't fail.
			fmt.Fprintf(os.Stderr, "Failed to check for updates: %v\n", err)
			return
		}
		
		// Save the check info.
		if err := SaveLastCheckInfo(dataDir, info); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save update check info: %v\n", err)
		}
		
		// Send update info if available.
		if info.Available {
			ch <- info
		}
	}()
	
	return ch
}