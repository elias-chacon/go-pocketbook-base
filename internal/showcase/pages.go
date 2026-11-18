package showcase

import (
	"fmt"
	"strings"
	"time"

	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// ---- Entradas: text, number, mask, search, text area ----

func (a *App) inputsPage() widget.Widget {
	name := widget.NewInput(a.t("inputs.name.placeholder"))
	name.OnChange = func(e *widget.Event) { a.say(a.t("say.name", name.Value)) }

	pass := widget.NewInput(a.t("inputs.password.placeholder"))
	pass.Type = widget.TypePassword

	search := widget.NewSearchInput(a.t("inputs.search.placeholder"))
	search.OnSubmit = func(e *widget.Event) { a.say(a.t("say.search", search.Value)) }

	qty := widget.NewNumberInput(1, 0)
	qty.Min, qty.Max = 0, 99
	qty.OnChange = func(e *widget.Event) { a.say(a.t("say.qty", qty.Display())) }

	price := widget.NewNumberInput(19.9, 2)
	price.Prefix, price.Step, price.Min, price.Max = a.t("inputs.price.prefix"), 0.5, 0, 1000
	price.DecimalSeparator = a.t("inputs.price.decimal")
	price.OnChange = func(e *widget.Event) { a.say(a.t("say.price", price.Display())) }

	phone := widget.NewMaskedInput("(00) 00000-0000", "11987654321")
	phone.OnChange = func(e *widget.Event) { a.say(a.t("say.phone", phone.Display())) }

	cpf := widget.NewMaskedInput("000.000.000-00", "")
	cpf.NoSteppers, cpf.Placeholder = true, "000.000.000-00"

	notes := widget.NewTextArea(a.t("inputs.notes.placeholder"), 4)
	notes.OnChange = func(e *widget.Event) { a.say(a.t("say.notes", len([]rune(notes.Value)))) }

	return twoColPage(
		widget.NewCard(a.t("inputs.card.text.title"), a.t("inputs.card.text.subtitle"),
			field(a.t("inputs.field.name"), withID(name, "input-name")),
			field(a.t("inputs.field.password"), withID(pass, "input-password")),
			field(a.t("inputs.field.search"), withID(search, "input-search"))),
		widget.NewCard(a.t("inputs.card.number.title"), a.t("inputs.card.number.subtitle"),
			field(a.t("inputs.field.qty"), withID(qty, "input-qty")),
			field(a.t("inputs.field.price"), withID(price, "input-price")),
			field(a.t("inputs.field.phone"), withID(phone, "input-phone")),
			field(a.t("inputs.field.taxid"), withID(cpf, "input-cpf"))),
		span(widget.NewCard(a.t("inputs.card.long.title"), a.t("inputs.card.long.subtitle"),
			withID(notes, "textarea-notes"))),
	)
}

// ---- Escolhas: check, radio, switch, combo, dropdown, chips, sliders ----

func (a *App) choicesPage() widget.Widget {
	checks := widget.NewColumn(4)
	for i := 1; i <= 3; i++ {
		cb := widget.NewCheckbox(a.t(fmt.Sprintf("choices.check.%d", i)), i == 1)
		cb.ID = fmt.Sprintf("check-%d", i)
		cb.OnChange = func(e *widget.Event) { a.say(fmt.Sprintf("%s: %v", cb.Label, cb.Checked)) }
		checks.AppendChild(cb)
	}

	radios := widget.NewColumn(4)
	for i := 1; i <= 3; i++ {
		label := a.t(fmt.Sprintf("choices.plan.%d", i))
		r := widget.NewRadio(label, "plan", label)
		r.ID, r.Checked = fmt.Sprintf("radio-%d", i), i == 2
		r.OnChange = func(e *widget.Event) { a.say(a.t("say.plan", r.Value)) }
		radios.AppendChild(r)
	}

	switches := widget.NewColumn(4)
	for i := 1; i <= 3; i++ {
		sw := widget.NewSwitch(a.t(fmt.Sprintf("choices.switch.%d", i)), i != 2)
		sw.ID = fmt.Sprintf("switch-%d", i)
		sw.OnChange = func(e *widget.Event) { a.say(fmt.Sprintf("%s: %v", sw.Label, sw.Checked)) }
		switches.AppendChild(sw)
	}

	// The cities are sample data: they are not translated.
	city := widget.NewComboBox(a.t("choices.city.placeholder"),
		widget.Option{Label: "São Paulo", Value: "sp"}, widget.Option{Label: "Rio de Janeiro", Value: "rj"},
		widget.Option{Label: "Belo Horizonte", Value: "bh"}, widget.Option{Label: "Curitiba", Value: "cwb"})
	city.OnChange = func(e *widget.Event) {
		a.say(a.t("say.city", city.Options[city.Index].Label, city.Value()))
	}

	actions := widget.NewDropdown(a.t("choices.actions"),
		widget.MenuItem{Label: a.t("choices.action.duplicate"), Icon: gfx.IconPlus, OnSelect: func() { a.say(a.t("say.duplicated")) }},
		widget.MenuItem{Label: a.t("choices.action.rename"), OnSelect: func() { a.say(a.t("say.rename")) }},
		widget.MenuItem{Separator: true},
		widget.MenuItem{Label: a.t("choices.action.delete"), Icon: gfx.IconClose, OnSelect: func() { a.say(a.t("say.deleted")) }},
		widget.MenuItem{Label: a.t("choices.action.archive"), Disabled: true})

	return twoColPage(
		widget.NewCard(a.t("choices.card.check.title"), a.t("choices.card.check.subtitle"), checks),
		widget.NewCard(a.t("choices.card.radio.title"), a.t("choices.card.radio.subtitle"), radios),
		widget.NewCard(a.t("choices.card.switch.title"), a.t("choices.card.switch.subtitle"), switches),
		widget.NewCard(a.t("choices.card.combo.title"), a.t("choices.card.combo.subtitle"),
			field(a.t("choices.field.combo"), withID(city, "combo-city")),
			field(a.t("choices.field.dropdown"), withID(actions, "dropdown-actions"))),
		widget.NewCard(a.t("choices.card.chips.title"), a.t("choices.card.chips.subtitle"), a.chips()),
		widget.NewCard(a.t("choices.card.slider.title"), a.t("choices.card.slider.subtitle"), a.sliders()),
	)
}

// chipGroup builds a group of three chips named choices keyed prefix.1..n.
func (a *App) chipGroup(id, prefix, chipID string, n int, exclusive bool, onChange func(g *widget.ChipGroup)) *widget.ChipGroup {
	g := widget.NewChipGroup(exclusive)
	g.ID = id
	for i := 1; i <= n; i++ {
		c := widget.NewChip(a.t(fmt.Sprintf("%s.%d", prefix, i)))
		c.ID, c.Selected = fmt.Sprintf("%s-%d", chipID, i), i == 1
		if onChange != nil {
			c.OnChange = func(e *widget.Event) { onChange(g) }
		}
		g.AppendChild(c)
	}
	return g
}

func (a *App) chips() widget.Widget {
	multi := a.chipGroup("chips-multi", "chips.multi", "chip", 3, false,
		func(g *widget.ChipGroup) { a.say(a.t("say.categories", g.SelectedLabels())) })
	order := a.chipGroup("chips-order", "chips.order", "chip-order", 3, true,
		func(g *widget.ChipGroup) { a.say(a.t("say.sortby", g.SelectedLabels())) })

	tags := widget.NewChipGroup(false)
	tags.ID = "chips-tags"
	for i := 1; i <= 2; i++ {
		c := widget.NewChip(a.t(fmt.Sprintf("chips.tag.%d", i)))
		c.ID, c.Closable = fmt.Sprintf("chip-tag-%d", i), true
		tags.AppendChild(c)
	}
	return widget.NewColumn(18,
		field(a.t("chips.field.multi"), multi),
		field(a.t("chips.field.exclusive"), order),
		field(a.t("chips.field.removable"), tags))
}

func (a *App) sliders() widget.Widget {
	val := widget.NewLabel("40")
	val.Style.Bold, val.Style.TextAlign, val.Style.Width = true, gfx.AlignEnd, widget.Px(90)

	basic := widget.NewSlider(0, 100, 5, 40)
	basic.ID, basic.Style.FlexGrow = "slider-basic", 1
	basic.OnInput = func(e *widget.Event) { val.Text = fmt.Sprintf("%.0f", basic.Value) }
	basic.OnChange = func(e *widget.Event) { a.say(a.t("say.volume", basic.Value)) }
	a.refresh = func() { val.Text = fmt.Sprintf("%.0f", basic.Value) }

	thin := widget.NewSlider(0, 10, 1, 3)
	thin.ID, thin.ThumbSize, thin.Style.Width = "slider-small", 40, widget.Px(300)
	thin.OnChange = func(e *widget.Event) { a.say(a.t("say.small", thin.Value)) }

	// Slide button with three positions: O====== (left), ===O=== (center), ======O (right).
	pos := widget.NewSlider(0, 100, 50, 50)
	pos.ID, pos.Style.Width = "slider-position", widget.Px(420)
	pos.OnChange = func(e *widget.Event) {
		where := map[float64]string{0: "sliders.left", 50: "sliders.center", 100: "sliders.right"}[pos.Value]
		a.say(a.t("say.position", a.t(where)))
	}

	big := widget.NewSlider(0, 100, 10, 70)
	big.ID, big.ThumbSize = "slider-big", 84
	big.OnChange = func(e *widget.Event) { a.say(a.t("say.large", big.Value)) }

	return widget.NewColumn(18,
		field(a.t("sliders.field.full"), widget.NewRow(12, basic, val)),
		field(a.t("sliders.field.position"), pos),
		field(a.t("sliders.field.small"), thin),
		field(a.t("sliders.field.large"), big))
}

// ---- Dados: table, progress, gauge, card ----

// machado is an author that appears several times in the sample table.
const machado = "Machado de Assis"

func (a *App) dataPage() widget.Widget {
	t := widget.NewTable(5,
		widget.TableColumn{Title: a.t("data.col.book"), Flex: 3, Sortable: true},
		widget.TableColumn{Title: a.t("data.col.author"), Flex: 2, Sortable: true},
		widget.TableColumn{Title: a.t("data.col.pages"), Flex: 1, Align: gfx.AlignEnd, Sortable: true})
	t.ID = "table-books"
	t.Summary = func(from, to, total int) string { return a.t("data.summary", from, to, total) }
	// The books are sample data: they are not translated.
	t.Rows = [][]string{
		{"Dom Casmurro", machado, "256"}, {"Grande Sertão: Veredas", "Guimarães Rosa", "608"},
		{"Memórias Póstumas", machado, "368"}, {"Capitães da Areia", "Jorge Amado", "280"},
		{"A Hora da Estrela", "Clarice Lispector", "88"}, {"Vidas Secas", "Graciliano Ramos", "176"},
		{"O Cortiço", "Aluísio Azevedo", "320"}, {"Iracema", "José de Alencar", "160"},
		{"O Guarani", "José de Alencar", "432"}, {"Macunaíma", "Mário de Andrade", "224"},
		{"Quincas Borba", machado, "304"}, {"São Bernardo", "Graciliano Ramos", "208"},
	}
	t.OnSelect = func(e *widget.Event) { a.say(a.t("say.book", t.Rows[t.Selected][0])) }

	bar := widget.NewProgress(35)
	bar.ID = "progress-download"
	more := widget.NewButton(a.t("data.advance"))
	more.ID = "button-advance"
	more.OnClick = func(e *widget.Event) {
		bar.Value = float64(int(bar.Value+15) % 115)
		a.say(a.t("say.download", bar.Value))
	}
	done := widget.NewProgress(100)
	done.Text = a.t("data.done")

	g1, g2 := widget.NewGauge(0, 100, 72, "%"), widget.NewGauge(0, 240, 135, " km/h")
	g1.Label, g2.Label = a.t("data.battery"), a.t("data.speed")
	g1.Style.Width, g2.Style.Width = widget.Px(260), widget.Px(260)

	learn := widget.NewButton(a.t("data.learn"))
	learn.SetVariant(widget.VariantPrimary)
	learn.OnClick = func(e *widget.Event) { a.say(a.t("say.card", learn.Text)) }
	later := widget.NewButton(a.t("data.later"))
	later.OnClick = func(e *widget.Event) { a.say(a.t("say.card", later.Text)) }
	body := widget.NewLabel(a.t("data.card.body"))

	return widget.NewGrid(24, twoColumns,
		span(widget.NewCard(a.t("data.card.table.title"), a.t("data.card.table.subtitle"), t)),
		widget.NewCard(a.t("data.card.progress.title"), a.t("data.card.progress.subtitle"),
			field(a.t("data.field.download"), bar), more, field(a.t("data.field.sync"), done)),
		widget.NewCard(a.t("data.card.gauge.title"), a.t("data.card.gauge.subtitle"),
			widget.NewRow(16, g1, g2)),
		span(widget.NewCard(a.t("data.card.card.title"), a.t("data.card.card.subtitle"), body, widget.NewRow(16, learn, later))),
	)
}

// ---- Calendário ----

// calendarLocale is the calendar's language: English only for English.
func (a *App) calendarLocale() widget.Locale {
	if strings.HasPrefix(a.tr.Lang(), "en") {
		return widget.LocaleEN
	}
	return widget.LocalePTBR
}

func (a *App) calendarPage() widget.Widget {
	cal := widget.NewCalendar(time.Now())
	cal.ID, cal.Locale = "calendar", a.calendarLocale()

	chosen := widget.NewLabel("")
	chosen.Style.FontSize, chosen.Style.Bold = widget.FontTitle, true
	describe := func() {
		if cal.Selected.IsZero() {
			chosen.Text = a.t("calendar.none")
			return
		}
		chosen.Text = cal.Selected.Format(a.t("calendar.layout"))
	}
	describe()
	a.refresh = describe
	cal.OnChange = func(e *widget.Event) { describe(); a.say(a.t("say.date", chosen.Text)) }

	jump := func(label string, f func()) *widget.Button {
		b := widget.NewButton(label)
		b.OnClick = func(e *widget.Event) { f(); describe() }
		return b
	}
	pick := func(t time.Time) { cal.Selected, cal.Year, cal.Month = t, t.Year(), t.Month() }
	today := jump(a.t("calendar.today"), func() { pick(time.Now()) })
	week := jump(a.t("calendar.week"), func() { pick(time.Now().AddDate(0, 0, 7)) })
	clear := jump(a.t("calendar.clear"), func() { cal.Selected = time.Time{} })
	for i, b := range []*widget.Button{today, week, clear} {
		b.ID = fmt.Sprintf("cal-shortcut-%d", i+1)
	}

	return twoColPage(
		cal,
		widget.NewColumn(24,
			widget.NewCard(a.t("calendar.card.selected.title"), "", chosen),
			widget.NewCard(a.t("calendar.card.shortcuts.title"), a.t("calendar.card.shortcuts.subtitle"), widget.NewRow(14, today, week, clear))),
	)
}
