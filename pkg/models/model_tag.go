package models

import (
	"strings"

	"github.com/thoas/go-funk"
)

type Tag struct {
	ID     uint    `gorm:"primary_key" json:"id"`
	Scenes []Scene `gorm:"many2many:scene_tags;" json:"scenes"`
	Name   string  `gorm:"index" json:"name"`
	Clean  string  `gorm:"index" json:"clean"`
	Count  int     `json:"count"`
}

func (t *Tag) Save() error {
	db, _ := GetDB()
	err := db.Save(t).Error
	db.Close()
	return err
}

func ConvertTag(t string) string {
	t = strings.ToLower(t)

	if funk.Contains([]string{"180", "60fps", "60 fps", "5k", "5k+", "big dick", "big cocks", "axaxqxrrysrwqua", "girl-boy", "virtual reality", "vr porn"}, t) {
		return ""
	}

	if funk.Contains([]string{"sixty-nine"}, t) {
		return "69"
	}

	if funk.Contains([]string{"anal"}, t) {
		return "anal sex"
	}

	if funk.Contains([]string{"butt plug"}, t) {
		return "anal toys"
	}

	if funk.Contains([]string{"cum in ass", "creampie - ass"}, t) {
		return "anal creampie"
	}

	if funk.Contains([]string{"athletic"}, t) {
		return "athletic body"
	}

	if funk.Contains([]string{"threesome bgg", "bgg", "girl-girl-boy"}, t) {
		return "threesome ffm"
	}

	if funk.Contains([]string{"threesome bbg", "bbg", "mmf"}, t) {
		return "threesome fmm"
	}

	if funk.Contains([]string{"big boobs"}, t) {
		return "big tits"
	}

	if funk.Contains([]string{"blow job"}, t) {
		return "blowjob"
	}

	if funk.Contains([]string{"boobs job", "titty fucking", "titjob"}, t) {
		return "titty fuck"
	}

	if funk.Contains([]string{"catsuite"}, t) {
		return "catsuit"
	}

	if funk.Contains([]string{"cum swapping"}, t) {
		return "cum swap"
	}

	if funk.Contains([]string{"cum shot", "cum-shot"}, t) {
		return "cumshot"
	}

	if funk.Contains([]string{"curvy woman"}, t) {
		return "curvy"
	}

	if funk.Contains([]string{"cowgirl reverse"}, t) {
		return "reverse cowgirl"
	}

	if funk.Contains([]string{"deepthroat", "deepthroating"}, t) {
		return "deep throat"
	}

	if funk.Contains([]string{"dominating"}, t) {
		return "dominant"
	}

	if funk.Contains([]string{"double penetration"}, t) {
		return "dp"
	}

	if funk.Contains([]string{"doggy", "doggy style"}, t) {
		return "doggystyle"
	}

	if funk.Contains([]string{"face cumshot", "facial cumshot", "facial", "face cumshot"}, t) {
		return "cum on face"
	}

	if funk.Contains([]string{"girlfrien"}, t) {
		return "girlfriend"
	}

	if funk.Contains([]string{"hand job"}, t) {
		return "handjob"
	}

	if funk.Contains([]string{"latin", "latin babe"}, t) {
		return "latina"
	}

	if funk.Contains([]string{"lesbian love", "lesbians"}, t) {
		return "lesbian"
	}

	if funk.Contains([]string{"milfs", "cougar", "mother", "mom", "british mom"}, t) {
		return "milf"
	}

	if funk.Contains([]string{"european"}, t) {
		return "euro"
	}

	if funk.Contains([]string{"red head"}, t) {
		return "redhead"
	}

	if funk.Contains([]string{"role playing"}, t) {
		return "role play"
	}

	if funk.Contains([]string{"shaved"}, t) {
		return "shaved pussy"
	}

	if funk.Contains([]string{"squirt"}, t) {
		return "squirting"
	}

	if funk.Contains([]string{"teens"}, t) {
		return "teen"
	}

	if funk.Contains([]string{"trimmed"}, t) {
		return "trimmed pussy"
	}

	if funk.Contains([]string{"voayer"}, t) {
		return "voyeur"
	}

	if funk.Contains([]string{"small boobs", "small natural tits"}, t) {
		return "small tits"
	}

	if funk.Contains([]string{"natural boobs"}, t) {
		return "natural tits"
	}

	if funk.Contains([]string{"medium boobs"}, t) {
		return "medium tits"
	}

	if funk.Contains([]string{"shaved"}, t) {
		return "shaved pussy"
	}

	if funk.Contains([]string{"pussy eating"}, t) {
		return "pussy licking"
	}

	if funk.Contains([]string{"pussy cumshot"}, t) {
		return "cum on pussy"
	}

	if funk.Contains([]string{"tits cumshoot"}, t) {
		return "cum on tits"
	}

	if funk.Contains([]string{"hairy", "hairy bush"}, t) {
		return "hairy pussy"
	}

	if funk.Contains([]string{"no tattoo"}, t) {
		return "no tattoos"
	}

	if funk.Contains([]string{"tattoo", "tatoos"}, t) {
		return "tattoos"
	}

	if funk.Contains([]string{"piercing", "pirced pussy"}, t) {
		return "piercings"
	}

	if funk.Contains([]string{"russian girl"}, t) {
		return "russian"
	}

	if funk.Contains([]string{"spanish girl"}, t) {
		return "spanish"
	}

	if funk.Contains([]string{"stepbro"}, t) {
		return "step brother"
	}

	if funk.Contains([]string{"stepsis"}, t) {
		return "step sister"
	}

	if funk.Contains([]string{"toys", "vibrator"}, t) {
		return "sex toys"
	}

	if funk.Contains([]string{"ass cumshot"}, t) {
		return "cum on ass"
	}

	if funk.Contains([]string{"busty"}, t) {
		return "big tits"
	}

	if funk.Contains([]string{"mature mother"}, t) {
		return "mature"
	}

	if funk.Contains([]string{"latin step sister"}, t) {
		return "latina"
	}

	if funk.Contains([]string{"group"}, t) {
		return "group sex"
	}

	if funk.Contains([]string{"lesbian mom"}, t) {
		return "lesbian"
	}

	if funk.Contains([]string{"twin sisters"}, t) {
		return "twins"
	}

	if funk.Contains([]string{"threesomes"}, t) {
		return "threesome"
	}

	if funk.Contains([]string{"feet cumshot"}, t) {
		return "cum on feet"
	}

	if funk.Contains([]string{"black female"}, t) {
		return "black"
	}

	if funk.Contains([]string{"double penetration"}, t) {
		return "dp"
	}

	return t
}
