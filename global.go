package bootloader

var global = newBootloader()

func GetModuler(name string) (Moduler, error) {
	return global.GetModuler(name)
}

func AddModuler(name string, m Moduler) error {
	return global.AddModuler(name, m)
}

func AddModuler2(name string, fn func() (Moduler, error)) error {
	return global.AddModuler2(name, fn)
}

func AddModulerFromType(m Moduler) error {
	return global.AddModulerFromType(m)
}

func AddModulerFromType2(fn func() (Moduler, error)) error {
	return global.AddModulerFromType2(fn)
}

func SetProperties(data interface{}) error {
	return global.SetProperties(data)
}

func GetProperty(name string) (interface{}, bool) {
	return global.GetProperty(name)
}

func RemoveModuler(name string) {
	global.RemoveModuler(name)
}

func RemoveModulerFromType(m Moduler) {
	global.RemoveModulerFromType(m)
}

func Launch() error {
	return global.Launch()
}

func ShowLog(b bool) {
	global.ShowLog(b)
}

func Shutdown() error {
	return global.Shutdown()
}
