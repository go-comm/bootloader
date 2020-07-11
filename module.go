package bootloader

type Module interface {
}

type OnCreater interface {
	OnCreate()
}

type OnMounter interface {
	OnMount()
}

type OnDestroyer interface {
	OnDestroy()
}

type OnStarter interface {
	OnStart()
}
