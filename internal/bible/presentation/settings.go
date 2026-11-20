package presentation

import (
	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// OpenSettings opens the Settings modal: the size and the typeface of the text, the high contrast
// mode, reading aloud, the chapter previews and the language of the app.
func (s *Screen) OpenSettings() {
	p := s.prefs

	preview := widget.NewLabel(s.t("bible.settings.preview"))
	preview.ID = "settings-preview"
	preview.Style.FontSize = float64(p.FontSize)
	sizeLabel := widget.NewLabel(s.t("bible.settings.fontsize", p.FontSize))
	sizeLabel.ID = "settings-size-label"
	size := widget.NewSlider(domain.MinFontSize, domain.MaxFontSize, domain.FontSizeStep, float64(p.FontSize))
	size.ID = "settings-size"
	size.OnInput = func(*widget.Event) {
		n := int(size.Value)
		sizeLabel.Text = s.t("bible.settings.fontsize", n)
		preview.Style.FontSize = float64(n)
		s.Doc.Invalidate()
	}

	font := widget.NewComboBox(s.t("bible.settings.font"),
		widget.Option{Label: s.t("bible.settings.font.go"), Value: string(gfx.FamilyGo)},
		widget.Option{Label: s.t("bible.settings.font.medium"), Value: string(gfx.FamilyMedium)},
		widget.Option{Label: s.t("bible.settings.font.mono"), Value: string(gfx.FamilyMono)})
	font.ID = "settings-font"
	font.SetValue(p.Font)

	invert := widget.NewSwitch(s.t("bible.settings.invert"), p.Invert)
	invert.ID = "settings-invert"
	speech := widget.NewSwitch(s.t("bible.settings.tts"), p.Speech)
	speech.ID = "settings-tts"
	titles := widget.NewSwitch(s.t("bible.settings.titles"), p.ChapterTitles)
	titles.ID = "settings-titles"

	lang := widget.NewComboBox(s.t("bible.settings.lang"),
		widget.Option{Label: s.t("bible.lang.pt-BR"), Value: "pt-BR"},
		widget.Option{Label: s.t("bible.lang.en"), Value: "en"})
	lang.ID = "settings-lang"
	lang.SetValue(s.tr.Lang())

	m := widget.NewModal(s.t("bible.settings.title"),
		sizeLabel, size, preview, font, invert, speech, titles, lang)
	m.ID = "modal-settings"
	m.OKText, m.CancelText = s.t("bible.settings.save"), s.t("bible.settings.cancel")
	m.SetButtons(widget.ModalOKCancel)
	m.SetControls(widget.ModalClose | widget.ModalMaximize)
	m.OnOK = func(*widget.Event) {
		p.FontSize = int(size.Value)
		p.Font = font.Value()
		p.Invert, p.Speech, p.ChapterTitles = invert.Checked, speech.Checked, titles.Checked
		if v := lang.Value(); v != "" {
			p.Lang = v
		}
		s.ApplyPreferences(p)
	}
	m.Open(s.Doc)
}

// ApplyPreferences keeps new preferences and puts them in effect: the look at once, the language
// by building the screen again.
func (s *Screen) ApplyPreferences(p domain.Preferences) {
	oldLang := s.tr.Lang()
	s.prefs = p.Normalize()
	if err := s.cfg.Bible.SavePreferences(s.prefs); err != nil {
		s.fail(err)
	}
	s.applyLook()
	s.tr.SetLang(s.uiLang())
	if s.tr.Lang() != oldLang {
		page := s.tabs.Current()
		s.build()
		s.tabs.Show(page)
	} else {
		s.tabs.Show(s.tabs.Current())
	}
	if s.reader != nil && s.view == viewReader {
		s.startReading()
	}
	s.Doc.Invalidate()
}
