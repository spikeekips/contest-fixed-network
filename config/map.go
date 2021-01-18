package config

import (
	"reflect"

	"golang.org/x/xerrors"
)

func MergeItem(a, b interface{}) (map[string]interface{}, error) {
	m := mergeItem(reflect.ValueOf(a), reflect.ValueOf(b)).Interface()

	if i, ok := m.(map[string]interface{}); !ok {
		return nil, xerrors.Errorf("merged node config is not type of map[string]interface{}, %T", m)
	} else {
		return i, nil
	}
}

func mergeMap(a, b reflect.Value) reflect.Value {
	n := copyValue(a)

	br := b.MapRange()
	for br.Next() {
		key := reflect.ValueOf(br.Key().Interface())
		var m reflect.Value
		if i := n.MapIndex(key); i.IsValid() {
			m = mergeItem(i, br.Value())
		} else {
			m = copyValue(br.Value())
		}

		setMapIndexStringKey(n, key, m)
	}

	return n
}

func mergeItem(a, b reflect.Value) reflect.Value {
	a = reflect.ValueOf(a.Interface())
	b = reflect.ValueOf(b.Interface())

	if a.Kind() != b.Kind() {
		return copyValue(b)
	}

	switch {
	case !a.IsValid() && !b.IsValid():
		return reflect.Value{}
	case !a.IsValid():
		return copyValue(b)
	case !b.IsValid():
		return a
	}

	switch a.Kind() {
	case reflect.Map:
		return mergeMap(a, b)
	default:
		return copyValue(b)
	}
}

func setMapIndexStringKey(m, k, v reflect.Value) {
	key := k.Interface().(string)
	m.SetMapIndex(reflect.ValueOf(key), v)
}

func copyValue(v reflect.Value) reflect.Value {
	v = reflect.ValueOf(v.Interface())
	switch v.Kind() {
	case reflect.Map:
		n := reflect.ValueOf(map[string]interface{}{})
		vr := v.MapRange()
		for vr.Next() {
			setMapIndexStringKey(n, vr.Key(), copyValue(vr.Value()))
		}

		return n
	case reflect.Slice:
		n := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(v.Interface()).Elem()), 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			n = reflect.Append(n, copyValue(v.Index(i)))
		}

		return n
	default:
		return v
	}
}

func CopyValue(i interface{}) interface{} {
	return copyValue(reflect.ValueOf(i)).Interface()
}
