package util

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
)

type Net struct {
	Filename string
	Name     string `json:"name,omitempty"`
	Type     string `json:"type,omitempty"`
	IPAlloc  struct {
		Type   string `json:"type,omitempty"`
		Subnet string `json:"subnet,omitempty"`
	}
}

func LoadNet(path string, n interface{}) error {
	c, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(c, n); err != nil {
		return err
	}

	// populate n.Filename if exists
	v := reflect.ValueOf(n)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		if v.Kind() == reflect.Struct {
			if fn := v.FieldByName("Filename"); fn.IsValid() {
				if fn.Type().Kind() == reflect.String && fn.CanSet() {
					fn.SetString(path)
				}
			}
		}
	}

	return nil
}
