package envconf

import (
	"bytes"
	"encoding"
	"go/ast"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/go-courier/reflectx"
)

func EnvVarsFromEnviron(prefix string, envs []string) *EnvVars {
	e := NewEnvVars(prefix)
	for _, kv := range envs {
		keyValuePair := strings.Split(kv, "=")
		if len(keyValuePair) == 2 {
			if strings.HasPrefix(keyValuePair[0], prefix) {
				e.Set(EnvVarFromKeyValue(keyValuePair[0], keyValuePair[1]))
			}
		}
	}
	return e
}

func NewEnvVars(prefix string) *EnvVars {
	e := &EnvVars{
		Prefix: prefix,
	}
	return e
}

type EnvVars struct {
	Prefix string
	Values map[string]*EnvVar
}

func (e *EnvVars) Set(envVar *EnvVar) {
	if e.Values == nil {
		e.Values = map[string]*EnvVar{}
	}
	e.Values[envVar.KeyPath] = envVar
}

func (e *EnvVars) MaskBytes() []byte {
	values := map[string]string{}
	for _, envVar := range e.Values {
		if envVar.Mask != "" {
			values[envVar.Key(e.Prefix)] = envVar.Mask
			continue
		}
		values[envVar.Key(e.Prefix)] = envVar.Value
	}
	return DotEnv(values)
}

func (e *EnvVars) Bytes() []byte {
	values := map[string]string{}
	for _, envVar := range e.Values {
		values[envVar.Key(e.Prefix)] = envVar.Value
	}
	return DotEnv(values)
}

var interfaceTextMarshaller = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
var interfaceTextUnmarshaller = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()

func (e *EnvVars) Len(key string) int {
	maxIdx := -1

	for _, envVar := range e.Values {
		if strings.HasPrefix(envVar.KeyPath, key) {
			v := strings.TrimLeft(envVar.KeyPath, key+"_")
			parts := strings.Split(v, "_")
			i, err := strconv.ParseInt(parts[0], 10, 64)
			if err == nil {
				if int(i) > maxIdx {
					maxIdx = int(i)
				}
			}
		}
	}

	return maxIdx + 1
}

func (e *EnvVars) Get(key string) *EnvVar {
	if e.Values == nil {
		return nil
	}
	return e.Values[key]
}

func NewDotEnvDecoder(envVars *EnvVars) *DotEnvDecoder {
	return &DotEnvDecoder{
		envVars: envVars,
	}
}

type DotEnvDecoder struct {
	envVars *EnvVars
}

func (d *DotEnvDecoder) Decode(v interface{}) error {
	walker := NewPathWalker()
	rv, ok := v.(reflect.Value)
	if !ok {
		rv = reflect.ValueOf(v)
	}
	return d.scanAndSetValue(walker, rv)
}

func (d *DotEnvDecoder) scanAndSetValue(walker *PathWalker, rv reflect.Value) error {
	kind := rv.Kind()

	if kind != reflect.Ptr && rv.CanAddr() {
		if defaultsSetter, ok := rv.Addr().Interface().(interface{ SetDefaults() }); ok {
			defaultsSetter.SetDefaults()
		}
	}

	switch kind {
	case reflect.Ptr:
		if rv.IsNil() {
			rv.Set(reflectx.New(rv.Type()))
		}
		return d.scanAndSetValue(walker, rv.Elem())
	case reflect.Func, reflect.Interface, reflect.Chan, reflect.Map:
		// skip
	default:
		typ := rv.Type()
		if typ.Implements(interfaceTextUnmarshaller) || reflect.PtrTo(typ).Implements(interfaceTextUnmarshaller) {
			v := d.envVars.Get(walker.String())
			if v != nil {
				if err := reflectx.UnmarshalText(rv, []byte(v.Value)); err != nil {
					return err
				}
			}
			return nil
		}

		switch kind {
		case reflect.Array, reflect.Slice:
			n := d.envVars.Len(walker.String())

			if kind == reflect.Slice {
				rv.Set(reflect.MakeSlice(rv.Type(), n, n))
			}

			for i := 0; i < rv.Len(); i++ {
				walker.Enter(i)
				if err := d.scanAndSetValue(walker, rv.Index(i)); err != nil {
					return err
				}
				walker.Exit()
			}

		case reflect.Struct:
			tpe := rv.Type()
			for i := 0; i < rv.NumField(); i++ {
				field := tpe.Field(i)

				flags := (map[string]bool)(nil)
				name := field.Name

				if !ast.IsExported(name) {
					continue
				}

				if tag, ok := field.Tag.Lookup("env"); ok {
					n, fs := tagValueAndFlags(tag)
					if n == "-" {
						continue
					}
					if n != "" {
						name = n
					}
					flags = fs
				}

				inline := flags == nil && reflectx.Deref(field.Type).Kind() == reflect.Struct && field.Anonymous

				if !inline {
					walker.Enter(name)
				}

				if err := d.scanAndSetValue(walker, rv.Field(i)); err != nil {
					return err
				}

				if !inline {
					walker.Exit()
				}
			}
		default:
			v := d.envVars.Get(walker.String())
			if v != nil {
				if err := reflectx.UnmarshalText(rv, []byte(v.Value)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewDotEnvEncoder(envVars *EnvVars) *DotEnvEncoder {
	return &DotEnvEncoder{
		envVars: envVars,
	}
}

type DotEnvEncoder struct {
	envVars  *EnvVars
	flagsSet map[string]map[string]bool
}

func (d *DotEnvEncoder) setFlags(k string, flags map[string]bool) {
	if d.flagsSet == nil {
		d.flagsSet = map[string]map[string]bool{}
	}
	d.flagsSet[k] = flags
}

func (d *DotEnvEncoder) SecurityEncode(v interface{}) ([]byte, error) {
	walker := NewPathWalker()

	rv, ok := v.(reflect.Value)
	if !ok {
		rv = reflect.ValueOf(v)
	}
	if err := d.scan(walker, rv); err != nil {
		return nil, err
	}

	return d.envVars.MaskBytes(), nil
}

func (d *DotEnvEncoder) Encode(v interface{}) ([]byte, error) {
	walker := NewPathWalker()

	rv, ok := v.(reflect.Value)
	if !ok {
		rv = reflect.ValueOf(v)
	}
	if err := d.scan(walker, rv); err != nil {
		return nil, err
	}

	return d.envVars.Bytes(), nil
}

func (d *DotEnvEncoder) scan(walker *PathWalker, rv reflect.Value) error {
	kind := rv.Kind()

	setValue := func(rv reflect.Value) error {
		key := walker.String()

		envVar := &EnvVar{
			KeyPath: key,
		}

		if d.flagsSet != nil {
			if flags, ok := d.flagsSet[key]; ok {
				envVar.ShouldConf = true
				if flags["upstream"] {
					envVar.IsUpstream = true
				}
			}
		}

		if securityStringer, ok := rv.Interface().(SecurityStringer); ok {
			envVar.Mask = securityStringer.SecurityString()
		}

		text, err := reflectx.MarshalText(rv)
		if err != nil {
			return err
		}

		envVar.Value = string(text)

		d.envVars.Set(envVar)
		return nil
	}

	switch kind {
	case reflect.Ptr:
		if rv.IsNil() {
			return nil
		}
		return d.scan(walker, rv.Elem())
	case reflect.Func, reflect.Interface, reflect.Chan, reflect.Map:
		// skip
	default:
		typ := rv.Type()
		if typ.Implements(interfaceTextMarshaller) {
			if err := setValue(rv); err != nil {
				return err
			}
			return nil
		}

		switch kind {
		case reflect.Array, reflect.Slice:
			for i := 0; i < rv.Len(); i++ {
				walker.Enter(i)
				if err := d.scan(walker, rv.Index(i)); err != nil {
					return err
				}
				walker.Exit()
			}
		case reflect.Struct:
			tpe := rv.Type()
			for i := 0; i < rv.NumField(); i++ {
				field := tpe.Field(i)

				flags := (map[string]bool)(nil)
				name := field.Name

				if !ast.IsExported(name) {
					continue
				}

				if tag, ok := field.Tag.Lookup("env"); ok {
					n, fs := tagValueAndFlags(tag)
					if n == "-" {
						continue
					}
					if n != "" {
						name = n
					}
					flags = fs
				}

				inline := flags == nil && reflectx.Deref(field.Type).Kind() == reflect.Struct && field.Anonymous

				if !inline {
					walker.Enter(name)
				}

				if flags != nil {
					d.setFlags(walker.String(), flags)
				}

				if err := d.scan(walker, rv.Field(i)); err != nil {
					return err
				}

				if !inline {
					walker.Exit()
				}
			}
		default:
			if err := setValue(rv); err != nil {
				return err
			}
		}
	}
	return nil
}

func tagValueAndFlags(tagString string) (string, map[string]bool) {
	valueAndFlags := strings.Split(tagString, ",")
	v := valueAndFlags[0]
	tagFlags := map[string]bool{}
	if len(valueAndFlags) > 1 {
		for _, flag := range valueAndFlags[1:] {
			tagFlags[flag] = true
		}
	}
	return v, tagFlags
}

func DotEnv(keyValues map[string]string) []byte {
	buf := bytes.NewBuffer(nil)

	keys := make([]string, 0)
	for k := range keyValues {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		buf.WriteString(k)
		buf.WriteRune('=')
		buf.WriteString(keyValues[k])
		buf.WriteRune('\n')
	}

	return buf.Bytes()
}
