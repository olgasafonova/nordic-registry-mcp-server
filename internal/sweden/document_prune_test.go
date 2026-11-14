package sweden

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// writeAged creates a file in dir with the given name and modification time.
func writeAged(t *testing.T, dir, name string, modTime time.Time) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("zip"), 0o600); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("setting mtime on %s: %v", name, err)
	}
	return path
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func TestPruneOldDocuments(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	expired := writeAged(t, dir, "sweden-arsredovisning-111.zip", now.Add(-documentRetention-time.Minute))
	fresh := writeAged(t, dir, "sweden-arsredovisning-222.zip", now.Add(-documentRetention+time.Minute))
	foreign := writeAged(t, dir, "someone-elses-report.zip", now.Add(-10*documentRetention))

	pruneOldDocuments(dir, now)

	if exists(expired) {
		t.Error("a download older than the retention window was not removed")
	}
	if !exists(fresh) {
		t.Error("a download inside the retention window was removed")
	}
	if !exists(foreign) {
		t.Error("a file this server did not create was removed")
	}
}

func TestPruneOldDocuments_SkipsNonRegularFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	// A directory whose name matches the pattern must survive, however old.
	matchingDir := filepath.Join(dir, "sweden-arsredovisning-333.zip")
	if err := os.Mkdir(matchingDir, 0o700); err != nil {
		t.Fatalf("creating directory: %v", err)
	}
	old := now.Add(-10 * documentRetention)
	if err := os.Chtimes(matchingDir, old, old); err != nil {
		t.Fatalf("setting mtime: %v", err)
	}

	pruneOldDocuments(dir, now)

	if !exists(matchingDir) {
		t.Error("a non-regular entry matching the pattern was removed")
	}
}

func TestPruneOldDocuments_MissingDirIsNoOp(t *testing.T) {
	// Must not panic when the directory does not exist.
	pruneOldDocuments(filepath.Join(t.TempDir(), "does-not-exist"), time.Now())
}

func TestWriteDocumentToTempFile_UsesPrunablePattern(t *testing.T) {
	path, err := writeDocumentToTempFile([]byte("zip"))
	if err != nil {
		t.Fatalf("writeDocumentToTempFile: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	matched, err := filepath.Match(documentFilePattern, filepath.Base(path))
	if err != nil || !matched {
		t.Errorf("written file %q does not match the cleanup pattern %q", path, documentFilePattern)
	}
}
