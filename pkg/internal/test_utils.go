package internal

import (
	"encoding/json"
	"github.com/go-resty/resty"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"regexp"
	"testing"
)

/*
 * Defer the invocation of this method.
 */
func ErrorOnPanic(t *testing.T) {
	if r := recover(); r != nil {
		t.Errorf("Recovered panic [%s]", r)
	}
}

func Dos2Unix(value string) string {
	re := regexp.MustCompile("\r\n")
	return re.ReplaceAllLiteralString(value, "\n")
}

func GetTestDir() (string, error) {
	return filepath.Abs(".")
}

func Marshal(value interface{}) ([]byte, error) {
	return json.MarshalIndent(value, "", "    ")
}

func MockRestyResponseStatus(statusCode int) *resty.Response {
	return &resty.Response{
		RawResponse: &http.Response{
			StatusCode: statusCode,
		},
	}
}

func ReadRelativeFile(path string) (result []byte, err error) {
	absolutePath, err := filepath.Abs(path)

	if err == nil {
		result, err = ioutil.ReadFile(absolutePath)
	}

	return result, err
}

func Stringify(value interface{}) (string, error) {
	bytes, err := Marshal(value)
	result := ""

	if err == nil {
		result = StringifyBytes(bytes)
	}

	return result, err
}

func StringifyBytes(bytes []byte) string {
	result := string(bytes)
	return Dos2Unix(result)
}

func StringifyRelativeFile(path string) (result string, err error) {
	bytes, err := ReadRelativeFile(path)

	if err == nil {
		result = string(bytes)
		result = Dos2Unix(result)
	}

	return result, err
}
