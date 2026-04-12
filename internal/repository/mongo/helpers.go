package mongo

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func encodeDocument(value any) (bson.M, error) {
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("encodeDocument expects struct, got %T", value)
	}

	doc := bson.M{}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("firestore")
		if tag == "" || tag == "-" {
			continue
		}
		doc[tag] = encodeValue(v.Field(i))
	}
	return doc, nil
}

func encodeValue(v reflect.Value) any {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		return encodeValue(v.Elem())
	}
	return v.Interface()
}

func decodeDocument(doc bson.M, out any) error {
	outValue := reflect.ValueOf(out)
	if outValue.Kind() != reflect.Pointer || outValue.IsNil() {
		return fmt.Errorf("decodeDocument expects non-nil pointer")
	}

	elem := outValue.Elem()
	elemType := elem.Type()
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		tag := field.Tag.Get("firestore")
		if tag == "" || tag == "-" {
			continue
		}
		raw, ok := doc[tag]
		if !ok {
			continue
		}
		if err := assignValue(elem.Field(i), raw); err != nil {
			return fmt.Errorf("decode field %s: %w", field.Name, err)
		}
	}
	return nil
}

func assignValue(target reflect.Value, raw any) error {
	if !target.CanSet() {
		return nil
	}
	if raw == nil {
		if target.Kind() == reflect.Pointer {
			target.Set(reflect.Zero(target.Type()))
		}
		return nil
	}

	if target.Kind() == reflect.Pointer {
		elem := reflect.New(target.Type().Elem())
		if err := assignValue(elem.Elem(), raw); err != nil {
			return err
		}
		target.Set(elem)
		return nil
	}

	switch target.Type() {
	case reflect.TypeOf(time.Time{}):
		t, err := asTime(raw)
		if err != nil {
			return err
		}
		target.Set(reflect.ValueOf(t))
		return nil
	}

	switch target.Kind() {
	case reflect.String:
		target.SetString(asString(raw))
	case reflect.Bool:
		b, err := asBool(raw)
		if err != nil {
			return err
		}
		target.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := asInt64(raw)
		if err != nil {
			return err
		}
		target.SetInt(n)
	case reflect.Float32, reflect.Float64:
		f, err := asFloat64(raw)
		if err != nil {
			return err
		}
		target.SetFloat(f)
	case reflect.Slice:
		if target.Type().Elem().Kind() == reflect.String {
			values, err := asStringSlice(raw)
			if err != nil {
				return err
			}
			target.Set(reflect.ValueOf(values))
			return nil
		}
		return fmt.Errorf("unsupported slice type: %s", target.Type())
	default:
		return fmt.Errorf("unsupported target type: %s", target.Type())
	}
	return nil
}

func asString(raw any) string {
	switch v := raw.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", raw)
	}
}

func asBool(raw any) (bool, error) {
	switch v := raw.(type) {
	case bool:
		return v, nil
	case string:
		return strconv.ParseBool(v)
	default:
		return false, fmt.Errorf("unsupported bool value %T", raw)
	}
}

func asInt64(raw any) (int64, error) {
	switch v := raw.(type) {
	case int:
		return int64(v), nil
	case int32:
		return int64(v), nil
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case primitive.DateTime:
		return int64(v), nil
	default:
		return 0, fmt.Errorf("unsupported int value %T", raw)
	}
}

func asFloat64(raw any) (float64, error) {
	switch v := raw.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("unsupported float value %T", raw)
	}
}

func asStringSlice(raw any) ([]string, error) {
	switch v := raw.(type) {
	case []string:
		return v, nil
	case primitive.A:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, asString(item))
		}
		return out, nil
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, asString(item))
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported string slice value %T", raw)
	}
}

func asTime(raw any) (time.Time, error) {
	switch v := raw.(type) {
	case time.Time:
		return v, nil
	case primitive.DateTime:
		return v.Time(), nil
	case int64:
		return time.UnixMilli(v), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported time value %T", raw)
	}
}
