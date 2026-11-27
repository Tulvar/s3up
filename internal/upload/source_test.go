package upload

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenSourceRejectsSymlinkReplacement(t *testing.T) {
	t.Parallel()

	for _, replace := range []string{"single file", "nested file", "subdirectory", "source directory"} {
		t.Run(replace, func(t *testing.T) {
			parent := t.TempDir()
			source := filepath.Join(parent, "source")
			file := filepath.Join(source, "nested", "artifact.txt")
			outside := filepath.Join(parent, "outside")
			writeFile(t, file, "original")
			writeFile(t, filepath.Join(outside, "nested", "artifact.txt"), "outside")

			req := Request{Source: source, Recursive: true}
			if replace == "single file" {
				req = Request{Source: file}
			}
			plan, err := Plan(req)
			if err != nil || len(plan) != 1 {
				t.Fatalf("planning returned %d files and error %v", len(plan), err)
			}
			input := UploadInput(plan[0])
			checkSourceContents(t, input, "original")

			var oldPath, target string
			switch replace {
			case "single file", "nested file":
				oldPath, target = file, filepath.Join(outside, "nested", "artifact.txt")
			case "subdirectory":
				oldPath, target = filepath.Dir(file), filepath.Join(outside, "nested")
			case "source directory":
				oldPath, target = source, outside
			}
			if err := os.Rename(oldPath, oldPath+".old"); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, oldPath); err != nil {
				t.Fatal(err)
			}
			if opened, err := input.OpenSource(); err == nil {
				opened.Close()
				t.Fatal("opened a source after it was replaced by a symlink")
			}
		})
	}
}

func TestOpenSourceFollowsSymlinkWhenEnabled(t *testing.T) {
	t.Parallel()

	parent := t.TempDir()
	source := filepath.Join(parent, "source")
	target := filepath.Join(parent, "outside.txt")
	writeFile(t, target, "outside")
	if err := os.Symlink(target, source); err != nil {
		t.Fatal(err)
	}
	plan, err := Plan(Request{Source: source, FollowLinks: true})
	if err != nil || len(plan) != 1 {
		t.Fatalf("planning returned %d files and error %v", len(plan), err)
	}
	checkSourceContents(t, UploadInput(plan[0]), "outside")
}

func checkSourceContents(t *testing.T, input UploadInput, want string) {
	t.Helper()
	file, err := input.OpenSource()
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != want {
		t.Errorf("got %q, want %q", body, want)
	}
}
