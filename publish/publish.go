package publish

import (
	"fmt"
	"net/http"

	"github.com/moisespsena-go/aorm"
	"github.com/ecletus/admin"
	"github.com/ecletus/l10n"
	"github.com/ecletus/publish"
	"github.com/ecletus/core"
)

type availableLocalesInterface interface {
	AvailableLocales() []string
}

type publishableLocalesInterface interface {
	PublishableLocales() []string
}

type editableLocalesInterface interface {
	EditableLocales() []string
}

func getPublishableLocales(req *http.Request, currentUser interface{}) []string {
	if user, ok := currentUser.(publishableLocalesInterface); ok {
		return user.PublishableLocales()
	}

	if user, ok := currentUser.(editableLocalesInterface); ok {
		return user.EditableLocales()
	}

	if user, ok := currentUser.(availableLocalesInterface); ok {
		return user.AvailableLocales()
	}
	return []string{l10n.Global}
}

// RegisterL10nForPublish register l10n language switcher for publish
func RegisterL10nForPublish(Publish *publish.Publish, Admin *admin.Admin) {
	searchHandler := Publish.SearchHandler
	Publish.SearchHandler = func(db *aorm.DB, context *core.Context) *aorm.DB {
		if context != nil {
			if context.Request != nil && context.Request.URL.Query().Get("locale") == "" {
				publishableLocales := getPublishableLocales(context.Request, context.currentUser)
				return searchHandler(db, context).Set("l10n:mode", "unscoped").Scopes(func(db *aorm.DB) *aorm.DB {
					scope := db.NewScope(db.Val)
					if l10n.IsLocalizable(scope) {
						return db.Where(fmt.Sprintf("%v.language_code IN (?)", scope.QuotedTableName()), publishableLocales)
					}
					return db
				})
			}
			return searchHandler(db, context).Set("l10n:mode", "locale")
		}
		return searchHandler(db, context).Set("l10n:mode", "unscoped")
	}

	Admin.RegisterViewPath("github.com/ecletus/l10n/publish/views")

	Admin.RegisterFuncMap("publishable_locales", func(context admin.Context) []string {
		return getPublishableLocales(context.Request, context.currentUser)
	})
}
