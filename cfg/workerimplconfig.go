package cfg

import (
	"fmt"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// The configuration for a worker implementation. This must at least have "implementation",
// plus any additional worker implementation-specific properties.
type WorkerImplementationConfig struct {
	Implementation string
	data           map[string]interface{}
}

func (pc *WorkerImplementationConfig) UnmarshalYAML(node *yaml.Node) error {
	err := node.Decode(&pc.data)
	if err != nil {
		return err
	}

	pt, ok := pc.data["implementation"]
	if !ok {
		return fmt.Errorf("worker implementation config must have an `implementation` property")
	}

	pc.Implementation, ok = pt.(string)
	if !ok {
		return fmt.Errorf("worker implementation config's `implementation` property must be a string")
	}
	delete(pc.data, "implementation")

	return nil
}

// Unpack this WorkerImplementationConfig to a worker implementation's configuration struct.  This will produce
// an error for any missing properties.  Note that recursion is not supported.
//
// Structs should be tagged with `workerimpl:"name"`, with the name defaulting to the
// lowercased version of the field name.
func (pc *WorkerImplementationConfig) Unpack(out interface{}) error {
	outval := reflect.ValueOf(out)
	if outval.Kind() != reflect.Ptr || outval.IsNil() {
		return fmt.Errorf("expected a pointer, got %s", outval.Kind())
	}
	destval := reflect.Indirect(outval)
	if destval.Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to struct, got &%s", destval.Kind())
	}
	desttype := destval.Type()
	numfield := desttype.NumField()
	for i := 0; i < numfield; i++ {
		// get the expected property name
		field := desttype.Field(i)
		var name string
		tag := field.Tag.Get("workerimpl")
		if tag == "" {
			name = strings.ToLower(field.Name[:1]) + field.Name[1:]
		} else {
			name = tag
		}

		// get the value
		val, ok := pc.data[name]
		if !ok {
			return fmt.Errorf("Configuration value `worker.%s` not found", name)
		}

		// check types and set the struct field
		destfield := destval.Field(i)
		gotval := reflect.ValueOf(val)
		if destfield.Type() != gotval.Type() {
			return fmt.Errorf("Configuration value `worker.%s` should have type %s, got %s", name, destfield.Type(), gotval.Type())
		}
		destfield.Set(gotval)
	}
	return nil
}
