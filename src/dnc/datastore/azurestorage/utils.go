package azurestorage

import "strings"

// BuildFilterOnProperty builds a Storage/Cosmos OData query.
// It filters on a partition, given a property and values to match on.
func BuildFilterOnProperty(partitionKey, property string, values []string) string {
	var b strings.Builder
	b.WriteString("PartitionKey eq '")
	b.WriteString(partitionKey)
	b.WriteString("'")

	if len(values) == 0 {
		return b.String()
	}

	b.WriteString(" and ")

	for i, val := range values {
		if i != 0 {
			b.WriteString(" or ")
		}
		b.WriteString(property)
		b.WriteString(" eq '")
		b.WriteString(val)
		b.WriteString("'")
	}

	return b.String()
}

// BuildFilterOnPartition builds a Storage/Cosmos OData query.
// It filters on a partition only
func BuildFilterOnPartition(partitionKey string) string {
	var b strings.Builder
	b.WriteString("PartitionKey eq '")
	b.WriteString(partitionKey)
	b.WriteString("'")

	return b.String()
}
