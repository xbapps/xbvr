package models

import "encoding/json"

type Aliases struct {
	Aliases []string
}

type ModelAliases []Aliases

func (m *ModelAliases) GetAliases() (modelAliases []Aliases) {
	return *m
}

func (m *ModelAliases) ConvertName(searchedName string) string {
	for _, v := range m.GetAliases() {
		if v.ContainsName(searchedName) {
			return v.ConvertName(searchedName)
		}
	}
	return searchedName
}

func (a *Aliases) UnmarshalJSON(data []byte) error {
	var s []string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}
	a.Aliases = s
	return nil
}

func (a Aliases) First() string {
	return a.Aliases[0]
}

func (a Aliases) ContainsName(name string) bool {
	for _, v := range a.Aliases {
		if name == v {
			return true
		}
	}
	return false
}

func (a Aliases) ConvertName(name string) string {
	if a.ContainsName(name) {
		log.Debugf("Model alias found for: %v (%v)", name, a.First())
		return a.First()
	}
	return name
}
