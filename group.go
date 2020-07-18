package bootloader

import (
	"fmt"
	"reflect"
	"sync"
)

func newGroup(OnBeforeAdding,
	OnAfterAdded func(*wrappedModule)) *group {
	return &group{
		namedDict:      make(map[string]*wrappedModule),
		dict:           make([]*wrappedModule, 0, 15),
		ignores:        make(map[string]struct{}),
		OnBeforeAdding: OnBeforeAdding,
		OnAfterAdded:   OnAfterAdded,
	}
}

type group struct {
	namedDict      map[string]*wrappedModule
	dict           []*wrappedModule
	ignores        map[string]struct{}
	mutex          sync.RWMutex
	OnBeforeAdding func(*wrappedModule)
	OnAfterAdded   func(*wrappedModule)
	log            Logger
}

func (g *group) SetIgnores(name ...string) error {
	g.mutex.Lock()
	for i := len(name) - 1; i >= 0; i-- {
		g.ignores[name[i]] = struct{}{}
	}
	g.mutex.Unlock()
	return nil
}

func (g *group) AddByName(name string, m *wrappedModule) bool {
	g.mutex.RLock()
	_, ignore := g.ignores[name]
	g.mutex.RUnlock()
	if ignore {
		return false
	}
	if g.OnBeforeAdding != nil {
		g.OnBeforeAdding(m)
	}
	g.mutex.Lock()
	if _, ok := g.namedDict[name]; ok {
		g.mutex.Unlock()
		panic(fmt.Errorf("bootloader: %s has been added", name))
	}
	g.namedDict[name] = m
	g.dict = append(g.dict, m)
	g.mutex.Unlock()
	g.log.Println("bootloader: AddByName", m.Path(), "named:", name)
	if g.OnAfterAdded != nil {
		g.OnAfterAdded(m)
	}
	return true
}

func (g *group) AddByType(m *wrappedModule) bool {
	if g.OnBeforeAdding != nil {
		g.OnBeforeAdding(m)
	}
	g.mutex.Lock()
	g.dict = append(g.dict, m)
	g.mutex.Unlock()
	g.log.Println("bootloader: AddByType", m.Path())
	if g.OnAfterAdded != nil {
		g.OnAfterAdded(m)
	}
	return true
}

func (g *group) FindByName(name string) *wrappedModule {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.findByName(name)
}

func (g *group) findByName(name string) *wrappedModule {
	return g.namedDict[name]
}

func (g *group) FindByType(tp reflect.Type) *wrappedModule {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.findByType(tp)
}

func (g *group) findByType(tp reflect.Type) *wrappedModule {
	for _, m := range g.dict {
		if m.rt == tp {
			return m
		}
	}
	for _, m := range g.dict {
		if m.rt.ConvertibleTo(tp) {
			return m
		}
	}
	return nil
}

func (g *group) Verify() {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	for i := len(g.dict) - 1; i >= 0; i-- {
		m := g.dict[i]
		m.MustInject()
	}
}

func (g *group) List() []*wrappedModule {
	g.mutex.RLock()
	var copied = make([]*wrappedModule, len(g.dict))
	n := copy(copied, g.dict)
	g.mutex.RUnlock()
	return copied[:n]
}

func (g *group) ForEach(fn func(x interface{})) []*wrappedModule {
	g.mutex.RLock()
	var copied = make([]*wrappedModule, 0, len(g.dict))
	n := copy(copied, g.dict)
	g.mutex.RUnlock()
	copied = copied[:n]
	for i := len(copied) - 1; i >= 0; i-- {
		fn(copied[i].rv.Interface())
	}
	return copied
}
