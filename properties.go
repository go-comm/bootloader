package bootloader

import (
	"reflect"
	"strings"
	"sync"
)

type property struct {
	name  string
	value reflect.Value
}

func newProperties(prefix string) *properties {
	return &properties{data: make(map[string]reflect.Value), prefix: prefix}
}

type properties struct {
	data   map[string]reflect.Value
	prefix string
	mutex  sync.RWMutex
}

func (p *properties) set(data interface{}) {
	value, ok := data.(reflect.Value)
	if !ok {
		value = reflect.ValueOf(data)
	}
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.walk("", value)
}

func (p *properties) walk(dot string, data reflect.Value) {
	if data.IsZero() {
		return
	}
	if data.Kind() == reflect.Ptr {
		data = data.Elem()
	}
	dataType := data.Type()
	if data.Kind() == reflect.Struct {
		for i := 0; i < data.NumField(); i++ {
			ft := dataType.Field(i)
			fv := data.Field(i)
			name := dot + "." + strings.ToLower(ft.Name)
			p.data[p.prefix+name[1:]] = fv
			p.walk(name, fv)
		}
	} else if data.Kind() == reflect.Map {
		for _, k := range data.MapKeys() {
			v := data.MapIndex(k)
			if k.Kind() == reflect.String {
				name := dot + "." + strings.ToLower(k.String())
				p.data[p.prefix+name[1:]] = v
				p.walk(name, v)
			}
		}
	}
}

func (p *properties) value(name string) reflect.Value {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if v, ok := p.data[p.prefix+strings.ToLower(name)]; ok {
		return v
	}
	return zero
}

var zero reflect.Value
