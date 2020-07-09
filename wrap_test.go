package bootloader

import "testing"

func Test_ModuleHandler_TravelFields(t *testing.T) {

	var data struct {
		Filed1 int `bloader:"Filed1"`
		Filed2 int
		Filed3 string `bloader:"Filed3"`
	}

	m := newWrappedModule(&data)

	for _, f := range m.Fields() {
		t.Logf("%+v", f)
	}

}
