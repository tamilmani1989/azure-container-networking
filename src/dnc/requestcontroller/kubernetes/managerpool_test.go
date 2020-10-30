package kubernetes

import "testing"

func TestGetRemovedItems(t *testing.T) {
	old := []string{"a", "b", "c"}
	newer := []string{"b", "c"}

	removed := getRemovedItems(old, newer)

	if len(removed) != 1 || removed[0] != "a" {
		t.Fatalf("getRemovedItems() failed, got %v but expected %v", removed, []string{"a"})
	}
}
