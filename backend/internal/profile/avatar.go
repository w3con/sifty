package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/image/webp"
)

// ErrAvatarNotFound reports that the profile has no stored photo.
var ErrAvatarNotFound = errors.New("profile photo is not set")

// Avatar is the API-facing metadata for the stored profile photo. The image bytes are served
// by a separate endpoint so a normal profile read does not carry the file.
type Avatar struct {
	ContentType string `json:"contentType"`
	ByteSize    int    `json:"byteSize"`
	UpdatedAt   string `json:"updatedAt"`
}

// avatarMetadata is the persisted pointer to the one managed photo file. It is kept beside the
// other profile originals under the configured data directory, not mixed into CV documents.
type avatarMetadata struct {
	StoragePath string `json:"storagePath"`
	ContentType string `json:"contentType"`
	ByteSize    int    `json:"byteSize"`
	UpdatedAt   string `json:"updatedAt"`
}

const avatarMetadataFile = "profile-avatar.json"

func (s *Service) avatarMetadataPath() (string, error) {
	return s.dirs.Under(avatarMetadataFile)
}

func readAvatarMetadata(path string) (*avatarMetadata, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read profile photo metadata: %w", err)
	}
	var meta avatarMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return nil, fmt.Errorf("decode profile photo metadata: %w", err)
	}
	if meta.StoragePath == "" || meta.ContentType == "" {
		return nil, fmt.Errorf("profile photo metadata is incomplete")
	}
	return &meta, nil
}

func writeFileAtomic(path string, data []byte) error {
	tmp := path + ".tmp-" + uuid.NewString()
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// GetAvatar returns the stored photo metadata, or nil when no photo is set.
func (s *Service) GetAvatar() (*Avatar, error) {
	metaPath, err := s.avatarMetadataPath()
	if err != nil {
		return nil, err
	}
	meta, err := readAvatarMetadata(metaPath)
	if err != nil || meta == nil {
		return nil, err
	}
	path, err := s.dirs.Under(meta.StoragePath)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read profile photo: %w", err)
	}
	return &Avatar{ContentType: meta.ContentType, ByteSize: meta.ByteSize, UpdatedAt: meta.UpdatedAt}, nil
}

// AvatarFile resolves the stored photo path and its content type.
func (s *Service) AvatarFile() (string, string, error) {
	metaPath, err := s.avatarMetadataPath()
	if err != nil {
		return "", "", err
	}
	meta, err := readAvatarMetadata(metaPath)
	if err != nil {
		return "", "", err
	}
	if meta == nil {
		return "", "", ErrAvatarNotFound
	}
	path, err := s.dirs.Under(meta.StoragePath)
	if err != nil {
		return "", "", err
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", "", ErrAvatarNotFound
		}
		return "", "", fmt.Errorf("read profile photo: %w", err)
	}
	return path, meta.ContentType, nil
}

// decodeAvatarImage recognises and fully decodes the supported formats. A signature or a
// truncated header is not enough: the bytes must form a complete image before they can replace
// the current photo.
func decodeAvatarImage(content []byte) (ext, mimeType string, err error) {
	detected := http.DetectContentType(content)
	switch detected {
	case "image/png":
		if _, err := png.Decode(bytes.NewReader(content)); err != nil {
			return "", detected, err
		}
		return "png", detected, nil
	case "image/jpeg":
		if _, err := jpeg.Decode(bytes.NewReader(content)); err != nil {
			return "", detected, err
		}
		return "jpg", detected, nil
	case "image/webp":
		if _, err := webp.Decode(bytes.NewReader(content)); err != nil {
			return "", detected, err
		}
		return "webp", detected, nil
	}
	// Some Go releases do not recognise the RIFF/WEBP container; check and decode it explicitly.
	if len(content) >= 12 && string(content[:4]) == "RIFF" && string(content[8:12]) == "WEBP" {
		if _, err := webp.Decode(bytes.NewReader(content)); err != nil {
			return "", "image/webp", err
		}
		return "webp", "image/webp", nil
	}
	return "", detected, fmt.Errorf("unsupported image type %q", detected)
}

func avatarTypeAllowed(configured []string, ext string) bool {
	for _, item := range configured {
		value := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(item), "."))
		if value == ext || (ext == "jpg" && value == "jpeg") {
			return true
		}
	}
	return false
}

// SaveAvatar validates and stores the photo. A replacement writes the new file first and then
// atomically swaps the metadata pointer, so a failed replacement leaves the previous photo
// serving. The server checks the actual bytes; the filename is metadata only.
func (s *Service) SaveAvatar(filename string, content []byte) (*Avatar, error) {
	if len(content) == 0 {
		return nil, fmt.Errorf("profile photo content is empty")
	}
	cfg := s.cfg.Get()
	max := cfg.Runtime.EffectiveProfileAvatarMaxBytes()
	if len(content) > max {
		return nil, fmt.Errorf("profile photo is %d bytes; configured limit is %d", len(content), max)
	}
	ext, mimeType, decodeErr := decodeAvatarImage(content)
	if decodeErr != nil {
		return nil, fmt.Errorf("profile photo must be a complete PNG, JPEG or WebP image: %w", decodeErr)
	}
	if !avatarTypeAllowed(cfg.Runtime.EffectiveProfileAvatarTypes(), ext) {
		return nil, fmt.Errorf("profile photo type %q is not accepted; configured types: %s", ext,
			strings.Join(cfg.Runtime.EffectiveProfileAvatarTypes(), ", "))
	}
	metaPath, err := s.avatarMetadataPath()
	if err != nil {
		return nil, err
	}
	old, err := readAvatarMetadata(metaPath)
	if err != nil {
		return nil, err
	}
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "photo." + ext
	}
	rel := filepath.Join("profile-avatar", uuid.NewString()+"."+ext)
	abs, err := s.dirs.Under(rel)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o700); err != nil {
		return nil, fmt.Errorf("create profile photo directory: %w", err)
	}
	if err := writeFileAtomic(abs, content); err != nil {
		return nil, fmt.Errorf("write profile photo: %w", err)
	}
	meta := &avatarMetadata{
		StoragePath: rel,
		ContentType: mimeType,
		ByteSize:    len(content),
		UpdatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
	}
	encoded, err := json.Marshal(meta)
	if err != nil {
		_ = os.Remove(abs)
		return nil, fmt.Errorf("encode profile photo metadata: %w", err)
	}
	if err := writeFileAtomic(metaPath, encoded); err != nil {
		_ = os.Remove(abs)
		return nil, fmt.Errorf("save profile photo metadata: %w", err)
	}
	if old != nil && old.StoragePath != rel {
		if oldPath, pathErr := s.dirs.Under(old.StoragePath); pathErr == nil {
			_ = os.Remove(oldPath)
		}
	}
	return &Avatar{ContentType: meta.ContentType, ByteSize: meta.ByteSize, UpdatedAt: meta.UpdatedAt}, nil
}

// DeleteAvatar removes the stored photo. The metadata pointer is cleared first so a failed file
// removal can only leave an unreferenced file, never a broken visible photo.
func (s *Service) DeleteAvatar() (bool, error) {
	metaPath, err := s.avatarMetadataPath()
	if err != nil {
		return false, err
	}
	old, err := readAvatarMetadata(metaPath)
	if err != nil {
		return false, err
	}
	if old == nil {
		return false, nil
	}
	if err := os.Remove(metaPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("remove profile photo metadata: %w", err)
	}
	if oldPath, pathErr := s.dirs.Under(old.StoragePath); pathErr == nil {
		_ = os.Remove(oldPath)
	}
	return true, nil
}
