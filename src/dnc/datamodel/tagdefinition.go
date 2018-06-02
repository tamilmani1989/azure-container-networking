package datamodel

import (
	"fmt"
	"strings"
	"time"
)

// TagDefinition - struct to hold tag definitions.
type TagDefinition struct {
	Name                string
	Type                string
	Values              []string
	CreatedDateTime     time.Time
	LastUpdatedDateTime time.Time
	Version             uint64
}

// String - ToString implementation of tag definition.
func (tagDefinition *TagDefinition) String() string {
	return fmt.Sprintf("TagDefinition:[Name:%s, Type:%s, Value:%s, Version:%d, CreatedDateTime:%s, LastUpdatedDateTime:%s]",
		tagDefinition.Name,
		tagDefinition.Type,
		strings.Join(tagDefinition.Values, ","),
		tagDefinition.Version,
		tagDefinition.CreatedDateTime.String(),
		tagDefinition.LastUpdatedDateTime.String())
}
