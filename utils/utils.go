package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

func BoolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func GetStructFieldJSONNameByName(thing interface{}, name string) string {
	field, err := reflect.TypeOf(thing).FieldByName(name)
	if !err {
		panic(err)
	}
	return GetReflectFieldJSONName(field)
}

func GetReflectFieldJSONName(field reflect.StructField) string {
	fieldName := field.Name

	switch jsonTag := field.Tag.Get("json"); jsonTag {
	case "-":
		fallthrough
	case "":
		return fieldName
	default:
		parts := strings.Split(jsonTag, ",")
		name := parts[0]
		if name == "" {
			name = fieldName
		}
		return name
	}
}

func ExtractEntityAndValueFromURL(url string) (string, string, bool) {
	re := regexp.MustCompile(`^/([^/]+)/([^/]+)/?$`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 2 {
		return fmt.Sprintf("/%s", matches[1]), matches[2], true
	}
	return "", "", false
}
