package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func CheckStatusCode(statusCode int, acceptedStatusCodesJSON string) bool {
	var codes []string
	if err := parseJSON(acceptedStatusCodesJSON, &codes); err != nil {
		codes = []string{"200-299"}
	}

	for _, c := range codes {
		if strings.Contains(c, "-") {
			parts := strings.SplitN(c, "-", 2)
			low, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
			high, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			if statusCode >= low && statusCode <= high {
				return true
			}
		} else {
			val, err := strconv.Atoi(strings.TrimSpace(c))
			if err == nil && statusCode == val {
				return true
			}
		}
	}
	return false
}

func parseJSON(raw string, v interface{}) error {
	s := strings.TrimSpace(raw)
	if s == "" || s == "null" {
		return fmt.Errorf("empty json")
	}
	return json.Unmarshal([]byte(s), v)
}
