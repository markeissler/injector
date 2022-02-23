package jsonutil

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/markeissler/injector/pkg/stringutil"
)

// ConvertUnicodeToASCII converts selected unicode characters to an ascii representation to support intended output.
// For instance, a byte stream may include a unicode representation of the ampersand symbol (`&`) or various formatting
// characters and in many cases ascii is the desired output with a common example being JSON.
func ConvertUnicodeToASCII(data []byte) []byte {
	// less than
	data = bytes.Replace(data, []byte("\\u003c"), []byte("<"), -1)
	// greater than
	data = bytes.Replace(data, []byte("\\u003e"), []byte(">"), -1)
	// ampersand
	data = bytes.Replace(data, []byte("\\u0026"), []byte("&"), -1)
	// backspace
	data = bytes.Replace(data, []byte("\\u0008"), []byte("\\b"), -1)
	// form feed
	data = bytes.Replace(data, []byte("\\u000c"), []byte("\\f"), -1)

	return data
}

// Flatten parses JSON data into a flattened string array of key/value pairs formatted with the provided formatter
// string. The path value determines which part of the object should be plucked for parsing. Flattened keys will consist
// of uppercase characters only with path elements separated by underscores.
//
// For example:
// ```json
// {
//    "environment": {
//        "jwt_token": "006da898a814ff27e55900b8",
//        "new_relic": {
//            "app_name": "test-app",
//            "enabled": "false",
//            "license_key": "1a2fc8a8eeb33ad0bba6cc5c",
//            "log_level": "info"
//        },
//    }
// }
// ```
//
// Given the path `environment` only its children `jwt_token`, `new_relic`, `app_name`, etc. will be inspected. An array
// of key/value strings returned will look like the following, if given the formatter `%s="%s"`:
//
// ```go
// [
// 		`JWT_TOKEN="006da898a814ff27e55900b8"`,
//		`NEW_RELIC_APP_NAME="processor-staging"`,
//		`NEW_RELIC_ENABLED="false"`,
//		`NEW_RELIC_LICENSE_KEY=""`,
//		`NEW_RELIC_LOG_LEVEL="info"`,
//		`SERVER_BASE_URL="https://institutional-api-staging.alphaflow.com/v1"`,
// ]
// ```
func Flatten(jsonBytes []byte, path, formatter string) []string {
	result := gjson.GetBytes(jsonBytes, path)

	return _recursivelyFlatten("", result, formatter)
}

// _recursivelyFlatten is a recursive function that will dig through a gjson.Result and resolve a list of key/value
// strings wherein keys only appear at the top-level and their named are derived from a flattened path.
//
// See: flattenJSON for examples.
func _recursivelyFlatten(parent string, result gjson.Result, formatter string) []string {
	s := make([]string, 0)
	result.ForEach(func(key, value gjson.Result) bool {
		keyName := strings.ToUpper(key.String())
		if !stringutil.IsBlank(parent) {
			keyName = strings.Join([]string{parent, keyName}, "_")
		}
		if value.Type == gjson.JSON {
			s = append(s, _recursivelyFlatten(keyName, value, formatter)...)
		} else {
			s = append(s, fmt.Sprintf(formatter, keyName, value.String()))
		}
		return true
	})

	return s
}
