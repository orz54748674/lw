package common

import (
	"github.com/qor/i18n"
	"github.com/qor/i18n/backends/yaml"
	"path/filepath"
	"sync"
	"vn/common/utils"
)

//var CurLanguage = "zh_CN"
var CurLanguage = "VN"
var i18 *i18n.I18n
var once18 sync.Once

func getI18n() *i18n.I18n {
	if i18 != nil {
		return i18
	}
	once18.Do(func() {
		if i18 != nil {
			return
		}
		path := utils.GetProjectAbsPath()
		i18 = i18n.New(
			yaml.New(filepath.Join(path, "bin/locales/VN.yml")), // load translations from the YAML files in directory `config/locales`
			//yaml.New("E:\\space\\golang\\vn_server\\bin\\locales\\cn-ZH.yml"), // load translations from the YAML files in directory `config/locales`
			yaml.New(filepath.Join(path, "bin/locales/zh_CN.yml")), // load translations from the YAML files in directory `config/locales`
		)
	})
	return i18
}

func I18str(key string, args ...interface{}) string {
	return string(getI18n().T(CurLanguage, key, args))
}
