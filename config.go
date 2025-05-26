package saving

type cliVars struct {
	ListenPort int `envvar:"SAVING_LISTEN_PORT" default:"80"`
	TargetPort int `envvar:"SAVING_TARGET_PORT" default:"3000"`

	Timeout  int `envvar:"SAVING_WAKE_TIMEOUT" default:"300"`
	Schedule int `envvar:"SAVING_WAKE_SCHEDULE"`

	HealthCheckPath string `envvar:"SAVING_HEALTH_CHECK_PATH" default:"/health"`
}
