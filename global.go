package bootloader

var global = newBootloader()

func Get(name string) (interface{}, error) {
	return global.Get(name)
}

func Add(name string, x interface{}) error {
	return global.Add(name, x)
}

func AddFromType(x interface{}) error {
	return global.AddFromType(x)
}

func SetProperties(data interface{}) error {
	return global.SetProperties(data)
}

func GetProperty(name string) (interface{}, bool) {
	return global.GetProperty(name)
}

func Remove(name string) {
	global.Remove(name)
}

func RemoveFromType(x interface{}) {
	global.RemoveFromType(x)
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
