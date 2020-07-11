package bootloader

import "testing"

var global = newBootloader()

func Global() Bootloader {
	return global
}

func Get(name string) (interface{}, error) {
	return global.Get(name)
}

func Add(name string, x interface{}) error {
	return global.Add(name, x)
}

func AddFromType(x interface{}) error {
	return global.AddFromType(x)
}

func AddByAuto(x interface{}) error {
	return global.AddByAuto(x)
}

func SetIgnores(name ...string) error {
	return global.SetIgnores(name...)
}

func SetProperties(data interface{}) error {
	return global.SetProperties(data)
}

func GetProperty(name string) (interface{}, bool) {
	return global.GetProperty(name)
}

func MuestGetProperty(name string) interface{} {
	return global.MuestGetProperty(name)
}

func Launch() error {
	return global.Launch()
}

func TestUnit(fn func() error) error {
	return global.TestUnit(fn)
}

func Run() error {
	return global.Run()
}

func Wait() error {
	return global.Wait()
}

func ShowLog(b bool) {
	global.ShowLog(b)
}

func Shutdown() error {
	return global.Shutdown()
}

func AssertNil(t *testing.T, fn func() error) {
	global.AssertNil(t, fn)
}
