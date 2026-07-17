package tasks

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func ResetAutoTags() {
	db, _ := models.GetDB()
	defer db.Close()

	var tags []models.Tag
	db.Where("is_system = ?", true).Find(&tags)

	for _, tag := range tags {
		db.Exec("DELETE FROM scene_tags WHERE tag_id = ?", tag.ID)
		db.Delete(&tag)
	}

	var t models.Tag
	t.CountTags()
}

func GenerateAutoTags() {
	cfg := config.Config.AutoTag
	if !cfg.BreastType && !cfg.Age && !cfg.Height && !cfg.Nationality &&
		!cfg.Ethnicity && !cfg.HairColor && !cfg.EyeColor && !cfg.CupSize &&
		!cfg.Resolution && !cfg.VideoFormat && !cfg.Duration && !cfg.Interracial {
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	// Build gender filter set (empty = include all genders)
	genderFilter := make(map[string]bool)
	for _, g := range cfg.GenderFilter {
		genderFilter[strings.ToLower(strings.TrimSpace(g))] = true
	}
	filterByGender := len(genderFilter) > 0

	// Pre-cache existing system tags: lowercase name -> ID
	tagCache := make(map[string]uint)
	var existingTags []models.Tag
	db.Where("is_system = ?", true).Find(&existingTags)
	for _, t := range existingTags {
		tagCache[strings.ToLower(t.Name)] = t.ID
	}

	heightShortMax := cfg.HeightShortMax
	heightAvgMax := cfg.HeightAverageMax
	durShortMax := cfg.DurationShortMax
	durStdMax := cfg.DurationStandardMax

	const batchSize = 500
	offset := 0
	for {
		var scenes []models.Scene
		db.Model(&models.Scene{}).Preload("Cast").Preload("Tags").Preload("Files").
			Limit(batchSize).Offset(offset).Find(&scenes)
		if len(scenes) == 0 {
			break
		}

		for _, scene := range scenes {
			// Snapshot existing scene tags for duplicate detection
			existingSceneTags := make(map[string]bool)
			for _, t := range scene.Tags {
				existingSceneTags[strings.ToLower(t.Name)] = true
			}

			// Filter cast by gender if configured
			filteredCast := scene.Cast
			if filterByGender {
				filteredCast = nil
				for _, actor := range scene.Cast {
					if genderFilter[strings.ToLower(strings.TrimSpace(actor.Gender))] {
						filteredCast = append(filteredCast, actor)
					}
				}
			}

			addTag := func(name string) {
				addTagToScene(db, &scene, name, existingSceneTags, tagCache)
			}

			// Breast Type
			if cfg.BreastType {
				isNatural, isFake := false, false
				for _, actor := range filteredCast {
					if strings.EqualFold(actor.BreastType, "Natural") {
						isNatural = true
					}
					if strings.EqualFold(actor.BreastType, "Fake") ||
						strings.EqualFold(actor.BreastType, "Silicone") ||
						strings.EqualFold(actor.BreastType, "Enhanced") {
						isFake = true
					}
				}
				if isNatural {
					addTag("Breast Type - Natural")
				}
				if isFake {
					addTag("Breast Type - Fake")
				}
			}

			// Age
			if cfg.Age && !scene.ReleaseDate.IsZero() {
				for _, actor := range filteredCast {
					if !actor.BirthDate.IsZero() {
						age := scene.ReleaseDate.Year() - actor.BirthDate.Year()
						if scene.ReleaseDate.YearDay() < actor.BirthDate.YearDay() {
							age--
						}
						if age >= 18 && age < 100 {
							addTag(fmt.Sprintf("Age: %d", age))
						}
					}
				}
			}

			// Height
			if cfg.Height {
				for _, actor := range filteredCast {
					if actor.Height > 0 {
						if actor.Height <= heightShortMax {
							addTag("Height: Short")
						} else if actor.Height <= heightAvgMax {
							addTag("Height: Average")
						} else {
							addTag("Height: Tall")
						}
					}
				}
			}

			// Nationality
			if cfg.Nationality {
				for _, actor := range filteredCast {
					if actor.Nationality != "" {
						addTag("Nationality: " + actor.Nationality)
					}
				}
			}

			// Ethnicity
			if cfg.Ethnicity {
				for _, actor := range filteredCast {
					if actor.Ethnicity != "" {
						addTag("Ethnicity: " + actor.Ethnicity)
					}
				}
			}

			// Hair Color
			if cfg.HairColor {
				for _, actor := range filteredCast {
					if actor.HairColor != "" {
						addTag("Hair: " + actor.HairColor)
					}
				}
			}

			// Eye Color
			if cfg.EyeColor {
				for _, actor := range filteredCast {
					if actor.EyeColor != "" {
						addTag("Eyes: " + actor.EyeColor)
					}
				}
			}

			// Cup Size
			if cfg.CupSize {
				for _, actor := range filteredCast {
					if actor.CupSize != "" {
						addTag("Cup: " + actor.CupSize)
					}
				}
			}

			// Interracial — only among gender-filtered cast
			if cfg.Interracial && len(filteredCast) > 0 {
				ethnicities := make(map[string]bool)
				for _, actor := range filteredCast {
					if actor.Ethnicity != "" {
						ethnicities[strings.ToLower(actor.Ethnicity)] = true
					}
				}
				if len(ethnicities) > 1 {
					addTag("Interracial")
				}
			}

			// Duration
			if cfg.Duration && scene.Duration > 0 {
				if scene.Duration <= durShortMax {
					addTag("Duration: Short")
				} else if scene.Duration <= durStdMax {
					addTag("Duration: Standard")
				} else {
					addTag("Duration: Long")
				}
			}

			// Resolution & Format
			if (cfg.Resolution || cfg.VideoFormat) && len(scene.Files) > 0 {
				var bestFile models.File
				for _, f := range scene.Files {
					if f.VideoHeight > bestFile.VideoHeight {
						bestFile = f
					}
				}

				if cfg.Resolution && bestFile.VideoHeight > 0 {
					switch {
					case bestFile.VideoHeight >= 4320:
						addTag("Res: 8K")
					case bestFile.VideoHeight >= 2880:
						addTag("Res: 6K")
					case bestFile.VideoHeight >= 2700:
						addTag("Res: 5K")
					case bestFile.VideoHeight >= 1900:
						addTag("Res: 4K")
					case bestFile.VideoHeight >= 1440:
						addTag("Res: 1440p")
					case bestFile.VideoHeight >= 1080:
						addTag("Res: 1080p")
					case bestFile.VideoHeight >= 720:
						addTag("Res: 720p")
					default:
						addTag("Res: SD")
					}
				}

				if cfg.VideoFormat {
					switch bestFile.VideoProjection {
					case "180_sbs", "180_tb":
						addTag("Format: 180°")
					case "360_sbs", "360_tb", "360_mono":
						addTag("Format: 360°")
					case "flat":
						addTag("Format: Flat")
					}
				}
			}
		}
		offset += batchSize
	}

	var t models.Tag
	t.CountTags()
}

func addTagToScene(db *gorm.DB, scene *models.Scene, tagName string, existingSceneTags map[string]bool, tagCache map[string]uint) {
	lowerName := strings.ToLower(tagName)
	if existingSceneTags[lowerName] {
		return
	}
	existingSceneTags[lowerName] = true

	var tag models.Tag
	if id, ok := tagCache[lowerName]; ok {
		tag.ID = id
		tag.Name = tagName
	} else {
		db.Where(models.Tag{Name: tagName}).FirstOrCreate(&tag)
		if !tag.IsSystem {
			db.Model(&tag).Update("is_system", true)
			tag.IsSystem = true
		}
		tagCache[lowerName] = tag.ID
	}

	db.Model(scene).Association("Tags").Append(tag)
}
