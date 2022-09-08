package controllers

import (
	"errors"
	"fmt"
)

var ErrEmptyOptionValue = errors.New("empty option value")

func validateOptionValue(val string) error {
	if val != "" {
		return nil
	}

	return ErrEmptyOptionValue
}

func generateIngressPolicyName(prefix string) string {
	return fmt.Sprintf("%s-ingress", prefix)
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
