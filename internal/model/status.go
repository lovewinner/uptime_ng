package model

import (
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
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
		}
	}

	switch vptr := v.(type) {
	case *[]string:
		if len(s) < 2 || s[0] != '[' {
			return fmt.Errorf("not a json array")
		}
		inner := s[1 : len(s)-1]
		if strings.TrimSpace(inner) == "" {
			*vptr = []string{}
			return nil
		}
		parts := splitJSONArray(inner)
		*vptr = parts
		return nil
	}

	return fmt.Errorf("unsupported parse target")
}

func splitJSONArray(s string) []string {
	var result []string
	var current strings.Builder
	inString := false
	escaped := false
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			current.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			current.WriteByte(c)
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			current.WriteByte(c)
			continue
		}
		if !inString {
			if c == '[' || c == '{' {
				depth++
			} else if c == ']' || c == '}' {
				depth--
			}
			if c == ',' && depth == 0 {
				result = append(result, strings.Trim(strings.TrimSpace(current.String()), "\""))
				current.Reset()
				continue
			}
		}
		current.WriteByte(c)
	}
	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		result = append(result, strings.Trim(strings.TrimSpace(remaining), "\""))
	}
	return result
}