package bullion_main_server_interfaces

type DeviceType string

const (
	DEVICE_TYPE_ANDROID DeviceType = "ANDROID"
	DEVICE_TYPE_IOS     DeviceType = "IOS"
	DEVICE_TYPE_BROWSER DeviceType = "BROWSER"
)

func (s DeviceType) String() string {
	switch s {
	case DEVICE_TYPE_ANDROID:
		return "ANDROID"
	case DEVICE_TYPE_IOS:
		return "IOS"
	case DEVICE_TYPE_BROWSER:
		return "BROWSER"
	}
	return "unknown"
}

func (s DeviceType) IsValid() bool {
	switch s {
	case
		DEVICE_TYPE_ANDROID,
		DEVICE_TYPE_IOS,
		DEVICE_TYPE_BROWSER:
		return true
	}

	return false
}
