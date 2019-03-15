package l10n

import (
	"github.com/aghape/db"
	"github.com/aghape/plug"
)

type Plugin struct {
	db.DBNames
	plug.EventDispatcher
}

func (p *Plugin) OnRegister() {
	db.Events(p).DBOnInitGorm(func(e *db.DBEvent) {
		RegisterCallbacks(e.DB.DB)
	})
}
