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
	// Struct slices go through encodeDocument so that they are written with the
	// firestore tags this package reads back. Handing them to the driver whole
	// would name the keys by its own lowercasing rule, which only happens to
	// agree with the tags while every field name is a single word.
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Struct &&
		v.Type().Elem() != reflect.TypeOf(time.Time{}) {
		docs := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			doc, err := encodeDocument(v.Index(i).Interface())
			if err != nil {
				// Not reachable for a struct element, but falling back to the
				// raw value keeps a future type from silently writing nothing.
				return v.Interface()
			}
			docs = append(docs, doc)
		}
		return docs
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
		// Slices of structs - a product's spec rows and description blocks -
		// decode element by element under the same firestore-tag rules as a
		// top-level document, mirroring how encodeValue writes them.
		if target.Type().Elem().Kind() == reflect.Struct {
			items, err := asSlice(raw)
			if err != nil {
				return err
			}
			values := reflect.MakeSlice(target.Type(), 0, len(items))
			for _, item := range items {
				doc, err := asDocument(item)
				if err != nil {
					return err
				}
				elem := reflect.New(target.Type().Elem())
				if err := decodeDocument(doc, elem.Interface()); err != nil {
					return err
				}
				values = reflect.Append(values, elem.Elem())
			}
			target.Set(values)
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

func asSlice(raw any) ([]any, error) {
	switch v := raw.(type) {
	case primitive.A:
		return []any(v), nil
	case []any:
		return v, nil
	default:
		return nil, fmt.Errorf("unsupported slice value %T", raw)
	}
}

func asDocument(raw any) (bson.M, error) {
	switch v := raw.(type) {
	case bson.M:
		return v, nil
	case map[string]any:
		return bson.M(v), nil
	case bson.D:
		return v.Map(), nil
	default:
		return nil, fmt.Errorf("unsupported document value %T", raw)
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
