package models

// Country Code-Name mapping is saved in the kvs table
import (
	"encoding/json"
	"net/http"
	"sort"
)

type CountryDetails struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	FlagUrl string `json:"flag_url"`
}

type countryJson struct {
	Flags struct {
		PNG string `json:"png"`
		SVG string `json:"svg"`
		Alt string `json:"alt"`
	} `json:"flags"`
	Name struct {
		Common     string `json:"common"`
		Official   string `json:"official"`
		NativeName struct {
			Dan struct {
				Official string `json:"official"`
				Common   string `json:"common"`
			} `json:"dan"`
			Fao struct {
				Official string `json:"official"`
				Common   string `json:"common"`
			} `json:"fao"`
		} `json:"nativeName"`
	} `json:"name"`
	Cca2 string `json:"cca2"`
}

func GetCountryList() []CountryDetails {
	db, _ := GetDB()
	defer db.Close()

	out := []CountryDetails{}
	var kvs KV

	kvs.Key = "countries"
	db.Find(&kvs)
	if kvs.Value == "" {
		// if the kvs doesn't exist get a new country list
		resp, err := http.Get("https://restcountries.com/v3.1/all?fields=name,flags,cca2")
		if err != nil {
			return out
		}
		defer resp.Body.Close()

		var countries []countryJson
		err = json.NewDecoder(resp.Body).Decode(&countries)
		if err != nil || len(countries) == 0 {
			return out
		}
		for _, country := range countries {
			mapurl := country.Flags.SVG
			if mapurl == "" {
				mapurl = country.Flags.PNG
			}
			out = append(out, CountryDetails{Code: country.Cca2, Name: country.Name.Common, FlagUrl: mapurl})
		}
		jsonString, _ := json.Marshal(out)
		kvs = KV{Key: "countries", Value: string(jsonString)}
		kvs.Save()
	}
	json.Unmarshal([]byte(kvs.Value), &out)

	// Define a comparison function for sorting by the Name field
	byName := func(i, j int) bool {
		return out[i].Name < out[j].Name
	}
	sort.Slice(out, byName)

	return out
}
