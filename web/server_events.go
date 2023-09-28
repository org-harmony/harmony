package web

import (
	"github.com/org-harmony/harmony/core"
)

type ServerSetupEvent struct {
	S Server
}

func (e *ServerSetupEvent) ID() string {
	return core.BuildEventID(ServerMod, "server", "setup")
}

func (e *ServerSetupEvent) Payload() any {
	return e
}

type ServerStartEvent struct {
	S Server
}

func (e *ServerStartEvent) ID() string {
	return core.BuildEventID(ServerMod, "server", "start")
}

func (e *ServerStartEvent) Payload() any {
	return e
}
