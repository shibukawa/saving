package saving

type HealthStatus int

const (
	InitialChecking HealthStatus = iota + 1
	Healthy
	CheckFailed
	Unhealthy
	Timeout
)

func (s HealthStatus) NotBad() bool {
	return s != Unhealthy
}

func (s HealthStatus) NotGood() bool {
	return s == Unhealthy
}

func (s HealthStatus) GoString() string {
	switch s {
	case InitialChecking:
		return "InitialChecking"
	case Healthy:
		return "Healthy"
	case CheckFailed:
		return "CheckFailed"
	case Unhealthy:
		return "Unhealthy"
	case Timeout:
		return "Timeout"
	default:
		return "Unknown"
	}
}
