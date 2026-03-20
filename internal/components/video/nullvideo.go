package video

type NullVideo struct {
	*Device
}

func NewNullVideo(name string) *NullVideo {
	device, err := NewDevice(name, DefaultConfig())
	if err != nil {
		panic(err)
	}
	return &NullVideo{Device: device}
}
