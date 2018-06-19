package config

import (
	"encoding/json"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-hsm/constants"
	"github.com/pritunl/pritunl-hsm/errortypes"
	"io/ioutil"
)

var (
	Config            = &ConfigData{}
	StaticRoot        = ""
	StaticTestingRoot = ""
)

type ConfigData struct {
	path            string   `json:"-"`
	loaded          bool     `json:"-"`
	PritunlZeroHosts []string `json:"pritunl_zero_hosts"`
}

func (c *ConfigData) Save() (err error) {
	if !c.loaded {
		err = &errortypes.WriteError{
			errors.New("config: Config file has not been loaded"),
		}
		return
	}

	data, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "config: File marshal error"),
		}
		return
	}

	err = ioutil.WriteFile(constants.ConfPath, data, 0600)
	if err != nil {
		err = &errortypes.WriteError{
			errors.Wrap(err, "config: File write error"),
		}
		return
	}

	return
}

func Load() (err error) {
	data := &ConfigData{}

	file, err := ioutil.ReadFile(constants.ConfPath)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "config: File read error"),
		}
		return
	}

	err = json.Unmarshal(file, data)
	if err != nil {
		err = &errortypes.ReadError{
			errors.Wrap(err, "config: File unmarshal error"),
		}
		return
	}

	data.loaded = true

	Config = data

	return
}

func Save() (err error) {
	err = Config.Save()
	if err != nil {
		return
	}

	return
}

func Init() (err error) {
	err = Load()
	if err != nil {
		return
	}

	return
}
