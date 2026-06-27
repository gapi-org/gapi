package core

import (
	"reflect"
	"strings"
)

// Validator can be implemented by input or nested model types for custom validation.
type Validator interface {
	ValidateGapi() []FieldError
}

func validateCustom(value reflect.Value) []FieldError {
	value = dereferenceValidationValue(value)
	if !value.IsValid() {
		return nil
	}

	var fields []FieldError
	validateCustomInto(value, "", &fields)
	return fields
}

func validateCustomInto(value reflect.Value, prefix string, fields *[]FieldError) {
	value = dereferenceValidationValue(value)
	if !value.IsValid() {
		return
	}

	if value.CanInterface() {
		if validator, ok := value.Interface().(Validator); ok {
			for _, field := range validator.ValidateGapi() {
				field.Field = joinValidationField(prefix, field.Field)
				*fields = append(*fields, field)
			}
		}
	}

	if value.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < value.NumField(); i++ {
		structField := value.Type().Field(i)
		if structField.PkgPath != "" {
			continue
		}
		validateCustomInto(value.Field(i), joinValidationField(prefix, validationFieldName(structField)), fields)
	}
}

func dereferenceValidationValue(value reflect.Value) reflect.Value {
	for value.IsValid() && value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}
	return value
}

func validationFieldName(field reflect.StructField) string {
	for _, tagName := range []string{"json", "path", "query", "header", "cookie", "body"} {
		tag, ok := field.Tag.Lookup(tagName)
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name != "" {
			return name
		}
		if tagName == "body" {
			return "body"
		}
	}
	return field.Name
}

func joinValidationField(prefix, name string) string {
	if prefix == "" {
		return name
	}
	if name == "" {
		return prefix
	}
	return prefix + "." + name
}
