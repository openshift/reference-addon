package controllers

import (
	"errors"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var ErrEmptyOptionValue = errors.New("empty option value")

func ValidateOptionValue(val string) error {
	if val != "" {
		return nil
	}

	return ErrEmptyOptionValue
}

func BoolPtr(b bool) *bool {
	return &b
}

func StringPtr(s string) *string {
	return &s
}

func HasNamePrefix(pfx string) predicate.Funcs {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return strings.HasPrefix(obj.GetName(), pfx)
		},
	)
}

func HasName(name string) predicate.Funcs {
	return predicate.NewPredicateFuncs(
		func(obj client.Object) bool {
			return obj.GetName() == name
		},
	)
}
