package bootloader

type Module interface {
}

type OnCreater interface {
	OnCreate()
}

type OnDestroyer interface {
	OnDestroy()
}

type OnStarter interface {
	OnStart()
}
