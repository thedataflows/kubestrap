package reflectutil

import (
	"reflect"
	"strings"
)

// GetStructFields gets a list of field names from current struct
func GetStructFieldTag(aStruct any, fieldName, key string) string {
	field, ok := reflect.TypeOf(aStruct).Elem().FieldByName(fieldName)
	if !ok {
		return ""
	}
	if len(key) == 0 {
		key = "yaml"
	}
	return field.Tag.Get(key)
}

// GetStructFields gets a list of field names from current struct
func GetStructFields(aStruct any) []string {
	ref := reflect.TypeOf(aStruct).Elem()
	fields := make([]string, ref.NumField())
	for i := 0; i < ref.NumField(); i++ {
		fields[i] = ref.Field(i).Name
	}
	return fields
}

// GetStructValue gets value from struct field matching case insensitive by field name
func GetStructValue(aStruct any, fieldName string) any {
	name := strings.ToLower(fieldName)
	ref := reflect.ValueOf(aStruct)
	for i := 0; i < ref.Type().NumField(); i++ {
		if strings.ToLower(ref.Type().Field(i).Name) == name {
			return ref.Field(i).Interface()
		}
	}
	return nil
}
