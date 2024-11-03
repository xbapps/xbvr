package models

import (
	"time"
)

type IdName struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type StashStudio struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Parent  IdName    `json:"parent"`
	Updated time.Time `json:"updated"`
}
type StashPerformerStudio struct {
	SceneCount int         `json:"scene_count"`
	Studio     StashStudio `json:"studio"`
}

type StashPerformer struct {
	ID              string                  `json:"id"`
	Name            string                  `json:"name"`
	Disambiguation  string                  `json:"disambiguation"`
	Aliases         []string                `json:"aliases"`
	Gender          string                  `json:"gender"`
	URLs            []StashURL              `json:"urls"`
	BirthDate       string                  `json:"birth_date"`
	Country         string                  `json:"country"`
	Ethnicity       string                  `json:"ethnicity"`
	Height          int                     `json:"height"`
	Weight          int                     `json:"weight"`
	EyeColor        string                  `json:"eye_color"`
	HairColor       string                  `json:"hair_color"`
	CupSize         string                  `json:"cup_size"`
	BandSize        int                     `json:"band_size"`
	WaistSize       int                     `json:"waist_size"`
	HipSize         int                     `json:"hip_size"`
	BreastType      string                  `json:"breast_type"`
	CareerStartYear int                     `json:"career_start_year"`
	CareerEndYear   int                     `json:"career_end_year"`
	Tattoos         []StashBodyModification `json:"tattoos"`
	Piercings       []StashBodyModification `json:"piercings"`
	Images          []Image                 `json:"images"`
	Deleted         bool                    `json:"deleted"`
	MergedIds       []string                `json:"merged_ids"`
	Created         string                  `json:"created"`
	Updated         time.Time               `json:"updated"`
	Studios         []StashPerformerStudio  `json:"studios"`
}

type StashBodyModification struct {
	Description string `json:"description"`
	Location    string `json:"location"`
}

type StashURL struct {
	URL  string `json:"url"`
	Type string `json:"type"`
	Site Site   `json:"site"`
}

type StashScene struct {
	ID         string             `json:"id"`
	Title      string             `json:"title"`
	Details    string             `json:"details"`
	Date       string             `json:"date"`
	Updated    time.Time          `json:"updated"`
	URLs       []StashURL         `json:"urls"`
	Performers []StashPerformerAs `json:"performers"`
	Studio     StashStudio        `json:"studio"`
	Duration   int                `json:"duration"`
	Code       string             `json:"code"`
	Images     []Image            `json:"images"`
}

type StashPerformerAs struct {
	Performer StashPerformer ``
	As        string         ``
}

type DELETEStashPerformerMin struct {
	ID      string `json:"id"`
	Updated string `json:"updated"`
	Gender  string `json:"gender"`
	Name    string `json:"name"`
}
type StashImage struct {
	Url    StashPerformer `json:"url"`
	Width  int            `json:"width"`
	Height int            `json:"height"`
}
