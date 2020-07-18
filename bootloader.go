package bootloader

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

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
	loader.log = &logger{}
	loader.log.Show(envBLoaderLogEnabled)
	ctx := context.Background()
	ctx, loader.cancel = context.WithCancel(context.Background())
	loader.errg, _ = errgroup.WithContext(ctx)
	loader.props = newProperties(propNamePrefix)
	loader.g = newGroup(loader.OnBeforeAdding,
		loader.OnAfterAdded)
	loader.g.log = loader.log
	loader.h = newInjectionHandler(loader.g,
		loader.OnBeforeInjectFieldHook,
		loader.OnAfterInjectFieldHook,
		loader.OnInjectCompleted)
	return loader
}

type Bootloader interface {
	Get(name string) (interface{}, error)
	Add(name string, x interface{}) error
	AddFromType(x interface{}) error
	AddByAuto(x interface{}) error
	SetIgnores(name ...string) error
	SetProperties(data interface{}) error
	GetProperty(name string) (interface{}, bool)
	MuestGetProperty(name string) interface{}
	Launch() error
	TestUnit(fn func() error) error
	AssertNil(t *testing.T, fn func() error)
	Run() error
	Wait() error
	Shutdown() error
	ShowLog(bool)
}

type bootloader struct {
	cancel context.CancelFunc
	errg   *errgroup.Group
	props  *properties
	g      *group
	h      *injectionHandler
	log    Logger
}

func (loader *bootloader) Get(name string) (interface{}, error) {
	m := loader.g.FindByName(name)
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
	wrapped.log = loader.log
	loader.g.AddByType(wrapped)
	loader.h.Inject(wrapped)
	return nil
}

func (loader *bootloader) Add(name string, x interface{}) error {
	m, err := loader.extractModuler(x, maxDeep)
	if err != nil {
		panic(err)
	}
	wrapped := newWrappedModule(m)
	wrapped.log = loader.log
	loader.g.AddByName(name, wrapped)
	loader.h.Inject(wrapped)
	return nil
}

func (loader *bootloader) SetIgnores(name ...string) error {
	return loader.g.SetIgnores(name...)
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
	loader.log.Show(b)
}

func (loader *bootloader) OnBeforeAdding(m *wrappedModule) {
	m.Create()
}

func (loader *bootloader) OnAfterAdded(m *wrappedModule) {
	if len(m.Fields()) <= 0 {
		loader.doMount(m)
		return
	}
	loader.h.Inject(m)
}

func (loader *bootloader) doMount(m *wrappedModule) {
	m.Mount()
	loader.errg.Go(func() error {
		m.Start()
		return nil
	})
}

func (loader *bootloader) OnInjectCompleted(m *wrappedModule) {
	loader.doMount(m)
}

func (loader *bootloader) OnBeforeInjectFieldHook(m *wrappedModule, f *wrappedField) {
	// nothing
}

func (loader *bootloader) OnAfterInjectFieldHook(m *wrappedModule, f *wrappedField) {
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
		loader.log.Println("bootloader: setprop", m.Path(), "FieldName:", f.name)
	}
}

func (loader *bootloader) Launch() (err error) {
	err = loader.Run()
	if err != nil {
		return
	}
	return loader.Wait()
}

func (loader *bootloader) TestUnit(fn func() error) (err error) {
	return loader.run(fn)
}

func (loader *bootloader) Run() (err error) {
	return loader.run(nil)
}

func (loader *bootloader) run(fn func() error) (err error) {

	// inject
	loader.h.InjectAll()

	// verify all module
	loader.h.Verify()

	// for test function
	if fn != nil {
		err := fn()
		if err != nil {
			panic(fmt.Sprintf("bootloader: %v", err))
		}
	}
	return nil
}

func (loader *bootloader) Wait() (err error) {
	loader.errg.Wait()
	// destroy
	for _, m := range loader.g.List() {
		destroyer, _ := m.rv.Interface().(OnDestroyer)
		if destroyer != nil {
			loader.errg.Go(func() error {
				destroyer.OnDestroy()
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

func (loader *bootloader) AssertNil(t *testing.T, fn func() error) {
	if err := loader.run(fn); err != nil {
		t.Fatal(err)
	}
}
