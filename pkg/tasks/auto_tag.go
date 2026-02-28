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

	// Find all system tags
	var tags []models.Tag
	db.Where("is_system = ?", true).Find(&tags)

	for _, tag := range tags {
		// Remove tag from all scenes
		db.Exec("DELETE FROM scene_tags WHERE tag_id = ?", tag.ID)

		// Delete the tag itself (optional, but cleaner)
		db.Delete(&tag)
	}

	// Recalculate counts
	var t models.Tag
	t.CountTags()
}

func GenerateAutoTags() {
	if !config.Config.AutoTag.BreastType {
		return
	}

	db, _ := models.GetDB()
	defer db.Close()

	var scenes []models.Scene
	db.Model(&models.Scene{}).Preload("Cast").Preload("Tags").Preload("Files").Find(&scenes)

	// Calculate ranges for height and duration once
	heightShortMax := config.Config.AutoTag.HeightShortMax
	heightAvgMax := config.Config.AutoTag.HeightAverageMax
	durShortMax := config.Config.AutoTag.DurationShortMax
	durStdMax := config.Config.AutoTag.DurationStandardMax

	for _, scene := range scenes {
		// Breast Type
		if config.Config.AutoTag.BreastType {
			isNatural := false
			isFake := false

			for _, actor := range scene.Cast {
				if strings.EqualFold(actor.BreastType, "Natural") {
					isNatural = true
				}
				if strings.EqualFold(actor.BreastType, "Fake") || strings.EqualFold(actor.BreastType, "Silicone") || strings.EqualFold(actor.BreastType, "Enhanced") {
					isFake = true
				}
			}

			if isNatural {
				addTagToScene(db, &scene, "Breast Type - Natural")
			}
			if isFake {
				addTagToScene(db, &scene, "Breast Type - Fake")
			}
		}

		// Age
		if config.Config.AutoTag.Age && !scene.ReleaseDate.IsZero() {
			for _, actor := range scene.Cast {
				if !actor.BirthDate.IsZero() {
					age := scene.ReleaseDate.Year() - actor.BirthDate.Year()
					// Adjust for month/day if needed, but year diff is usually sufficient for "Age: XX" tags
					if scene.ReleaseDate.YearDay() < actor.BirthDate.YearDay() {
						age--
					}
					if age >= 18 && age < 100 { // Basic sanity check
						addTagToScene(db, &scene, fmt.Sprintf("Age: %d", age))
					}
				}
			}
		}

		// Height
		if config.Config.AutoTag.Height {
			for _, actor := range scene.Cast {
				if actor.Height > 0 {
					if actor.Height <= heightShortMax {
						addTagToScene(db, &scene, "Height: Short")
					} else if actor.Height <= heightAvgMax {
						addTagToScene(db, &scene, "Height: Average")
					} else {
						addTagToScene(db, &scene, "Height: Tall")
					}
				}
			}
		}

		// Nationality
		if config.Config.AutoTag.Nationality {
			for _, actor := range scene.Cast {
				if actor.Nationality != "" {
					addTagToScene(db, &scene, "Nationality: "+actor.Nationality)
				}
			}
		}

		// Ethnicity
		if config.Config.AutoTag.Ethnicity {
			for _, actor := range scene.Cast {
				if actor.Ethnicity != "" {
					addTagToScene(db, &scene, "Ethnicity: "+actor.Ethnicity)
				}
			}
		}

		// Hair Color
		if config.Config.AutoTag.HairColor {
			for _, actor := range scene.Cast {
				if actor.HairColor != "" {
					addTagToScene(db, &scene, "Hair: "+actor.HairColor)
				}
			}
		}

		// Eye Color
		if config.Config.AutoTag.EyeColor {
			for _, actor := range scene.Cast {
				if actor.EyeColor != "" {
					addTagToScene(db, &scene, "Eyes: "+actor.EyeColor)
				}
			}
		}

		// Cup Size
		if config.Config.AutoTag.CupSize {
			for _, actor := range scene.Cast {
				if actor.CupSize != "" {
					addTagToScene(db, &scene, "Cup: "+actor.CupSize)
				}
			}
		}

		// Interracial
		if config.Config.AutoTag.Interracial && len(scene.Cast) > 0 {
			ethnicities := make(map[string]bool)

			for _, actor := range scene.Cast {
				if actor.Ethnicity != "" {
					ethnicities[strings.ToLower(actor.Ethnicity)] = true
				}
			}

			if len(ethnicities) > 1 {
				addTagToScene(db, &scene, "Interracial")
			}
		}

		// Duration
		if config.Config.AutoTag.Duration && scene.Duration > 0 {
			// Duration is in minutes
			if scene.Duration <= durShortMax {
				addTagToScene(db, &scene, "Duration: Short")
			} else if scene.Duration <= durStdMax {
				addTagToScene(db, &scene, "Duration: Standard")
			} else {
				addTagToScene(db, &scene, "Duration: Long")
			}
		}

		// Resolution & Format
		if (config.Config.AutoTag.Resolution || config.Config.AutoTag.VideoFormat) && len(scene.Files) > 0 {
			// Find best file (highest resolution)
			var bestFile models.File
			for _, f := range scene.Files {
				if f.VideoHeight > bestFile.VideoHeight {
					bestFile = f
				}
			}

			if config.Config.AutoTag.Resolution && bestFile.VideoHeight > 0 {
				if bestFile.VideoHeight >= 4320 {
					addTagToScene(db, &scene, "Res: 8K")
				} else if bestFile.VideoHeight >= 2880 {
					addTagToScene(db, &scene, "Res: 6K")
				} else if bestFile.VideoHeight >= 2160 {
					addTagToScene(db, &scene, "Res: 5K") // Sometimes 5K is distinct
				} else if bestFile.VideoHeight >= 1900 { // Allow some tolerance for 4K
					addTagToScene(db, &scene, "Res: 4K")
				} else if bestFile.VideoHeight >= 1440 {
					addTagToScene(db, &scene, "Res: 1440p")
				} else if bestFile.VideoHeight >= 1080 {
					addTagToScene(db, &scene, "Res: 1080p")
				} else if bestFile.VideoHeight >= 720 {
					addTagToScene(db, &scene, "Res: 720p")
				} else {
					addTagToScene(db, &scene, "Res: SD")
				}
			}

			if config.Config.AutoTag.VideoFormat {
				if bestFile.VideoProjection == "180_sbs" || bestFile.VideoProjection == "180_tb" {
					addTagToScene(db, &scene, "Format: 180°")
				} else if bestFile.VideoProjection == "360_sbs" || bestFile.VideoProjection == "360_tb" || bestFile.VideoProjection == "360_mono" {
					addTagToScene(db, &scene, "Format: 360°")
				} else if bestFile.VideoProjection == "flat" {
					addTagToScene(db, &scene, "Format: Flat")
				}
			}
		}

		// Helper function inside loop to save code duplication
	}

	// Recalculate tag counts
	var t models.Tag
	t.CountTags()
}

func addTagToScene(db *gorm.DB, scene *models.Scene, tagName string) {
	tagExists := false
	for _, t := range scene.Tags {
		if strings.EqualFold(t.Name, tagName) {
			tagExists = true
			break
		}
	}

	if !tagExists {
		var tag models.Tag
		db.Where(models.Tag{Name: tagName}).FirstOrCreate(&tag)

		// Mark as system tag if it's new or update it
		if !tag.IsSystem {
			tag.IsSystem = true
			tag.Save()
		}

		db.Model(scene).Association("Tags").Append(tag)
	}
}
