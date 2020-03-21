package bootloader

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"sync/atomic"
)

const (
	stateAddingTO int32 = 1 << iota
	stateCreated
)

type wrappedModuler struct {
	name     string
	m        Moduler
	refValue reflect.Value
	state    int32
	cond     *sync.Cond
}

func (wrap *wrappedModuler) waitCreate() {
	for {
		if atomic.LoadInt32(&wrap.state)&stateCreated == stateCreated {
			return
		}
		wrap.cond.Wait()
	}
}

func (wrap *wrappedModuler) create(loader *bootloader) {
	creater, ok := wrap.m.(Creater)
	if ok {
		loader.log("[Bootloader] moduler", wrap.name, "creating")
		creater.OnCreate()
		loader.log("[Bootloader] moduler", wrap.name, "created")
		atomic.StoreInt32(&wrap.state, stateCreated)
		wrap.cond.Broadcast()
	} else {
		atomic.StoreInt32(&wrap.state, stateCreated)
	}
}

func (wrap *wrappedModuler) start(loader *bootloader) {
	if atomic.LoadInt32(&wrap.state)&stateCreated != stateCreated {
		panic(fmt.Errorf("[Bootloader] moduler %v not created", wrap.m))
	}
	starter, ok := wrap.m.(Starter)
	if !ok {
		return
	}
	loader.g.Go(func() error {
		log.Println("[Bootloader] moduler", wrap.name, "starting")
		starter.OnStart()
		log.Println("[Bootloader] moduler", wrap.name, "started")
		return nil
	})
}

func (wrap *wrappedModuler) destroy(loader *bootloader) {
	if atomic.LoadInt32(&wrap.state)&stateCreated != stateCreated {
		return
	}
	destroyer, ok := wrap.m.(Destroyer)
	if !ok {
		return
	}
	loader.g.Go(func() error {
		log.Println("[Bootloader] moduler", wrap.name, "destroying")
		destroyer.OnDestroy()
		log.Println("[Bootloader] moduler", wrap.name, "destroyed")
		return nil
	})

}

type Provider interface {
	GetModuler() (interface{}, error)
}

type ProviderFunc func() (interface{}, error)

func (f ProviderFunc) GetModuler() (interface{}, error) {
	return f()
}

type Moduler interface {
}

type Creater interface {
	Moduler
	OnCreate()
}

type Destroyer interface {
	Moduler
	OnDestroy()
}

type Starter interface {
	Moduler
	OnStart()
}
