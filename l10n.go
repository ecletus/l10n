package l10n

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strings"

	"github.com/ecletus/admin"
	"github.com/ecletus/core"
	"github.com/ecletus/core/resource"
	"github.com/ecletus/core/utils"
	"github.com/ecletus/roles"
	"github.com/moisespsena-go/xroute"
)

// Global global language
var Global = "en-US"

type l10nInterface interface {
	IsGlobal() bool
	SetLocale(locale string)
}

// Locale embed this struct into GROM-backend models to enable localization feature for your model
type Locale struct {
	LanguageCode string `sql:"size:20" gorm:"primary_key"`
}

// IsGlobal return if current locale is global
func (l Locale) IsGlobal() bool {
	return l.LanguageCode == Global
}

// SetLocale set model's locale
func (l *Locale) SetLocale(locale string) {
	l.LanguageCode = locale
}

// LocaleCreatable if you embed it into your model, it will make the resource be creatable from locales, by default, you can only create it from global
type LocaleCreatable struct {
	Locale
}

// CreatableFromLocale a method to allow your mod=el be creatable from locales
func (LocaleCreatable) CreatableFromLocale() {}

type availableLocalesInterface interface {
	AvailableLocales() []string
}

type viewableLocalesInterface interface {
	ViewableLocales() []string
}

type editableLocalesInterface interface {
	EditableLocales() []string
}

func getAvailableLocales(req *http.Request, currentUser interface{}) []string {
	if user, ok := currentUser.(viewableLocalesInterface); ok {
		return user.ViewableLocales()
	}

	if user, ok := currentUser.(availableLocalesInterface); ok {
		return user.AvailableLocales()
	}
	return []string{Global}
}

func getEditableLocales(req *http.Request, currentUser interface{}) []string {
	if user, ok := currentUser.(editableLocalesInterface); ok {
		return user.EditableLocales()
	}

	if user, ok := currentUser.(availableLocalesInterface); ok {
		return user.AvailableLocales()
	}
	return []string{Global}
}

func getLocaleFromContext(context *core.Context) string {
	if locale := utils.GetLocale(context); locale != "" {
		return locale
	}

	return Global
}

// LocalizeActionArgument localize action's argument
type LocalizeActionArgument struct {
	From string
	To   []string
}

// ConfigureQorResource configure qor locale for Qor Admin
func (l *Locale) ConfigureQorResource(res resource.Resourcer) {
	if res, ok := res.(*admin.Resource); ok {
		Admin := res.GetAdmin()
		res.UseTheme("l10n")

		if res.Permission == nil {
			res.Permission = roles.NewPermission()
		}
		res.Permission.Allow(roles.CRUD, "locale_admin").Allow(roles.Read, "locale_reader")

		if res.GetMeta("Localization") == nil {
			res.Meta(&admin.Meta{Name: "Localization", Type: "localization", Valuer: func(value interface{}, ctx *core.Context) interface{} {
				var languageCodes []string
				var db = ctx.GetDB()
				var scope = db.NewScope(value)
				db.New().Set("l10n:mode", "unscoped").Model(res.Value).Where(fmt.Sprintf("%v = ?", scope.PrimaryKey()), scope.PrimaryKeyValue()).Pluck("DISTINCT language_code", &languageCodes)
				return languageCodes
			}})
		}

		var attrs = res.ConvertSectionToStrings(res.IndexAttrs())
		var hasLocalization bool
		for _, attr := range attrs {
			if attr == "Localization" {
				hasLocalization = true
				break
			}
		}

		if hasLocalization {
			res.IndexAttrs(res.IndexAttrs(), "-LanguageCode")
		} else {
			res.IndexAttrs(res.IndexAttrs(), "-LanguageCode", "Localization")
		}
		res.NewAttrs(res.NewAttrs(), "-LanguageCode", "-Localization")
		res.EditAttrs(res.EditAttrs(), "-LanguageCode", "-Localization")
		res.ShowAttrs(res.ShowAttrs(), "-LanguageCode", "-Localization", false)

		// Set meta permissions
		for _, field := range res.FakeScope.Fields() {
			if isSyncField(field.StructField) {
				if meta := res.GetMeta(field.Name); meta != nil {
					permission := meta.Meta.Permission
					if permission == nil {
						permission = roles.Allow(roles.CRUD, "global_admin").Allow(roles.Read, "locale_reader")
					} else {
						permission = permission.Allow(roles.CRUD, "global_admin").Allow(roles.Read, "locale_reader")
					}

					meta.SetPermission(permission)
				} else {
					res.Meta(&admin.Meta{Name: field.Name, Permission: roles.Allow(roles.CRUD, "global_admin").Allow(roles.Read, "locale_reader")})
				}
			}
		}

		// Roles
		role := res.Permission.Role
		if _, ok := role.Get("global_admin"); !ok {
			role.Register("global_admin", func(req *http.Request, currentUser interface{}) bool {
				if getLocaleFromContext(&core.Context{Request: req}) == Global {
					for _, locale := range getEditableLocales(req, currentUser) {
						if locale == Global {
							return true
						}
					}
				}
				return false
			})
		}

		if _, ok := role.Get("locale_admin"); !ok {
			role.Register("locale_admin", func(req *http.Request, currentUser interface{}) bool {
				currentLocale := getLocaleFromContext(&core.Context{Request: req})
				for _, locale := range getEditableLocales(req, currentUser) {
					if locale == currentLocale {
						return true
					}
				}
				return false
			})
		}

		if _, ok := role.Get("locale_reader"); !ok {
			role.Register("locale_reader", func(req *http.Request, currentUser interface{}) bool {
				currentLocale := getLocaleFromContext(&core.Context{Request: req})
				for _, locale := range getAvailableLocales(req, currentUser) {
					if locale == currentLocale {
						return true
					}
				}
				return false
			})
		}

		Admin.OnRouter(func(r xroute.Router) {
			if r.GetMiddleware("qor:l10n.set_locale") == nil {
				// Middleware
				r.Use(&xroute.Middleware{
					Name: "qor:l10n.set_locale",
					Handler: func(chain *xroute.ChainHandler) {
						context := admin.ContextFromChain(chain)
						db := context.GetDB().Set("l10n:locale", getLocaleFromContext(context.Context))
						if mode := context.Request.URL.Query().Get("locale_mode"); mode != "" {
							db = db.Set("l10n:mode", mode)
						}

						usingLanguageCodeAsPrimaryKey := false
						if res := context.Resource; res != nil {
							for idx, primaryField := range res.PrimaryFields {
								if primaryField.Name == "LanguageCode" {
									_, params := res.PrimaryQuery(context.URLParam(res.ParamIDName()))
									if len(params) > idx {
										usingLanguageCodeAsPrimaryKey = true
										db = db.Set("l10n:locale", params[idx])

										// PUT usually used for localize
										if context.Request.Method == "PUT" {
											if _, ok := db.Get("l10n:localize_to"); !ok {
												db = db.Set("l10n:localize_to", getLocaleFromContext(context.Context))
											}
										}
										break
									}
								}
							}
						}

						if !usingLanguageCodeAsPrimaryKey {
							for key, values := range context.Request.URL.Query() {
								if regexp.MustCompile(`primary_key\[.+_language_code\]`).MatchString(key) {
									if len(values) > 0 {
										db = db.Set("l10n:locale", values[0])

										// PUT usually used for localize
										if context.Request.Method == "PUT" || context.Request.Method == "POST" {
											db = db.Set(key, "")
											if _, ok := db.Get("l10n:localize_to"); !ok {
												db = db.Set("l10n:localize_to", getLocaleFromContext(context.Context))
											}
										}
									}
								}
							}
						}

						if context.Request.URL.Query().Get("sorting") != "" {
							db = db.Set("l10n:mode", "locale")
						}
						context.SetDB(db)
						chain.Pass()
					},
				})
			}
		})

		// FunMap
		Admin.RegisterFuncMap("current_locale", func(context admin.Context) string {
			return getLocaleFromContext(context.Context)
		})

		Admin.RegisterFuncMap("global_locale", func() string {
			return Global
		})

		Admin.RegisterFuncMap("viewable_locales", func(context admin.Context) []string {
			return getAvailableLocales(context.Request, context.CurrentUser())
		})

		Admin.RegisterFuncMap("editable_locales", func(context admin.Context) []string {
			return getEditableLocales(context.Request, context.CurrentUser())
		})

		Admin.RegisterFuncMap("createable_locales", func(context admin.Context) []string {
			editableLocales := getEditableLocales(context.Request, context.CurrentUser())
			if _, ok := context.Resource.Value.(localeCreatableInterface); ok {
				return editableLocales
			}

			for _, locale := range editableLocales {
				if locale == Global {
					return []string{Global}
				}
			}
			return []string{}
		})

		if res.GetAction("Localize") == nil {
			argumentResource := Admin.NewResource(&LocalizeActionArgument{})
			argumentResource.Meta(&admin.Meta{
				Name: "From",
				Type: "select_one",
				Valuer: func(_ interface{}, context *core.Context) interface{} {
					return Global
				},
				Collection: func(value interface{}, context *core.Context) (results [][]string) {
					for _, locale := range getAvailableLocales(context.Request, context.CurrentUser()) {
						results = append(results, []string{locale, locale})
					}
					return
				},
			})
			argumentResource.Meta(&admin.Meta{
				Name: "To",
				Type: "select_many",
				Valuer: func(_ interface{}, context *core.Context) interface{} {
					return []string{getLocaleFromContext(context)}
				},
				Collection: func(value interface{}, context *core.Context) (results [][]string) {
					for _, locale := range getEditableLocales(context.Request, context.CurrentUser()) {
						results = append(results, []string{locale, locale})
					}
					return
				},
			})

			res.Action(&admin.Action{
				Name: "Localize",
				Handler: func(argument *admin.ActionArgument) error {
					var (
						db        = argument.Context.GetDB()
						arg       = argument.Argument.(*LocalizeActionArgument)
						results   = res.NewSlice()
						sqls      []string
						sqlParams []interface{}
					)

					for _, primaryValue := range argument.PrimaryValues {
						primaryQuerySQL, primaryParams := res.PrimaryQuery(primaryValue)
						sqls = append(sqls, primaryQuerySQL)
						sqlParams = append(sqlParams, primaryParams...)
					}

					db.Set("l10n:locale", arg.From).Where(strings.Join(sqls, " OR "), sqlParams...).Find(results)

					reflectResults := reflect.Indirect(reflect.ValueOf(results))
					for i := 0; i < reflectResults.Len(); i++ {
						for _, to := range arg.To {
							if err := db.Set("l10n:localize_to", to).Unscoped().Save(reflectResults.Index(i).Interface()).Error; err != nil {
								return err
							}
						}
					}
					return nil
				},
				Modes:      []string{"index", "menu_item"},
				Permission: roles.Allow(roles.CRUD, roles.Anyone),
				Resource:   argumentResource,
			})
		}
	}
}
