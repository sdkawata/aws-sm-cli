package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/joho/godotenv"
)

func TestJsonToDotEnv(t *testing.T) {
	testCases := []struct {
		from     string
		expected string
	}{
		{
			"{\"key\":\"value\", \"key2\":\"value2\"}",
			"key=value\nkey2=value2\n",
		},
		{
			"{\"key\":\"ここに二重引用符: \\\"value\"}",
			"key=ここに二重引用符: \"value\n",
		},
	}
	for _, testCase := range testCases {
		var parsedJson map[string]string
		err := json.Unmarshal([]byte(testCase.from), &parsedJson)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		parsedDotenv, err := godotenv.Parse(bytes.NewReader([]byte(testCase.expected)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(parsedJson, parsedDotenv) {
			t.Fatalf("test case does not match %v != %v", parsedJson, parsedDotenv)
		}
		actual, err := jsonToDotEnv(testCase.from)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if actual != testCase.expected {
			t.Fatalf("expected: %v, but got: %v", testCase.expected, actual)
		}
	}
}

func TestDotEnvToJson(t *testing.T) {
	testCases := []struct {
		from     string
		expected string
	}{
		{
			"key=value\nkey2=value2\n",
			"{\"key\":\"value\",\"key2\":\"value2\"}",
		},
		{
			"key=ここに二重引用符: \"value\n",
			"{\"key\":\"ここに二重引用符: \\\"value\"}",
		},
	}
	for _, testCase := range testCases {
		parsedDotenv, err := godotenv.Parse(bytes.NewReader([]byte(testCase.from)))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var parsedJson map[string]string
		err = json.Unmarshal([]byte(testCase.expected), &parsedJson)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !reflect.DeepEqual(parsedDotenv, parsedJson) {
			t.Fatalf("test case does not match %v != %v", parsedDotenv, parsedJson)
		}
		actual, err := dotEnvToJson(testCase.from)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if actual != testCase.expected {
			t.Fatalf("expected: %v, but got: %v", testCase.expected, actual)
		}
	}
}
