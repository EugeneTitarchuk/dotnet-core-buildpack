package sealights

import "github.com/cloudfoundry/libbuildpack"

type Launcher struct {
	Log     *libbuildpack.Logger
	Options *SealightsOptions
}

func NewLauncher(log *libbuildpack.Logger, options *SealightsOptions) *Launcher {
	return &Launcher{Log: log, Options: options}
}
