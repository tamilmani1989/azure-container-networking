package nmagentcontracts

import "fmt"

// Tag - Defines the tag struct
type Tag struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

func (tag *Tag) String() string {
	return fmt.Sprintf("Name:%s, Type:%s", tag.Name, tag.Type)
}
