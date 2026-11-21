package web

import (
	"net/http"
	"reflect"
)

func ParseFromForm[T any](r *http.Request, data *T) error {
	val := reflect.ValueOf(*data)
	if val.Type().Kind() != reflect.Struct {
		panic("automatic form parsing is only supported for struct types")
	}

	for field, _ := range val.Fields() {
		if !field.IsExported() {
			// Private fields are never set
			continue
		}

		// switch field.Type.Kind() {
		// case reflect.
		// }
	}
	
	return nil
}
