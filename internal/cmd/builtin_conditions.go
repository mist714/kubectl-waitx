package cmd

import (
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
)

func toStrings[T ~string](values ...T) []string {
	out := make([]string, len(values))
	for i, value := range values {
		out[i] = string(value)
	}
	return out
}

func resourceKind[T runtime.Object]() string {
	return reflect.TypeFor[T]().Elem().Name()
}
