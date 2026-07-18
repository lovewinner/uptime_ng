package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var defaultAcceptedStatusCodes = []string{"200-299"}

func CheckStatusCode(statusCode int, acceptedStatusCodesJSON string) bool {
	codes := acceptedStatusCodes(acceptedStatusCodesJSON)

	for _, code := range codes {
		if statusCodeRuleMatches(statusCode, code) {
			return true
		}
	}
	return false
}

func acceptedStatusCodes(raw string) []string {
	var codes []string
	if err := parseJSON(raw, &codes); err != nil {
		return defaultAcceptedStatusCodes
	}
	return codes
}

func statusCodeRuleMatches(statusCode int, rule string) bool {
	rule = strings.TrimSpace(rule)
	if strings.Contains(rule, "-") {
		parts := strings.SplitN(rule, "-", 2)
		low, errLow := strconv.Atoi(strings.TrimSpace(parts[0]))
		high, errHigh := strconv.Atoi(strings.TrimSpace(parts[1]))
		return errLow == nil && errHigh == nil && statusCode >= low && statusCode <= high
	}

	val, err := strconv.Atoi(rule)
	return err == nil && statusCode == val
}

func parseJSON(raw string, v any) error {
	s := strings.TrimSpace(raw)
	if s == "" || s == "null" {
		return fmt.Errorf("empty json")
	}
	return json.Unmarshal([]byte(s), v)
}
