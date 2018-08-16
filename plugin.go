package l10n

import (
	"github.com/aghape/db"
)

type Plugin struct {
	db.DisDBNames
}

func (p *Plugin) OnRegister() {
	p.DBOnInitGorm(func(e *db.GormDBEvent) {
		RegisterCallbacks(e.DB)
	})
}
