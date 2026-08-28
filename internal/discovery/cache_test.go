package discovery

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanCachesUnchangedFilesAndInvalidatesChangedAndRemovedFiles(t *testing.T) {
	home := t.TempDir()
	directory := filepath.Join(home, ".codex", "sessions", "2026", "08", "28")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "rollout-test.jsonl")
	first := `{"type":"response_item","payload":{"role":"user","content":[{"text":"First task"}]}}`
	second := `{"type":"response_item","payload":{"role":"user","content":[{"text":"Other task"}]}}`
	if len(first) != len(second) {
		t.Fatal("cache hit fixture contents must be the same size")
	}
	if err := os.WriteFile(path, []byte(first), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	scanner := New(home)
	firstResult := scanner.Scan(t.Context())
	if len(firstResult.Sessions) != 1 || firstResult.Sessions[0].Title != "First task" {
		t.Fatalf("first scan = %#v, warnings %v", firstResult.Sessions, firstResult.Warnings)
	}
	if _, err := os.Stat(scanner.CachePath); err != nil {
		t.Fatalf("cache was not created: %v", err)
	}
	cachedSnapshot, ok := scanner.Cached()
	if !ok || len(cachedSnapshot.Sessions) != 1 || cachedSnapshot.Sessions[0].Title != "First task" {
		t.Fatalf("cached snapshot = %#v, ok %t", cachedSnapshot.Sessions, ok)
	}

	if err := os.WriteFile(path, []byte(second), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
		t.Fatal(err)
	}
	cachedResult := scanner.Scan(t.Context())
	if len(cachedResult.Sessions) != 1 || cachedResult.Sessions[0].Title != "First task" {
		t.Fatalf("unchanged fingerprint did not use cache: %#v", cachedResult.Sessions)
	}

	changed := second + "\n" + `{"type":"response_item","payload":{"role":"user","content":[{"text":"Replacement details"}]}}`
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	changedTime := info.ModTime().Add(time.Second)
	if err := os.Chtimes(path, changedTime, changedTime); err != nil {
		t.Fatal(err)
	}
	changedResult := scanner.Scan(t.Context())
	if len(changedResult.Sessions) != 1 || changedResult.Sessions[0].Title != "Other task" {
		t.Fatalf("changed file did not invalidate cache: %#v", changedResult.Sessions)
	}

	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	removedResult := scanner.Scan(t.Context())
	if len(removedResult.Sessions) != 0 {
		t.Fatalf("removed file remained in scan: %#v", removedResult.Sessions)
	}
	db, err := openFileSessionCache(scanner.CachePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	entries, err := loadCachedFiles(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("removed file remained in cache: %#v", entries)
	}
}
