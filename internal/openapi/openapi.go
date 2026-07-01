package openapi

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/Kushagra1122/gapi/internal/binding"
)

type Config struct {
	Title           string
	Version         string
	SecuritySchemes []SecurityScheme
}

type Operation struct {
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Status      int
	Security    []string
}

type SecurityScheme struct {
	Name   string
	Type   string
	Scheme string
	In     string
}

type Route struct {
	Method    string
	Path      string
	Operation Operation
	Input     reflect.Type
	Output    reflect.Type
}

func Spec(config Config, routes []Route) map[string]any {
	paths := map[string]any{}
	for _, route := range routes {
		pathItem, _ := paths[route.Path].(map[string]any)
		if pathItem == nil {
			pathItem = map[string]any{}
			paths[route.Path] = pathItem
		}

		operation := map[string]any{
			"responses": map[string]any{
				strconv.Itoa(route.Operation.Status): map[string]any{
					"description": http.StatusText(route.Operation.Status),
					"content": map[string]any{
						"application/json": map[string]any{
							"schema": schemaFor(route.Output),
						},
					},
				},
				"400": problemResponse("Bad Request"),
				"422": problemResponse("Unprocessable Entity"),
				"500": problemResponse("Internal Server Error"),
			},
		}
		if route.Operation.OperationID != "" {
			operation["operationId"] = route.Operation.OperationID
		}
		if route.Operation.Summary != "" {
			operation["summary"] = route.Operation.Summary
		}
		if route.Operation.Description != "" {
			operation["description"] = route.Operation.Description
		}
		if len(route.Operation.Tags) > 0 {
			operation["tags"] = route.Operation.Tags
		}
		if len(route.Operation.Security) > 0 {
			operation["security"] = securityRequirements(route.Operation.Security)
		}

		parameters, requestBody := input(route.Input)
		if len(parameters) > 0 {
			operation["parameters"] = parameters
		}
		if requestBody != nil {
			operation["requestBody"] = requestBody
		}

		pathItem[strings.ToLower(route.Method)] = operation
	}

	components := map[string]any{
		"schemas": map[string]any{
			"Problem": problemSchema(),
		},
	}
	if len(config.SecuritySchemes) > 0 {
		components["securitySchemes"] = securitySchemes(config.SecuritySchemes)
	}

	spec := map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":   config.Title,
			"version": config.Version,
		},
		"paths":      paths,
		"components": components,
	}
	return spec
}

func securityRequirements(names []string) []map[string][]string {
	requirements := make([]map[string][]string, 0, len(names))
	for _, name := range names {
		requirements = append(requirements, map[string][]string{name: []string{}})
	}
	return requirements
}

func securitySchemes(schemes []SecurityScheme) map[string]any {
	out := map[string]any{}
	for _, scheme := range schemes {
		item := map[string]any{"type": scheme.Type}
		switch scheme.Type {
		case "http":
			item["scheme"] = scheme.Scheme
		case "apiKey":
			item["in"] = scheme.In
			item["name"] = scheme.Scheme
		}
		out[scheme.Name] = item
	}
	return out
}

func input(t reflect.Type) ([]map[string]any, map[string]any) {
	t = binding.DereferenceType(t)
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	var parameters []map[string]any
	var requestBody map[string]any
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if name, ok := field.Tag.Lookup("path"); ok {
			parameters = append(parameters, parameterSchema(name, "path", field.Type))
		}
		if name, ok := field.Tag.Lookup("query"); ok {
			parameters = append(parameters, parameterSchema(name, "query", field.Type))
		}
		if name, ok := field.Tag.Lookup("header"); ok {
			parameters = append(parameters, parameterSchema(name, "header", field.Type))
		}
		if name, ok := field.Tag.Lookup("cookie"); ok {
			parameters = append(parameters, parameterSchema(name, "cookie", field.Type))
		}
		if _, ok := field.Tag.Lookup("body"); ok {
			requestBody = map[string]any{
				"required": true,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": schemaFor(field.Type),
					},
				},
			}
		}
	}

	return parameters, requestBody
}

func parameterSchema(name, location string, t reflect.Type) map[string]any {
	parameter := map[string]any{
		"name":   name,
		"in":     location,
		"schema": schemaFor(t),
	}
	if location == "path" {
		parameter["required"] = true
	}
	return parameter
}

func schemaFor(t reflect.Type) map[string]any {
	t = binding.DereferenceType(t)

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer", "minimum": 0}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": schemaFor(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaFor(t.Elem())}
	case reflect.Struct:
		if t.PkgPath() == "time" && t.Name() == "Time" {
			return map[string]any{"type": "string", "format": "date-time"}
		}
		properties := map[string]any{}
		var required []string
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			name := jsonFieldName(field)
			if name == "-" {
				continue
			}
			fieldSchema := schemaFor(field.Type)
			if description := field.Tag.Get("doc"); description != "" {
				fieldSchema["description"] = description
			}
			if format := field.Tag.Get("format"); format != "" {
				fieldSchema["format"] = format
			}
			if example := field.Tag.Get("example"); example != "" {
				fieldSchema["example"] = example
			}
			if defaultValue := field.Tag.Get("default"); defaultValue != "" {
				fieldSchema["default"] = defaultValue
			}
			if enum := field.Tag.Get("enum"); enum != "" {
				fieldSchema["enum"] = splitEnum(enum)
			}
			applyValidation(fieldSchema, field.Tag.Get("validate"))
			if hasRequired(field.Tag.Get("validate")) {
				required = append(required, name)
			}
			properties[name] = fieldSchema
		}
		schema := map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	default:
		return map[string]any{}
	}
}

func problemResponse(description string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			"application/problem+json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/Problem"},
			},
		},
	}
}

func problemSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"required": []string{
			"type",
			"title",
			"status",
		},
		"properties": map[string]any{
			"type":   map[string]any{"type": "string"},
			"title":  map[string]any{"type": "string"},
			"status": map[string]any{"type": "integer"},
			"detail": map[string]any{"type": "string"},
			"errors": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field":   map[string]any{"type": "string"},
						"message": map[string]any{"type": "string"},
						"code":    map[string]any{"type": "string"},
					},
				},
			},
		},
	}
}

func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return field.Name
	}
	return name
}

func applyValidation(schema map[string]any, validateTag string) {
	for _, part := range strings.Split(validateTag, ",") {
		part = strings.TrimSpace(part)
		if part == "" || part == "required" {
			continue
		}
		code, value, _ := strings.Cut(part, "=")
		switch code {
		case "min":
			schema["minimum"] = parseNumber(value)
			schema["minLength"] = parseInteger(value)
		case "max":
			schema["maximum"] = parseNumber(value)
			schema["maxLength"] = parseInteger(value)
		case "len":
			length := parseInteger(value)
			schema["minLength"] = length
			schema["maxLength"] = length
		case "email", "uuid":
			schema["format"] = code
		case "oneof":
			schema["enum"] = strings.Fields(value)
		case "regexp":
			schema["pattern"] = value
		}
	}
}

func hasRequired(validateTag string) bool {
	for _, part := range strings.Split(validateTag, ",") {
		if strings.TrimSpace(part) == "required" {
			return true
		}
	}
	return false
}

func splitEnum(value string) []string {
	normalized := strings.ReplaceAll(value, ",", " ")
	return strings.Fields(normalized)
}

func parseNumber(value string) float64 {
	number, _ := strconv.ParseFloat(value, 64)
	return number
}

func parseInteger(value string) int {
	number, _ := strconv.Atoi(value)
	return number
}
