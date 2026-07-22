package session

import "testing"

func TestFilterRequiresEveryTermAcrossMetadataAndTranscript(t *testing.T) {
	items := []Session{
		{ID: "one", Provider: Codex, SearchText: "codex payments retry queue"},
		{ID: "two", Provider: Claude, SearchText: "claude payments dashboard"},
	}

	got := Filter(items, "payments retry")
	if len(got) != 1 || got[0].ID != "one" {
		t.Fatalf("Filter() = %#v, want only session one", got)
	}
	if got := Filter(items, "  "); len(got) != 2 {
		t.Fatalf("empty Filter() returned %d items, want 2", len(got))
	}
}

func TestSortNewest(t *testing.T) {
	items := []Session{{ID: "older"}, {ID: "newer"}}
	items[1].UpdatedAt = items[0].UpdatedAt.Add(1)
	SortNewest(items)
	if items[0].ID != "newer" {
		t.Fatalf("SortNewest() first = %q, want newer", items[0].ID)
	}
}
