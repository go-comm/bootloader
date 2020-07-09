package bootloader

import (
	"fmt"
	"reflect"
	"strings"
)

func newWrappedModule(i interface{}) *wrappedModule {
	m := &wrappedModule{}
	m.rv = reflect.ValueOf(i)
	m.rt = m.rv.Type()

	m.travelFields()
	return m
}

type wrappedField struct {
	injected bool
	name     string
	tag      string
	rt       reflect.Type
	rv       reflect.Value
}

func (wrapped *wrappedField) SetValue(v reflect.Value) {
	wrapped.rv.Set(v)
	wrapped.injected = true
}

type wrappedModule struct {
	injected bool
	name     string
	rt       reflect.Type
	rv       reflect.Value
	fields   []*wrappedField
}

func (m *wrappedModule) Fields() []*wrappedField {
	return m.fields
}

func (m *wrappedModule) TryInject() bool {
	if len(m.fields) <= 0 || m.injected {
		return false
	}
	need := false
	for i := len(m.fields) - 1; i >= 0; i-- {
		need = !m.fields[i].injected
		if need {
			break
		}
	}
	m.injected = !need
	return need
}

func (m *wrappedModule) MustInject() {
	if len(m.fields) <= 0 || m.injected {
		return
	}
	for i := len(m.fields) - 1; i >= 0; i-- {
		f := m.fields[i]
		if !f.injected {
			panic(fmt.Errorf("bootloader: Module %s, FieldName:%s, The injection was not completed", m.Path(), f.name))
		}
	}
	return
}

func (m *wrappedModule) travelFields() {
	rv := m.rv
	rt := m.rt

	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
		rt = rt.Elem()
	}
	if rv.Kind() == reflect.Struct {
		var fields []*wrappedField
		for i := rv.NumField() - 1; i >= 0; i-- {
			fv := rv.Field(i)
			ft := rt.Field(i)
			tag := ft.Tag.Get(structTag)
			if strings.TrimSpace(tag) != "" {
				f := &wrappedField{
					name: ft.Name,
					tag:  tag,
					rt:   ft.Type,
					rv:   fv,
				}
				fields = append(fields, f)
			}
		}
		m.fields = fields
	}
}

func (m *wrappedModule) Path() string {
	rt := m.rt
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt.PkgPath() + "." + rt.Name()
}
