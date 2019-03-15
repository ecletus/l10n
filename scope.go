package l10n

import (
	"reflect"

	"github.com/moisespsena-go/aorm"
	"github.com/ecletus/core/utils"
)

// IsLocalizable return model is localizable or not
func IsLocalizable(scope *aorm.Scope) (IsLocalizable bool) {
	if scope.GetModelStruct().ModelType == nil {
		return false
	}
	_, IsLocalizable = reflect.New(scope.GetModelStruct().ModelType).Interface().(l10nInterface)
	return
}

type localeCreatableInterface interface {
	CreatableFromLocale()
}

type localeCreatableInterface2 interface {
	LocaleCreatable()
}

func isLocaleCreatable(scope *aorm.Scope) (ok bool) {
	if _, ok = reflect.New(scope.GetModelStruct().ModelType).Interface().(localeCreatableInterface); ok {
		return
	}
	_, ok = reflect.New(scope.GetModelStruct().ModelType).Interface().(localeCreatableInterface2)
	return
}

func setLocale(scope *aorm.Scope, locale string) {
	for _, field := range scope.Fields() {
		if field.Name == "LanguageCode" {
			field.Set(locale)
		}
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
	for _, field := range scope.GetModelStruct().StructFields {
		if isSyncField(field) {
			columns = append(columns, field.DBName)
		}
	}
	return
}
