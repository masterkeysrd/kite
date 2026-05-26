package kitexdt

import (
	"github.com/masterkeysrd/kite/devtools/inspector"
	"github.com/masterkeysrd/kite/engine"
	"github.com/masterkeysrd/kite/extras/kitex"
)

type kitexExtension struct{}

func (k kitexExtension) Name() string {
	return "kitex"
}

func (k kitexExtension) GetPayload(eng *engine.Engine) any {
	return kitex.BuildDevToolsSnapshot(eng)
}

// Register registers the kitex devtools extension onto the given inspector.
func Register(insp *inspector.Inspector) {
	if insp != nil {
		insp.RegisterExtension(kitexExtension{})
	}
}
