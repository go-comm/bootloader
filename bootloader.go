package bootloader

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/sync/errgroup"
)

type Provider interface {
	GetModuler() (interface{}, error)
}

type ProviderFunc func() (interface{}, error)

func (f ProviderFunc) GetModuler() (interface{}, error) {
	return f()
}

const (
	envBLoaderLog    = "APP_BLOADER_LOG"
	typeNamePrefix   = "type-"
	propNamePrefix   = "prop-"
	structTag        = "bloader"
	structTagAutoVal = "auto"
	maxDeep          = 5
)

var (
	envBLoaderLogEnabled = strings.ToLower(os.Getenv(envBLoaderLog)) == "true" ||
		os.Getenv(envBLoaderLog) == ""
)

func newBootloader() Bootloader {
	loader := new(bootloader)
	ctx := context.Background()
	ctx, loader.cancel = context.WithCancel(context.Background())
	loader.errg, _ = errgroup.WithContext(ctx)
	loader.props = newProperties(propNamePrefix)
	loader.showLog = envBLoaderLogEnabled
	loader.g = newGroup()
	loader.g.beforeInjectHook = loader.beforeInjectHook
	loader.g.afterInjectHook = loader.afterInjectHook
	return loader
}

type Bootloader interface {
	Get(name string) (interface{}, error)
	Add(name string, x interface{}) error
	AddFromType(x interface{}) error
	AddByAuto(x interface{}) error
	SetProperties(data interface{}) error
	GetProperty(name string) (interface{}, bool)
	MuestGetProperty(name string) interface{}
	Launch() error
	Shutdown() error
	ShowLog(bool)
}

type bootloader struct {
	showLog bool
	cancel  context.CancelFunc
	errg    *errgroup.Group
	props   *properties
	g       *group
}

func (loader *bootloader) Get(name string) (interface{}, error) {
	m := loader.g.findByName(name)
	if m == nil {
		return nil, fmt.Errorf("bootloader: Module %s not found", name)
	}
	return m.rv.Interface(), nil
}

func (loader *bootloader) MustGet(name string) interface{} {
	i, err := loader.Get(name)
	if err != nil {
		panic(err)
	}
	return i
}

func (loader *bootloader) extractModuler(x interface{}, deep int) (interface{}, error) {
	if deep <= 0 {
		panic(fmt.Errorf("bootloader: Maximum depth %d exceeded", maxDeep))
	}
	switch v := x.(type) {
	case Provider:
		m, err := v.GetModuler()
		if err != nil {
			return nil, err
		}
		if m == x {
			return m, nil
		}
		return loader.extractModuler(m, deep-1)
	case ProviderFunc:
		return v()
	case func() (interface{}, error):
		return v()
	case func() interface{}:
		return v(), nil
	default:
		break
	}
	return x, nil
}

func (loader *bootloader) AddFromType(x interface{}) error {
	return loader.AddByAuto(x)
}

func (loader *bootloader) AddByAuto(x interface{}) error {
	m, err := loader.extractModuler(x, maxDeep)
	if err != nil {
		panic(err)
	}
	wrapped := newWrappedModule(m)
	loader.log("bootloader: AddByType", wrapped.Path())
	loader.g.AddByType(wrapped)
	return nil
}

func (loader *bootloader) Add(name string, x interface{}) error {
	m, err := loader.extractModuler(x, maxDeep)
	if err != nil {
		panic(err)
	}
	wrapped := newWrappedModule(m)
	loader.log("bootloader: AddByName", wrapped.Path(), "named:", name)
	loader.g.AddByName(name, wrapped)
	return nil
}

func (loader *bootloader) SetProperties(data interface{}) error {
	loader.props.set(data)
	return nil
}

func (loader *bootloader) GetProperty(name string) (interface{}, bool) {
	if prop := loader.props.value(name); prop != zero {
		return prop.Interface(), true
	}
	return nil, false
}

func (loader *bootloader) MuestGetProperty(name string) interface{} {
	if i, ok := loader.GetProperty(name); ok {
		return i
	}
	panic(fmt.Errorf("bootloader: property %s not found", name))
}

func (loader *bootloader) ShowLog(b bool) {
	loader.showLog = b
}

func (loader *bootloader) beforeInjectHook(m *wrappedModule, f *wrappedField) {
	// nothing
}

func (loader *bootloader) afterInjectHook(m *wrappedModule, f *wrappedField) {
	tag := f.tag
	if len(tag) > 0 && tag[0] == '$' {
		loader.injectProperty(m, f)
	}
}

func (loader *bootloader) injectProperty(m *wrappedModule, f *wrappedField) {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("bootloader: Module %s, FiledName:%s, %v", m.Path(), f.name, err))
		}
	}()
	shell, _ := getShellName(f.tag[1:])
	props := loader.props
	if props == nil {
		panic(fmt.Errorf("bootloader: props not set"))
	}
	if prop := props.value(shell); prop != zero {
		f.SetValue(prop)
		loader.log("bootloader: setprop", m.Path(), "FieldName:", f.name)
	}
}

func (loader *bootloader) Launch() (err error) {

	// inject
	loader.g.InjectAll()

	// verify all module
	loader.g.Verify()

	// create
	for _, m := range loader.g.List() {
		creater, _ := m.rv.Interface().(OnCreater)
		if creater != nil {
			loader.errg.Go(func() error {
				loader.logf("bootloader: create %s begin", m.Path())
				creater.OnCreate()
				loader.logf("bootloader: create %s end", m.Path())
				return nil
			})
		}
	}
	loader.errg.Wait()

	// start
	for _, m := range loader.g.List() {
		starter, _ := m.rv.Interface().(OnStarter)
		if starter != nil {
			loader.errg.Go(func() error {
				loader.logf("bootloader: start %s begin", m.Path())
				starter.OnStart()
				loader.logf("bootloader: start %s end", m.Path())
				return nil
			})
		}
	}
	loader.errg.Wait()

	// destroy
	for _, m := range loader.g.List() {
		destroyer, _ := m.rv.Interface().(OnDestroyer)
		if destroyer != nil {
			loader.errg.Go(func() error {
				loader.logf("bootloader: destory %s begin", m.Path())
				destroyer.OnDestroy()
				loader.logf("bootloader: destory %s end", m.Path())
				return nil
			})
		}
	}
	return nil
}

func (loader *bootloader) Shutdown() error {
	if loader.cancel != nil {
		loader.cancel()
	}
	return nil
}

func (loader *bootloader) log(v ...interface{}) {
	if loader.showLog {
		log.Println(v...)
	}
}

func (loader *bootloader) logf(format string, v ...interface{}) {
	if loader.showLog {
		log.Printf(format, v...)
	}
}
