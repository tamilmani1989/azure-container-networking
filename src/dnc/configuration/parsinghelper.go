package configuration

import (
	"log"
	"strconv"
)

// ParseInt - Converts string to Int var type
func ParseInt(stringVal string) int {
	var intVal int
	var err error
	if intVal, err = strconv.Atoi(stringVal); err != nil {
		log.Fatalf("Error while converting string {%s} to int.\n", stringVal)
	}

	return intVal
}

// ParseUInt16 - Converts string to uint16 var type
func ParseUInt16(stringVal string) uint16 {
	var intVal int
	var err error
	if intVal, err = strconv.Atoi(stringVal); err != nil {
		log.Fatalf("Error while converting string {%s} to int.\n", stringVal)
	}

	return uint16(intVal)
}

// ParseInt64 - Converts string to int64 var type
func ParseInt64(stringVal string) int64 {
	var intVal int64
	var err error
	if intVal, err = strconv.ParseInt(stringVal, 10, 64); err != nil {
		log.Fatalf("Error while converting string {%s} to int64.\n", stringVal)
	}

	return intVal
}

// ParseBool - Converts string to Bool var type
func ParseBool(stringVal string) bool {
	var boolVal bool
	var err error
	if boolVal, err = strconv.ParseBool(stringVal); err != nil {
		log.Fatalf("Error while converting string {%s} to bool.\n", stringVal)
	}

	return boolVal
}
