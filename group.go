package bootloader

import (
	"fmt"
	"reflect"
)

func newGroup() *group {
	return &group{
		namedDict: make(map[string]*wrappedModule),
		dict:      make([]*wrappedModule, 0, 15),
	}
}

type group struct {
	namedDict        map[string]*wrappedModule
	dict             []*wrappedModule
	afterInjectHook  func(m *wrappedModule, f *wrappedField)
	beforeInjectHook func(m *wrappedModule, f *wrappedField)
}

func (g *group) AddByName(name string, m *wrappedModule) {
	if _, ok := g.namedDict[name]; ok {
		panic(fmt.Errorf("bootloader: %s has been added", name))
	}
	g.namedDict[name] = m
	g.dict = append(g.dict, m)
}

func (g *group) AddByType(m *wrappedModule) {
	g.dict = append(g.dict, m)
}

func (g *group) findByName(name string) *wrappedModule {
	return g.namedDict[name]
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

func (g *group) InjectAll() {
	for i := len(g.dict) - 1; i >= 0; i-- {
		m := g.dict[i]
		if m.TryInject() {
			g.inject(m)
		}
	}
}

func (g *group) inject(m *wrappedModule) {
	for i := len(m.Fields()) - 1; i >= 0; i-- {
		f := m.Fields()[i]
		g.injectField(m, f)
	}
}

func (g *group) injectField(m *wrappedModule, f *wrappedField) {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("bootloader: Module %s, FiledName:%s, %v", m.Path(), f.name, err))
		}
		if g.afterInjectHook != nil {
			g.afterInjectHook(m, f)
		}
	}()
	if g.beforeInjectHook != nil {
		g.beforeInjectHook(m, f)
	}
	if f.tag == structTagAutoVal {
		m := g.findByType(f.rt)
		if m == nil {
			panic(fmt.Errorf("bootloader: Module %s, FiledName:%s, Type not found", m.Path(), f.name))
		} else {
			f.SetValue(m.rv)
		}
	} else {
		// Tag may be property
		m := g.findByName(f.tag)
		if m != nil {
			f.SetValue(m.rv)
		}
	}
}

func (g *group) Verify() {
	for i := len(g.dict) - 1; i >= 0; i-- {
		m := g.dict[i]
		m.MustInject()
	}
}

func (g *group) List() []*wrappedModule {
	return g.dict
}
