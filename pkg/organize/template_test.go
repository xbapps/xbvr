package organize

import (
	"testing"
	"time"
)

func testValues() templateValues {
	return templateValues{
		Studio: "BaDoinkVR",
		Date:   time.Date(2018, 1, 22, 0, 0, 0, 0, time.UTC),
		Cast:   "Adriana.Chechik",
		Title:  "Grangerous.Liaisons",
		FOV:    "VR180",
		Height: 1920,
	}
}

func TestRenderDefaultTemplate(t *testing.T) {
	rel, dir, ok := renderTemplate("", testValues())
	if !ok {
		t.Fatal("default template rendered nothing")
	}
	wantRel := "BaDoinkVR/BaDoinkVR.18.01.22.Adriana.Chechik.Grangerous.Liaisons.XXX.VR180.1920p"
	if rel != wantRel {
		t.Errorf("rel = %q, want %q", rel, wantRel)
	}
	if wantDir := "BaDoinkVR.18.01.22.Adriana.Chechik.Grangerous.Liaisons.XXX.VR180.1920p"; dir != wantDir {
		t.Errorf("dir = %q, want %q", dir, wantDir)
	}
}

func TestRenderTemplateWithoutStudioPrefix(t *testing.T) {
	rel, dir, ok := renderTemplate("{studio}/{date}.{title}.{height}p", testValues())
	if !ok {
		t.Fatal("template rendered nothing")
	}
	if want := "BaDoinkVR/18.01.22.Grangerous.Liaisons.1920p"; rel != want {
		t.Errorf("rel = %q, want %q", rel, want)
	}
	if want := "18.01.22.Grangerous.Liaisons.1920p"; dir != want {
		t.Errorf("dir = %q, want %q", dir, want)
	}
}

func TestRenderTemplateEmptyCast(t *testing.T) {
	v := testValues()
	v.Cast = ""
	rel, _, ok := renderTemplate("{studio}/{studio}.{date}.{cast}.{title}.{fov}.{height}p", v)
	if !ok {
		t.Fatal("template rendered nothing")
	}
	if want := "BaDoinkVR/BaDoinkVR.18.01.22.Grangerous.Liaisons.VR180.1920p"; rel != want {
		t.Errorf("rel = %q, want %q", rel, want)
	}
}

func TestRenderTemplateCaseInsensitiveTokens(t *testing.T) {
	rel, _, _ := renderTemplate("{Studio}/{Title}", testValues())
	if want := "BaDoinkVR/Grangerous.Liaisons"; rel != want {
		t.Errorf("rel = %q, want %q", rel, want)
	}
}

func TestRenderTemplateFlat(t *testing.T) {
	rel, dir, ok := renderTemplate("{title}", testValues())
	if !ok {
		t.Fatal("template rendered nothing")
	}
	if rel != "Grangerous.Liaisons" || dir != rel {
		t.Errorf("rel = %q, dir = %q", rel, dir)
	}
}

func TestValidateTemplate(t *testing.T) {
	valid := []string{
		"",
		"{studio}/{title}",
		"{studio}/{studio}.{date}.{cast}.{title}.XXX.{fov}.{height}p",
		"{year}/{mm}/{title}",
		"custom-folder/{title}",
	}
	for _, tmpl := range valid {
		if err := ValidateTemplate(tmpl); err != nil {
			t.Errorf("ValidateTemplate(%q) = %v, want nil", tmpl, err)
		}
	}

	invalid := []string{
		"/absolute/{title}",
		"{studio}//{title}",
		"../{title}",
		"{studio}/{title}/{titel}",
		"{studio}/{}",
	}
	for _, tmpl := range invalid {
		if err := ValidateTemplate(tmpl); err == nil {
			t.Errorf("ValidateTemplate(%q) = nil, want error", tmpl)
		}
	}
}
