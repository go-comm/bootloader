package bootloader

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
)

const (
	typeNamePrefix   = "type-"
	propNamePrefix   = "prop-"
	structTag        = "bloader"
	structTagAutoVal = "auto"
)

func newBootloader() *bootloader {
	loader := new(bootloader)
	ctx := context.Background()
	ctx, loader.cancel = context.WithCancel(context.Background())
	loader.g, _ = errgroup.WithContext(ctx)
	loader.props = newProperties(propNamePrefix)
	loader.showLog = true
	return loader
}

type Bootloader interface {
	AddModuler(string, Moduler) error
	AddModulerFromType(Moduler) error
	Launch() error
	Shutdown() error
	ShowLog(bool)
}

type bootloader struct {
	showLog bool
	cancel  context.CancelFunc
	g       *errgroup.Group
	data    sync.Map
	props   *properties
}

func (loader *bootloader) GetModuler(name string) (Moduler, error) {
	v, ok := loader.data.Load(name)
	if !ok {
		panic(fmt.Errorf("[Bootloader] GetModuler %s not found", name))
	}
	wrap, ok := v.(*wrappedModuler)
	if !ok || wrap.m == nil {
		panic(fmt.Errorf("[Bootloader] GetModuler %s is not type of Moduler", name))
	}
	return wrap.m, nil
}

func (loader *bootloader) ExtractModuler(m Moduler) (Moduler, error) {
	var err error
	var ok bool
	var getter ModulerGetter
	for {
		if getter, ok = m.(ModulerGetter); !ok {
			break
		}
		m, err = getter.GetModuler()
		if err != nil {
			return nil, err
		}
		if getter == m {
			break
		}
	}
	return m, nil
}

func (loader *bootloader) AddModuler2(name string, fn func() (Moduler, error)) error {
	return loader.AddModuler(name, ModulerGetterFunc(fn))
}

func (loader *bootloader) AddModuler(name string, m Moduler) error {
	m, err := loader.ExtractModuler(m)
	if err != nil {
		panic(err)
	}
	wrap := &wrappedModuler{name, m, reflect.ValueOf(m), stateAddingTO, sync.NewCond(&sync.Mutex{})}
	loader.data.Store(name, wrap)

	loader.inject(name, wrap.refValue)
	loader.log("[Bootloader] moduler", name, "added")

	wrap.create(loader)
	wrap.start(loader)

	return nil
}

func (loader *bootloader) inject(name string, refv reflect.Value) {
	if refv.Kind() == reflect.Ptr {
		refv = refv.Elem()
	}
	if refv.Kind() != reflect.Struct {
		return
	}
	reft := refv.Type()
	for i := 0; i < reft.NumField(); i++ {
		ft := reft.Field(i)
		fv := refv.Field(i)

		tag := strings.TrimSpace(ft.Tag.Get(structTag))
		fset := false
		if tag == "" {
			continue
		} else if tag[0] == '$' {
			shell, _ := getShellName(tag[1:])
			props := loader.props
			if props == nil {
				panic(fmt.Errorf("[Bootloader] props not set"))
			}
			if prop := props.value(propNamePrefix + shell); prop != zero {
				fv.Set(prop)
				fset = true
			}
			if !fset {
				panic(fmt.Errorf("[Bootloader] %v:%v Can't assign value to tag %v", reft.Name(), ft.Name, tag))
			}
		} else {
			var name = tag
			if name == structTagAutoVal {
				name = typeNamePrefix + fv.Type().String()
			}
			if v, ok := loader.data.Load(name); ok {
				if wrap, ok := v.(*wrappedModuler); ok {
					fv.Set(wrap.refValue)
					fset = true
				}
			}
			if !fset {
				panic(fmt.Errorf("[Bootloader] %v:%v tag %v no found, must be defined before", reft.Name(), ft.Name, tag))
			}
		}

	}
}

func (loader *bootloader) AddModulerFromType2(fn func() (Moduler, error)) error {
	return loader.AddModulerFromType(ModulerGetterFunc(fn))
}

func (loader *bootloader) AddModulerFromType(m Moduler) error {
	m, err := loader.ExtractModuler(m)
	if err != nil {
		panic(err)
	}
	name := reflect.TypeOf(m).String()
	return loader.AddModuler(typeNamePrefix+name, m)
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

func (loader *bootloader) ShowLog(b bool) {
	loader.showLog = b
}

func (loader *bootloader) RemoveModuler(name string) {
	loader.data.Delete(name)
}

func (loader *bootloader) RemoveModulerFromType(m Moduler) {
	name := reflect.TypeOf(m).String()
	loader.RemoveModuler(name)
}

func (loader *bootloader) Launch() (err error) {
	if err = loader.g.Wait(); err != nil {
		loader.log("[Bootloader]", err)
	}

	// handle all destroy
	loader.data.Range(func(k interface{}, v interface{}) bool {
		wrap, ok := v.(*wrappedModuler)
		if ok {
			wrap.destroy(loader)
		}
		return false
	})
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
