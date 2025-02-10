package main

import (
	"bytes"
	"fmt"
	"strings"

	"encoding/json"

	"github.com/joho/godotenv"
)

// jsonから.env形式に順序を維持して変換する
func jsonToDotEnv(s string) (string, error) {
	d := json.NewDecoder(bytes.NewReader([]byte(s)))
	result := bytes.NewBuffer([]byte{})
	startToken, err := d.Token()
	if err != nil {
		return "", err
	}
	if startToken != json.Delim('{') {
		return "", fmt.Errorf("不正なjsonです")
	}
	for {
		key, err := d.Token()
		if err != nil {
			return "", err
		}
		if key == json.Delim('}') {
			break
		}
		if keyString, ok := key.(string); ok {
			value, err := d.Token()
			if err != nil {
				return "", err
			}
			if valueString, ok := value.(string); ok {
				result.WriteString(fmt.Sprintf("%s=%s\n", keyString, valueString))
			} else {
				return "", fmt.Errorf("不正なjsonです")
			}
		} else {
			return "", fmt.Errorf("不正なjsonです")
		}
	}
	return result.String(), nil
}

// .env形式からjsonに順序を維持して変換する
func dotEnvToJson(s string) (string, error) {
	result := bytes.NewBuffer([]byte{})
	result.WriteString("{")
	lines := strings.Split(s, "\n")
	first := true
	for _, line := range lines {
		m, err := godotenv.Parse(bytes.NewReader([]byte(line)))
		if err != nil {
			return "", err
		}
		for key, value := range m {
			if !first {
				result.WriteString(",")
			}
			first = false
			jsonedKey, err := json.Marshal(key)
			if err != nil {
				return "", err
			}
			result.WriteString(string(jsonedKey))
			result.WriteString(":")
			jsonedValue, err := json.Marshal(value)
			if err != nil {
				return "", err
			}
			result.WriteString(string(jsonedValue))
		}
	}
	result.WriteString("}")
	return result.String(), nil
}
