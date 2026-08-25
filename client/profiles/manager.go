package profiles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	PinHash   string    `json:"pin_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type Manager struct {
	baseDir      string
	profilesDir  string
	registryFile string
}

func NewManager(appDataDir string) (*Manager, error) {
	m := &Manager{
		baseDir:      appDataDir,
		profilesDir:  filepath.Join(appDataDir, "profiles"),
		registryFile: filepath.Join(appDataDir, "profiles.json"),
	}

	if err := os.MkdirAll(m.profilesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create profiles directory: %v", err)
	}

	if _, err := os.Stat(m.registryFile); os.IsNotExist(err) {
		if err := m.saveProfiles([]Profile{}); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (m *Manager) GetProfiles() ([]Profile, error) {
	data, err := os.ReadFile(m.registryFile)
	if err != nil {
		return nil, err
	}

	var profiles []Profile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, err
	}

	return profiles, nil
}

func (m *Manager) CreateProfile(name, pin, color string) (Profile, error) {
	profiles, err := m.GetProfiles()
	if err != nil {
		return Profile{}, err
	}

	newID := uuid.New().String()

	pinHash := ""
	if pin != "" {
		pinHash = hashPin(pin)
	}

	newProfile := Profile{
		ID:        newID,
		Name:      name,
		Color:     color,
		PinHash:   pinHash,
		CreatedAt: time.Now(),
	}

	profilePath := m.GetProfilePath(newID)
	if err := os.MkdirAll(profilePath, 0755); err != nil {
		return Profile{}, fmt.Errorf("failed to create profile folder: %v", err)
	}
	_ = os.MkdirAll(filepath.Join(profilePath, "scores"), 0755)

	profiles = append(profiles, newProfile)
	if err := m.saveProfiles(profiles); err != nil {
		return Profile{}, err
	}

	return newProfile, nil
}

func (m *Manager) DeleteProfile(id string) error {
	profiles, err := m.GetProfiles()
	if err != nil {
		return err
	}

	var updated []Profile
	found := false
	for _, p := range profiles {
		if p.ID == id {
			found = true
		} else {
			updated = append(updated, p)
		}
	}

	if !found {
		return fmt.Errorf("profile not found")
	}

	if err := m.saveProfiles(updated); err != nil {
		return err
	}

	return os.RemoveAll(m.GetProfilePath(id))
}

func (m *Manager) VerifyPin(id, pin string) bool {
	profiles, err := m.GetProfiles()
	if err != nil {
		return false
	}

	for _, p := range profiles {
		if p.ID == id {
			if p.PinHash == "" {
				return true
			}
			return p.PinHash == hashPin(pin)
		}
	}
	return false
}

func (m *Manager) GetProfilePath(id string) string {
	return filepath.Join(m.profilesDir, id)
}

func (m *Manager) saveProfiles(profiles []Profile) error {
	data, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.registryFile, data, 0644)
}

func hashPin(pin string) string {
	hash := sha256.Sum256([]byte(pin))
	return hex.EncodeToString(hash[:])
}

func (m *Manager) UpdatePin(id, newPin string) error {
	profiles, err := m.GetProfiles()
	if err != nil {
		return err
	}

	updated := false
	for i, p := range profiles {
		if p.ID == id {
			if newPin == "" {
				profiles[i].PinHash = ""
			} else {
				profiles[i].PinHash = hashPin(newPin)
			}
			updated = true
			break
		}
	}

	if !updated {
		return fmt.Errorf("profile not found")
	}

	return m.saveProfiles(profiles)
}

func (m *Manager) CheckIfHasPin(id string) bool {
	profiles, err := m.GetProfiles()
	if err != nil {
		return false
	}
	for _, p := range profiles {
		if p.ID == id && p.PinHash != "" {
			return true
		}
	}
	return false
}
