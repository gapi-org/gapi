package core

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
)

type DependencyResolver interface {
	dependencyName() string
	resolveDependency(context.Context, *http.Request) (any, error)
}

// Dependency resolves a request-scoped value that can be injected into input structs.
type Dependency[T any] struct {
	Name     string
	Resolver func(context.Context, *http.Request) (T, error)
}

// Dep creates a named request dependency.
func Dep[T any](name string, resolver func(context.Context, *http.Request) (T, error)) Dependency[T] {
	return Dependency[T]{Name: name, Resolver: resolver}
}

func (dependency Dependency[T]) dependencyName() string {
	return dependency.Name
}

func (dependency Dependency[T]) resolveDependency(ctx context.Context, r *http.Request) (any, error) {
	return dependency.Resolver(ctx, r)
}

// Provide registers app-level dependencies.
func (app *App) Provide(dependencies ...DependencyResolver) {
	if app.dependencies == nil {
		app.dependencies = map[string]DependencyResolver{}
	}
	for _, dependency := range dependencies {
		app.dependencies[dependency.dependencyName()] = dependency
	}
}

// Require registers route-level dependencies.
func Require(dependencies ...DependencyResolver) OperationOption {
	return func(operation *Operation) {
		operation.dependencies = append(operation.dependencies, dependencies...)
	}
}

type dependencyPlan struct {
	fields []dependencyField
}

type dependencyField struct {
	index int
	name  string
}

func compileDependencyPlan(t reflect.Type) dependencyPlan {
	t = dereferenceType(t)
	if t.Kind() != reflect.Struct {
		return dependencyPlan{}
	}

	plan := dependencyPlan{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if name := field.Tag.Get("dep"); name != "" {
			plan.fields = append(plan.fields, dependencyField{index: i, name: name})
		}
	}
	return plan
}

func (plan dependencyPlan) resolve(ctx context.Context, r *http.Request, target reflect.Value, dependencies map[string]DependencyResolver) error {
	target = dereferenceValue(target)
	cache := map[string]any{}
	for _, field := range plan.fields {
		dependency, ok := dependencies[field.name]
		if !ok {
			return fmt.Errorf("dependency %q is not registered", field.name)
		}
		value, ok := cache[field.name]
		if !ok {
			resolvedValue, err := dependency.resolveDependency(ctx, r)
			if err != nil {
				return err
			}
			value = resolvedValue
			cache[field.name] = value
		}
		fieldValue := target.Field(field.index)
		resolved := reflect.ValueOf(value)
		if !resolved.Type().AssignableTo(fieldValue.Type()) {
			return fmt.Errorf("dependency %q returned %s, expected %s", field.name, resolved.Type(), fieldValue.Type())
		}
		fieldValue.Set(resolved)
	}
	return nil
}

func dependencyMap(appDependencies map[string]DependencyResolver, routeDependencies []DependencyResolver) map[string]DependencyResolver {
	dependencies := map[string]DependencyResolver{}
	for name, dependency := range appDependencies {
		dependencies[name] = dependency
	}
	for _, dependency := range routeDependencies {
		dependencies[dependency.dependencyName()] = dependency
	}
	return dependencies
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
