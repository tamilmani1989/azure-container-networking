package azurestorage

import "testing"

func TestBuildFilterOnPartition(t *testing.T) {
	tests := []struct {
		name     string
		values   []string
		expected string
	}{
		{
			name:     "some properties",
			values:   []string{"a", "b", "c"},
			expected: "PartitionKey eq 'partition1' and property1 eq 'a' or property1 eq 'b' or property1 eq 'c'",
		},
		{
			name:     "no properties",
			values:   nil,
			expected: "PartitionKey eq 'partition1'",
		},
	}

	for _, tt := range tests {
		ts := tt
		t.Run(ts.name, func(t *testing.T) {
			filter := BuildFilterOnPartition("partition1", "property1", ts.values)
			if filter != ts.expected {
				t.Fatalf("expected %s, got %s", ts.expected, filter)
			}
		})
	}
}
