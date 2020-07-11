package bootloader

import "fmt"

func newInjectionHandler(g *group,
	OnBeforeInjectFieldHook,
	OnAfterInjectFieldHook func(m *wrappedModule, f *wrappedField),
	OnInjectCompleted func(m *wrappedModule)) *injectionHandler {
	return &injectionHandler{
		g, OnBeforeInjectFieldHook, OnAfterInjectFieldHook, OnInjectCompleted,
	}
}

type injectionHandler struct {
	g                       *group
	OnBeforeInjectFieldHook func(m *wrappedModule, f *wrappedField)
	OnAfterInjectFieldHook  func(m *wrappedModule, f *wrappedField)
	OnInjectCompleted       func(m *wrappedModule)
}

func (h *injectionHandler) InjectAll() {
	ls := h.g.List()
	for i := len(ls) - 1; i >= 0; i-- {
		m := ls[i]
		h.Inject(m)
	}
}

func (h *injectionHandler) Inject(m *wrappedModule) {
	if !m.TryInject() {
		return
	}

	for i := len(m.Fields()) - 1; i >= 0; i-- {
		f := m.Fields()[i]
		h.injectField(m, f)
	}

	if !m.TryInject() {
		h.OnInjectCompleted(m)
	}
}

func (h *injectionHandler) injectField(m *wrappedModule, f *wrappedField) {
	defer func() {
		if err := recover(); err != nil {
			panic(fmt.Errorf("bootloader: Module %s, FiledName:%s, %v", m.Path(), f.name, err))
		}
		if h.OnAfterInjectFieldHook != nil {
			h.OnAfterInjectFieldHook(m, f)
		}
	}()
	if h.OnBeforeInjectFieldHook != nil {
		h.OnBeforeInjectFieldHook(m, f)
	}
	if f.tag == structTagAutoVal {
		m := h.g.FindByType(f.rt)
		if m != nil {
			f.SetValue(m.rv)
		}
	} else {
		m := h.g.FindByName(f.tag)
		if m != nil {
			f.SetValue(m.rv)
		}
	}
}

func (h *injectionHandler) Verify() {
	ls := h.g.List()
	for i := len(ls) - 1; i >= 0; i-- {
		ls[i].MustInject()
	}
}
