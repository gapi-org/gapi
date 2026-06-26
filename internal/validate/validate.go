package validate

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type Error struct {
	Fields []FieldError
}

func (err Error) Error() string {
	return "validation failed"
}

type Plan struct {
	fields []fieldPlan
}

type fieldPlan struct {
	index int
	name  string
	rules []rule
	child *Plan
}

type rule struct {
	code  string
	value string
	re    *regexp.Regexp
}

func Compile(t reflect.Type) Plan {
	t = dereferenceType(t)
	if t.Kind() != reflect.Struct {
		return Plan{}
	}

	plan := Plan{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}

		fieldPlan := fieldPlan{
			index: i,
			name:  fieldName(field),
			rules: parseRules(field.Tag.Get("validate"), field.Tag.Get("enum")),
		}

		childType := dereferenceType(field.Type)
		if childType.Kind() == reflect.Struct && childType.PkgPath() != "time" {
			child := Compile(childType)
			fieldPlan.child = &child
		}

		if len(fieldPlan.rules) > 0 || (fieldPlan.child != nil && len(fieldPlan.child.fields) > 0) {
			plan.fields = append(plan.fields, fieldPlan)
		}
	}

	return plan
}

func (plan Plan) Validate(value reflect.Value) error {
	value = dereferenceValue(value)
	if !value.IsValid() || value.Kind() != reflect.Struct {
		return nil
	}

	var fields []FieldError
	plan.validateInto(value, "", &fields)
	if len(fields) > 0 {
		return Error{Fields: fields}
	}
	return nil
}

func (plan Plan) validateInto(value reflect.Value, prefix string, fields *[]FieldError) {
	value = dereferenceValue(value)
	for _, fieldPlan := range plan.fields {
		fieldValue := value.Field(fieldPlan.index)
		name := joinField(prefix, fieldPlan.name)

		for _, rule := range fieldPlan.rules {
			if message, ok := rule.validate(fieldValue); !ok {
				*fields = append(*fields, FieldError{
					Field:   name,
					Message: message,
					Code:    rule.code,
				})
			}
		}

		if fieldPlan.child != nil {
			fieldPlan.child.validateInto(dereferenceValue(fieldValue), name, fields)
		}
	}
}

func (rule rule) validate(value reflect.Value) (string, bool) {
	value = dereferenceValue(value)

	switch rule.code {
	case "required":
		if isZero(value) {
			return "is required", false
		}
	case "min":
		min, _ := strconv.ParseFloat(rule.value, 64)
		if numeric, ok := numericValue(value); ok && numeric < min {
			return "must be at least " + rule.value, false
		}
		if length, ok := lengthValue(value); ok && float64(length) < min {
			return "length must be at least " + rule.value, false
		}
	case "max":
		max, _ := strconv.ParseFloat(rule.value, 64)
		if numeric, ok := numericValue(value); ok && numeric > max {
			return "must be at most " + rule.value, false
		}
		if length, ok := lengthValue(value); ok && float64(length) > max {
			return "length must be at most " + rule.value, false
		}
	case "len":
		want, _ := strconv.Atoi(rule.value)
		if length, ok := lengthValue(value); ok && length != want {
			return "length must be " + rule.value, false
		}
	case "email":
		if value.Kind() == reflect.String && !emailPattern.MatchString(value.String()) {
			return "must be a valid email address", false
		}
	case "uuid":
		if value.Kind() == reflect.String && !uuidPattern.MatchString(value.String()) {
			return "must be a valid UUID", false
		}
	case "oneof":
		if value.Kind() == reflect.String && !contains(strings.Fields(rule.value), value.String()) {
			return "must be one of " + rule.value, false
		}
	case "regexp":
		if value.Kind() == reflect.String && rule.re != nil && !rule.re.MatchString(value.String()) {
			return "must match " + rule.value, false
		}
	}

	return "", true
}

func parseRules(validateTag, enumTag string) []rule {
	var rules []rule
	for _, part := range strings.Split(validateTag, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		code, value, _ := strings.Cut(part, "=")
		parsed := rule{code: code, value: value}
		if code == "regexp" && value != "" {
			if re, err := regexp.Compile(value); err == nil {
				parsed.re = re
			}
		}
		rules = append(rules, parsed)
	}
	if enumTag != "" {
		rules = append(rules, rule{code: "oneof", value: strings.ReplaceAll(enumTag, ",", " ")})
	}
	return rules
}

func fieldName(field reflect.StructField) string {
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

func joinField(prefix, name string) string {
	if prefix == "" {
		return name
	}
	if name == "" {
		return prefix
	}
	return prefix + "." + name
}

func numericValue(value reflect.Value) (float64, bool) {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint()), true
	case reflect.Float32, reflect.Float64:
		return value.Float(), true
	default:
		return 0, false
	}
}

func lengthValue(value reflect.Value) (int, bool) {
	switch value.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return value.Len(), true
	default:
		return 0, false
	}
}

func isZero(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}
	return value.IsZero()
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func dereferenceType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func dereferenceValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

var (
	emailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	uuidPattern  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)
