package binding

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
)

type Plan struct {
	fields []fieldBinding
}

type Error struct {
	Field   string
	Message string
	Code    string
}

func (err Error) Error() string {
	return err.Field + ": " + err.Message
}

type fieldBinding struct {
	index  int
	name   string
	source string
	def    string
}

func Compile(t reflect.Type) Plan {
	t = DereferenceType(t)
	if t.Kind() != reflect.Struct {
		return Plan{}
	}

	plan := Plan{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		for _, source := range []string{"path", "query", "header", "cookie", "body"} {
			name, ok := field.Tag.Lookup(source)
			if !ok {
				continue
			}
			plan.fields = append(plan.fields, fieldBinding{
				index:  i,
				name:   name,
				source: source,
				def:    field.Tag.Get("default"),
			})
		}
	}
	return plan
}

func (plan Plan) Bind(r *http.Request, target reflect.Value) error {
	target = DereferenceValue(target)
	if !target.IsValid() || target.Kind() != reflect.Struct {
		return nil
	}

	for _, binding := range plan.fields {
		field := target.Field(binding.index)
		switch binding.source {
		case "path":
			raw := r.PathValue(binding.name)
			if raw == "" {
				raw = binding.def
			}
			if raw == "" {
				continue
			}
			if err := setScalar(field, raw); err != nil {
				return Error{Field: "path." + binding.name, Message: err.Error(), Code: "binding"}
			}
		case "query":
			raw := r.URL.Query().Get(binding.name)
			if raw == "" {
				raw = binding.def
			}
			if raw == "" {
				continue
			}
			if err := setScalar(field, raw); err != nil {
				return Error{Field: "query." + binding.name, Message: err.Error(), Code: "binding"}
			}
		case "header":
			raw := r.Header.Get(binding.name)
			if raw == "" {
				raw = binding.def
			}
			if raw == "" {
				continue
			}
			if err := setScalar(field, raw); err != nil {
				return Error{Field: "header." + binding.name, Message: err.Error(), Code: "binding"}
			}
		case "cookie":
			cookie, err := r.Cookie(binding.name)
			raw := ""
			if err == nil {
				raw = cookie.Value
			}
			if raw == "" {
				raw = binding.def
			}
			if raw == "" {
				continue
			}
			if err := setScalar(field, raw); err != nil {
				return Error{Field: "cookie." + binding.name, Message: err.Error(), Code: "binding"}
			}
		case "body":
			if r.Body == nil {
				continue
			}
			err := json.NewDecoder(r.Body).Decode(field.Addr().Interface())
			if errors.Is(err, io.EOF) {
				continue
			}
			if err != nil {
				return Error{Field: "body", Message: err.Error(), Code: "binding"}
			}
		}
	}

	return nil
}

func setScalar(field reflect.Value, raw string) error {
	field = DereferenceValue(field)
	if !field.CanSet() {
		return errors.New("field cannot be set")
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(value)
	default:
		return fmt.Errorf("unsupported scalar type %s", field.Type())
	}
	return nil
}

func DereferenceType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func DereferenceValue(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}
