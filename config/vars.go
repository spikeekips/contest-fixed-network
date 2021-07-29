package config

import (
	"reflect"
	"regexp"
	"strings"
	"sync"
	"text/template"

	"github.com/spikeekips/mitum/base/key"
	"golang.org/x/xerrors"
)

var (
	reFilterSymbol = regexp.MustCompile(`[\-\_\.]`)
	reFilterBlank  = regexp.MustCompile(`[\s][\s]*`)
)

var TranslateWords = map[string]string{
	"ssh": "SSH",
	"id":  "ID",
	"url": "URL",
	"uri": "URI",
}

type Vars struct {
	sync.RWMutex
	m       map[string]interface{}
	funcMap template.FuncMap
}

func NewVars(m map[string]interface{}) *Vars {
	if m == nil {
		m = map[string]interface{}{}
	}

	vs := &Vars{
		m:       m,
		funcMap: template.FuncMap{},
	}

	return vs
}

func (vs *Vars) Clone(m map[string]interface{}) *Vars {
	vars := func() *Vars {
		vs.RLock()
		defer vs.RUnlock()

		nvars := NewVars(CopyValue(vs.m).(map[string]interface{}))
		nvars.funcMap = vs.funcMap

		return nvars
	}()

	for k := range m {
		vars.Set(k, m[k])
	}

	return vars
}

func (vs *Vars) FuncMap() template.FuncMap {
	vs.RLock()
	defer vs.RUnlock()

	m := template.FuncMap{}
	for k := range vs.funcMap {
		m[k] = vs.funcMap[k]
	}

	base := vs.baseFuncMap()
	for k := range base {
		m[k] = base[k]
	}

	return m
}

func (vs *Vars) AddFunc(name string, f interface{}) *Vars {
	vs.Lock()
	defer vs.Unlock()

	vs.funcMap[name] = f

	return vs
}

func (vs *Vars) Map() map[string]interface{} {
	vs.RLock()
	defer vs.RUnlock()

	return vs.m
}

func (vs *Vars) Exists(keys string) bool {
	vs.RLock()
	defer vs.RUnlock()

	_, found := getVar(vs.m, keys)

	return found
}

func (vs *Vars) Value(keys string) (interface{}, bool) {
	vs.RLock()
	defer vs.RUnlock()

	return getVar(vs.m, keys)
}

func (vs *Vars) Set(keys string, value interface{}) {
	vs.Lock()
	defer vs.Unlock()

	_ = setVar(vs.m, keys, value)
}

func SanitizeVarsMap(m interface{}) interface{} {
	if m == nil {
		return m
	}

	v := reflect.ValueOf(m)
	switch v.Kind() {
	case reflect.Map:
		n := reflect.MakeMapWithSize(
			reflect.MapOf(reflect.TypeOf((*string)(nil)).Elem(), reflect.TypeOf((*interface{})(nil)).Elem()),
			0,
		)

		vr := v.MapRange()
		for vr.Next() {
			n.SetMapIndex(
				reflect.ValueOf(NormalizeVarsKey(vr.Key().Interface().(string))),
				reflect.ValueOf(SanitizeVarsMap(vr.Value().Interface())),
			)
		}

		v = n
	case reflect.Array, reflect.Slice:
		n := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf((*interface{})(nil)).Elem()), 0, v.Len())

		for i := 0; i < v.Len(); i++ {
			n = reflect.Append(n, reflect.ValueOf(SanitizeVarsMap(v.Index(i).Interface())))
		}
		v = n
	}

	return v.Interface()
}

func NormalizeVarsKey(s string) string {
	s = strings.TrimSpace(s)

	for _, f := range []func(i string) string{
		func(i string) string { // NOTE replace hyphen and underscore
			return string(reFilterSymbol.ReplaceAll([]byte(i), []byte(" ")))
		},
		func(i string) string { // NOTE replace 2 word into capitals
			a := ""
			for _, w := range reFilterBlank.Split(i, -1) {
				if x, found := TranslateWords[strings.ToLower(w)]; found {
					w = x
				}

				a += " " + w
			}

			return a
		},
		strings.Title,
		func(i string) string { // NOTE remove blank
			return strings.ReplaceAll(i, " ", "")
		},
	} {
		s = f(s)
	}

	return s
}

func getVar(v interface{}, keys string) (interface{}, bool) {
	m := v
	for _, k := range strings.Split(keys, ".") {
		if i, ok := m.(map[string]interface{}); !ok {
			return nil, false
		} else if j, found := i[k]; !found {
			return nil, false
		} else {
			m = j
		}
	}

	return m, true
}

func setVar(m map[string]interface{}, keys string, v interface{}) error {
	if m == nil {
		return xerrors.Errorf("nil map")
	}

	ks := strings.Split(keys, ".")
	if len(ks) < 2 {
		m[keys] = v

		return nil
	}

	l := m
	for _, k := range ks[:len(ks)-1] {
		if j, found := l[k]; !found {
			l[k] = map[string]interface{}{}
			l = l[k].(map[string]interface{})
		} else {
			l = j.(map[string]interface{})
		}
	}

	l[ks[len(ks)-1]] = v

	return nil
}

func (vs *Vars) baseFuncMap() template.FuncMap {
	return template.FuncMap{
		"TrimSpace": strings.TrimSpace,
		"ExistsVar": func(keys string) interface{} {
			vs.RLock()
			defer vs.RUnlock()

			i, _ := getVar(vs.m, keys)

			return i
		},
		"GetVar": func(keys string) interface{} {
			vs.RLock()
			defer vs.RUnlock()

			i, _ := getVar(vs.m, keys)

			return i
		},
		"SetVar": func(keys string, value interface{}) string {
			vs.Lock()
			defer vs.Unlock()

			if _, found := getVar(vs.m, keys); found {
				return ""
			}

			_ = setVar(vs.m, keys, value)

			return ""
		},
		"OverrideVar": func(keys string, value interface{}) string {
			vs.Lock()
			defer vs.Unlock()

			_ = setVar(vs.m, keys, value)

			return ""
		},
		"NewKey": func(keys, keyType string) key.Privatekey {
			vs.Lock()
			defer vs.Unlock()

			if i, found := getVar(vs.m, keys); found {
				return i.(key.Privatekey)
			}

			k := newKey(keyType)

			_ = setVar(vs.m, keys, k)

			return k
		},
	}
}

func newKey(keyType string) key.Privatekey {
	switch keyType {
	case "ether":
		return key.MustNewEtherPrivatekey()
	case "stellar":
		return key.MustNewStellarPrivatekey()
	default: // "btc"
		return key.MustNewBTCPrivatekey()
	}
}
