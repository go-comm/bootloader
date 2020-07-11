package bootloader

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
)

const (
	statusInitial int32 = iota
	statusCreating
	statusCreated
	statusMounting
	statusMounted
	statusStarting
	statusStarted
	statusDestroying
	statusDestroyed
)

func newWrappedModule(i interface{}) *wrappedModule {
	m := &wrappedModule{}
	m.rv = reflect.ValueOf(i)
	m.rt = m.rv.Type()
	m.status = statusInitial
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
	status   int32
	log      Logger
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

func (m *wrappedModule) Create() {
	if !atomic.CompareAndSwapInt32(&m.status, statusInitial, statusCreating) {
		panic(fmt.Errorf("bootloader: Unable to create Module %s, status %d expected %d", m.Path(), atomic.LoadInt32(&m.status), statusInitial))
	}
	creater, _ := m.rv.Interface().(OnCreater)
	if creater != nil {
		m.log.Printf("bootloader: create %s begin", m.Path())
		creater.OnCreate()
		m.log.Printf("bootloader: create %s end", m.Path())
	}
	atomic.StoreInt32(&m.status, statusCreated)
}

func (m *wrappedModule) Mount() {
	if !atomic.CompareAndSwapInt32(&m.status, statusCreated, statusMounting) {
		panic(fmt.Errorf("bootloader: Unable to mount Module %s, status %d expected %d", m.Path(), atomic.LoadInt32(&m.status), statusCreated))
	}
	mounter, _ := m.rv.Interface().(OnMounter)
	if mounter != nil {
		m.log.Printf("bootloader: mount %s begin", m.Path())
		mounter.OnMount()
		m.log.Printf("bootloader: mount %s end", m.Path())
	}
	atomic.StoreInt32(&m.status, statusMounted)
}

func (m *wrappedModule) Start() {
	if !atomic.CompareAndSwapInt32(&m.status, statusMounted, statusStarting) {
		panic(fmt.Errorf("bootloader: Unable to start Module %s, status %d expected %d", m.Path(), atomic.LoadInt32(&m.status), statusMounted))
	}
	starter, _ := m.rv.Interface().(OnStarter)
	if starter != nil {
		m.log.Printf("bootloader: start %s begin", m.Path())
		starter.OnStart()
		m.log.Printf("bootloader: start %s end", m.Path())
	}
	atomic.StoreInt32(&m.status, statusStarted)
}

func (m *wrappedModule) Destroy() {
	if !atomic.CompareAndSwapInt32(&m.status, statusStarted, statusDestroying) {
		panic(fmt.Errorf("bootloader: Unable to destroy Module %s, status %d expected %d", m.Path(), atomic.LoadInt32(&m.status), statusStarted))
	}
	destroyer, _ := m.rv.Interface().(OnDestroyer)
	if destroyer != nil {
		m.log.Printf("bootloader: destroy %s begin", m.Path())
		destroyer.OnDestroy()
		m.log.Printf("bootloader: destroy %s end", m.Path())
	}
	atomic.StoreInt32(&m.status, statusDestroyed)
}
