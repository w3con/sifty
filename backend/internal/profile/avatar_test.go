package profile_test

import (
	"encoding/base64"
	"errors"
	"os"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/profile"
)

// TestAvatarReplacementFailurePreservesPrevious catches the hidden photo-loss and persistence
// failures: an invalid replacement must not remove the current photo, and a stored photo must
// still load through a fresh service over the same data directory.
func TestAvatarReplacementFailurePreservesPrevious(t *testing.T) {
	d := dirs.Dirs{Root: t.TempDir()}
	if err := d.Ensure(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}
	cfg := config.NewHolder(&config.Config{})
	svc := profile.NewService(nil, cfg, profile.WithDirs(d))

	png, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	// The filename claims a text document; the server must trust the actual bytes.
	first, err := svc.SaveAvatar("photo.txt", png)
	if err != nil {
		t.Fatalf("save avatar: %v", err)
	}
	if first.ContentType != "image/png" || first.ByteSize != len(png) {
		t.Fatalf("saved avatar = %+v, want image/png %d bytes", first, len(png))
	}
	path, contentType, err := svc.AvatarFile()
	if err != nil {
		t.Fatalf("read avatar: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read avatar file: %v", err)
	}
	if string(raw) != string(png) || contentType != "image/png" {
		t.Fatalf("stored bytes/content type changed: %s %s", contentType, raw)
	}

	if _, err := svc.SaveAvatar("broken.png", []byte("<html>not an image</html>")); err == nil {
		t.Fatal("invalid replacement unexpectedly succeeded")
	}
	pathAfter, afterType, err := svc.AvatarFile()
	if err != nil {
		t.Fatalf("previous avatar lost after failed replacement: %v", err)
	}
	after, err := os.ReadFile(pathAfter)
	if err != nil {
		t.Fatalf("read previous avatar: %v", err)
	}
	if pathAfter != path || afterType != "image/png" || string(after) != string(png) {
		t.Fatal("failed replacement changed the previous photo")
	}

	// A signature or a truncated header is not a complete image and must not replace the photo.
	for name, broken := range map[string][]byte{
		"signature": png[:8],
		"truncated": png[:20],
		"jpeg":      {0xff, 0xd8, 0xff, 0xe0, 0, 0, 0, 0},
	} {
		if _, err := svc.SaveAvatar("broken-"+name+".png", broken); err == nil {
			t.Fatalf("incomplete %s image unexpectedly succeeded", name)
		}
		pathBroken, typeBroken, err := svc.AvatarFile()
		if err != nil || pathBroken != path || typeBroken != "image/png" {
			t.Fatalf("%s image changed the previous photo: %v", name, err)
		}
	}

	// A real metadata-write failure must leave the old pointer and file serving.
	if os.Geteuid() != 0 {
		if err := os.Chmod(d.Root, 0o555); err != nil {
			t.Fatalf("make data root read-only: %v", err)
		}
		_, saveErr := svc.SaveAvatar("blocked-metadata.png", png)
		if err := os.Chmod(d.Root, 0o755); err != nil {
			t.Fatalf("restore data root permissions: %v", err)
		}
		if saveErr == nil {
			t.Fatal("metadata-write failure unexpectedly succeeded")
		}
		pathFailed, typeFailed, err := svc.AvatarFile()
		if err != nil || pathFailed != path || typeFailed != "image/png" {
			t.Fatalf("metadata failure changed the previous photo: %v", err)
		}
	}

	restarted := profile.NewService(nil, cfg, profile.WithDirs(d))
	persisted, err := restarted.GetAvatar()
	if err != nil || persisted == nil || persisted.ContentType != "image/png" {
		t.Fatalf("avatar did not persist across a fresh service: %+v %v", persisted, err)
	}
	if _, _, err := restarted.AvatarFile(); err != nil {
		t.Fatalf("persisted avatar file missing: %v", err)
	}

	deleted, err := restarted.DeleteAvatar()
	if err != nil || !deleted {
		t.Fatalf("delete avatar = %v %v", deleted, err)
	}
	if _, _, err := restarted.AvatarFile(); !errors.Is(err, profile.ErrAvatarNotFound) {
		t.Fatalf("avatar still present after delete: %v", err)
	}
}
