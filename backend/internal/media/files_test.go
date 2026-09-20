package media

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveRejectsSiblingSymlinkAndDirectory(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "music")
	sibling := filepath.Join(dir, "music-private")
	for _, p := range []string{root, sibling} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("MEDIA_PATH", root)
	good := filepath.Join(root, "good.mp3")
	bad := filepath.Join(sibling, "private.mp3")
	for _, p := range []string{good, bad} {
		if err := os.WriteFile(p, []byte("audio"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	link := filepath.Join(root, "linked.mp3")
	if err := os.Symlink(bad, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(good); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{bad, link, root} {
		if _, err := Resolve(p); err == nil {
			t.Fatalf("accepted %s", p)
		}
	}
}
