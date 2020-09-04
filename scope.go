package l10n

import (
	"reflect"

	"github.com/ecletus/core/utils"
	"github.com/moisespsena-go/aorm"
)

// IsLocalizable return model is localizable or not
func IsLocalizable(scope *aorm.Scope) (IsLocalizable bool) {
	if scope.Struct().Type == nil {
		return false
	}
	_, IsLocalizable = reflect.New(scope.Struct().Type).Interface().(l10nInterface)
	return
}

type localeCreatableInterface interface {
	CreatableFromLocale()
}

type localeCreatableInterface2 interface {
	LocaleCreatable()
}

func isLocaleCreatable(scope *aorm.Scope) (ok bool) {
	if _, ok = reflect.New(scope.Struct().Type).Interface().(localeCreatableInterface); ok {
		return
	}
	_, ok = reflect.New(scope.Struct().Type).Interface().(localeCreatableInterface2)
	return
}

func setLocale(scope *aorm.Scope, locale string) {
	if field, ok := scope.Instance().FieldsMap["LanguageCode"]; ok {
		field.Set(locale)
	}
}

func getQueryLocale(scope *aorm.Scope) (locale string, isLocale bool) {
	if str, ok := scope.DB().Get("l10n:locale"); ok {
		if locale, ok := str.(string); ok && locale != "" {
			return locale, locale != Global
		}
	}
	return Global, false
}

func getLocale(scope *aorm.Scope) (locale string, isLocale bool) {
	if str, ok := scope.DB().Get("l10n:localize_to"); ok {
		if locale, ok := str.(string); ok && locale != "" {
			return locale, locale != Global
		}
	}

	return getQueryLocale(scope)
}

func isSyncField(field *aorm.StructField) bool {
	if _, ok := utils.ParseTagOption(field.Tag.Get("l10n"))["SYNC"]; ok {
		return true
	}
	return false
}

func syncColumns(scope *aorm.Scope) (columns []string) {
	for _, field := range scope.Struct().Fields {
		if isSyncField(field) {
			columns = append(columns, field.DBName)
		}
	}
	return
}
