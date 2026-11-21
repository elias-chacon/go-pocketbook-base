package main

import (
	"time"

	"go-pocketbook-base/examples/custom"
	"go-pocketbook-base/examples/recipes"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/showcase"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// example is one picture of the README.
type example struct {
	file  string
	width float64                                   // design px of the frame content
	extra float64                                   // extra height below the widget (for popups)
	build func() widget.Widget                      // the widget(s) to draw
	open  func(d *widget.Document, w widget.Widget) // optional: put the widget in a state (open a popup…)
}

func tapCenter(d *widget.Document, w widget.Widget) {
	c := w.Elem().Bounds.Center()
	d.HandlePointer(widget.PointerDown, c.X, c.Y)
	d.HandlePointer(widget.PointerUp, c.X, c.Y)
}

// captioned is a small muted caption above a widget.
func captioned(caption string, w widget.Widget) widget.Widget {
	l := widget.NewLabel(caption)
	l.Style.FontSize, l.Style.Color = widget.FontSmall, widget.Muted
	return widget.NewColumn(8, l, w)
}

// box is a labelled rectangle used to show layouts.
func box(text string, w, h float64) *widget.Panel {
	p := widget.NewRow(0, widget.NewLabel(text))
	s := &p.Style
	s.Width, s.Height = widget.Px(w), widget.Px(h)
	s.JustifyContent = widget.JustifyCenter
	s.BorderColor, s.BorderWidth, s.BorderRadius = widget.Ink, 3, 10
	s.Background = widget.Soft
	return p
}

// frame draws the outline of a container so its extent is visible.
func frame(p *widget.Panel) *widget.Panel {
	p.Style.BorderColor, p.Style.BorderWidth, p.Style.BorderRadius = widget.Mid, 2, 10
	p.Style.Padding = geom.All(12)
	return p
}

func examples() []example {
	return []example{
		{"flex", 640, 0, flexExample, nil},
		{"grid", 640, 0, gridExample, nil},
		{"label", 640, 0, labelExample, nil},
		{"button", 640, 0, buttonExample, nil},
		{"toggle", 760, 0, toggleExample, nil},
		{"combobox", 640, 250, comboExample, func(d *widget.Document, w widget.Widget) { tapCenter(d, w.Elem().Children()[1]) }},
		{"longcombobox", 900, 1750, longComboExample, func(d *widget.Document, w widget.Widget) { tapCenter(d, w.Elem().Children()[1]) }},
		{"oauthlogin", 760, 0, oauthExample, nil},
		{"oauthlogin-code", 760, 0, oauthExample, func(d *widget.Document, w widget.Widget) { tapCenter(d, d.GetElementByID("oauth-microsoft")) }},
		{"oauthlogin-qr", 760, 0, oauthExample, func(d *widget.Document, w widget.Widget) { tapCenter(d, d.GetElementByID("oauth-google")) }},
		{"dropdown", 640, 400, dropdownExample, func(d *widget.Document, w widget.Widget) { tapCenter(d, w.Elem().Children()[1]) }},
		{"menubar", 760, 300, menubarExample, func(d *widget.Document, w widget.Widget) {
			bar := w.(*widget.MenuBar)
			c := bar.MenuRect(2).Center() // the "Mais" menu
			d.HandlePointer(widget.PointerDown, c.X, c.Y)
			d.HandlePointer(widget.PointerUp, c.X, c.Y)
		}},
		{"input", 640, 0, inputExample, nil},
		{"textarea", 640, 0, textareaExample, nil},
		{"table", 900, 0, tableExample, nil},
		{"progress", 640, 0, progressExample, nil},
		{"gauge", 640, 0, gaugeExample, nil},
		{"slider", 640, 0, sliderExample, nil},
		{"chips", 640, 0, chipsExample, nil},
		{"calendar", 640, 0, calendarExample, nil},
		{"card", 640, 0, cardExample, nil},
		{"monthview", 1000, 0, monthViewExample, nil},
		{"timegrid", 1000, 0, timeGridExample, nil},
		{"pager", 760, 0, pagerExample, nil},
		{"textpager", 760, 0, textPagerExample, nil},
		{"paragraphpager", 760, 0, paragraphPagerExample, nil},
		{"image", 760, 0, imageExample, nil},
		{"modal", 760, 0, modalExample, nil},
		{"custom", 640, 0, customRecipe, nil},
		{"recipe-hello", 640, 0, func() widget.Widget { return recipes.Hello(noAction) }, nil},
		{"recipe-login", 640, 0, func() widget.Widget { return recipes.NewLoginForm(noLogin).Root }, nil},
		{"recipe-settings", 640, 0, func() widget.Widget { return recipes.NewSettings().Root }, nil},
		{"recipe-catalog", 760, 0, catalogRecipe, nil},
		{"recipe-datepicker", 640, 0, datepickerRecipe, nil},
		{"recipe-order", 640, 0, func() widget.Widget { return recipes.NewOrderForm().Root }, nil},
	}
}

func flexExample() widget.Widget {
	between := frame(widget.NewRow(12, box("A", 90, 64), box("B", 90, 64), box("C", 90, 64)))
	between.Style.JustifyContent = widget.JustifySpaceBetween

	center := frame(widget.NewRow(12, box("A", 90, 64), box("B", 90, 64)))
	center.Style.JustifyContent = widget.JustifyCenter

	grow := box("grow = 1", 0, 64)
	grow.Style.Width, grow.Style.FlexGrow = widget.Length{}, 1
	growRow := frame(widget.NewRow(12, box("fixo", 120, 64), grow))

	wrap := frame(widget.NewRow(12, box("1", 150, 56), box("2", 150, 56), box("3", 150, 56), box("4", 150, 56)))
	wrap.Style.FlexWrap, wrap.Style.AlignItems = true, widget.AlignStart

	column := frame(widget.NewColumn(10, box("linha 1", 0, 50), box("linha 2", 0, 50)))
	for _, k := range column.Children() {
		k.Elem().Style.Width = widget.Length{} // stretch to the column width
	}
	return widget.NewColumn(20,
		captioned("flex-direction: row · justify-content: space-between", between),
		captioned("justify-content: center", center),
		captioned("flex-grow: 1 ocupa o espaço que sobra", growRow),
		captioned("flex-wrap: wrap (quebra de linha)", wrap),
		captioned("flex-direction: column (estica na largura)", column))
}

func gridExample() widget.Widget {
	a, b, c := box("1fr", 0, 60), box("1fr", 0, 60), box("120 px", 0, 60)
	wide := box("span 2", 0, 60)
	wide.Style.GridColumnSpan = 2
	last := box("1fr", 0, 60)
	for _, p := range []*widget.Panel{a, b, c, wide, last} {
		p.Style.Width = widget.Length{}
	}
	g := frame(widget.NewGrid(12, []widget.Track{widget.FrTrack(1), widget.FrTrack(1), widget.PxTrack(120)}, a, b, c, wide, last))
	return captioned("GridColumns: 1fr · 1fr · 120 px, gap 12", g)
}

func labelExample() widget.Widget {
	title := widget.NewLabel("Título em negrito")
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	body := widget.NewLabel("O texto quebra linhas sozinho quando não cabe na largura disponível do elemento.")
	muted := widget.NewLabel("Texto secundário")
	muted.Style.Color, muted.Style.FontSize = widget.Muted, widget.FontSmall
	return widget.NewColumn(10, title, body, muted)
}

func buttonExample() widget.Widget {
	primary := widget.NewButton("Primário")
	primary.SetVariant(widget.VariantPrimary)
	secondary := widget.NewButton("Secundário")
	ghost := widget.NewButton("Fantasma")
	ghost.SetVariant(widget.VariantGhost)
	withIcon := widget.NewButton("Buscar")
	withIcon.Icon = gfx.IconSearch
	iconOnly := widget.NewIconButton(gfx.IconPlus)
	off := widget.NewButton("Desabilitado")
	off.Disabled = true
	offPrimary := widget.NewButton("Desabilitado")
	offPrimary.SetVariant(widget.VariantPrimary)
	offPrimary.Disabled = true

	row := func(ws ...widget.Widget) *widget.Panel {
		r := widget.NewRow(14, ws...)
		r.Style.FlexWrap = true
		return r
	}
	return widget.NewColumn(16, row(primary, secondary, ghost), row(withIcon, iconOnly), row(off, offPrimary))
}

func toggleExample() widget.Widget {
	on, off := widget.NewCheckbox("Marcado", true), widget.NewCheckbox("Desmarcado", false)
	dis := widget.NewCheckbox("Desabilitado", true)
	dis.Disabled = true
	r1, r2 := widget.NewRadio("Opção A", "g", "a"), widget.NewRadio("Opção B", "g", "b")
	r1.Checked = true
	s1, s2 := widget.NewSwitch("Ligado", true), widget.NewSwitch("Desligado", false)
	twoCols := []widget.Track{widget.FrTrack(1), widget.FrTrack(1)}
	return widget.NewGrid(24, twoCols,
		widget.NewColumn(2, on, off, dis),
		widget.NewColumn(2, r1, r2, s1, s2))
}

func comboExample() widget.Widget {
	cb := widget.NewComboBox("Selecione a cidade",
		widget.Option{Label: "São Paulo", Value: "sp"}, widget.Option{Label: "Rio de Janeiro", Value: "rj"},
		widget.Option{Label: "Curitiba", Value: "cwb"})
	cb.Index = 1
	return captioned("Combo-box com a lista aberta (opção atual destacada)", cb)
}

func longComboExample() widget.Widget {
	var options []widget.Option
	for _, name := range []string{"Afrikaans", "Alemão", "Árabe", "Armênio", "Basco", "Búlgaro", "Catalão", "Chinês", "Coreano", "Croata", "Dinamarquês", "Eslovaco", "Esloveno", "Espanhol", "Estoniano", "Finlandês", "Francês", "Galego", "Georgiano", "Grego", "Hebraico", "Hindi", "Holandês", "Húngaro", "Indonésio", "Inglês", "Irlandês", "Islandês", "Italiano", "Japonês", "Latim", "Letão", "Lituano", "Norueguês", "Persa", "Polonês", "Português", "Romeno", "Russo", "Sérvio", "Sueco", "Tailandês", "Tcheco", "Turco", "Ucraniano", "Vietnamita"} {
		options = append(options, widget.Option{Label: name, Value: name})
	}
	cb := widget.NewComboBox("Idioma", options...)
	cb.SearchPlaceholder, cb.NoMatchText = "Buscar", "Nada encontrado"
	cb.SetValue("Português")
	return captioned("Combo-box de lista longa: busca e páginas dentro do popup, na página da opção atual", cb)
}

func oauthExample() widget.Widget {
	login := widget.NewOAuthLogin(
		widget.OAuthProvider{ID: "google", Label: "Google",
			Start:    func() (widget.OAuthChallenge, error) { return widget.OAuthChallenge{URL: "https://accounts.google.com/o/oauth2/v2/auth?client_id=demo"}, nil },
			Complete: func(widget.OAuthChallenge, string) (bool, error) { return false, nil }},
		widget.OAuthProvider{ID: "microsoft", Label: "Microsoft",
			Start:    func() (widget.OAuthChallenge, error) { return widget.OAuthChallenge{URL: "https://microsoft.com/devicelogin", Code: "AB12-CD34"}, nil },
			Complete: func(widget.OAuthChallenge, string) (bool, error) { return false, nil }})
	login.Root.Style.Padding = geom.All(12)
	return captioned("Login OAuth: escolha do provedor", login.Root)
}

func dropdownExample() widget.Widget {
	dd := widget.NewDropdown("Ações",
		widget.MenuItem{Label: "Duplicar", Icon: gfx.IconPlus},
		widget.MenuItem{Label: "Renomear"},
		widget.MenuItem{Separator: true},
		widget.MenuItem{Label: "Excluir", Icon: gfx.IconClose},
		widget.MenuItem{Label: "Arquivar", Disabled: true})
	return captioned("Dropdown com o menu aberto", dd)
}

func menubarExample() widget.Widget {
	return widget.NewMenuBar(
		&widget.Menu{Label: "Entradas", Selected: true},
		&widget.Menu{Label: "Dados"},
		&widget.Menu{Label: "Mais", Items: []widget.MenuItem{
			{Label: "Sobre"}, {Separator: true}, {Label: "Sair", Icon: gfx.IconClose}}})
}

func inputExample() widget.Widget {
	name := widget.NewInput("Nome completo")
	filled := widget.NewInput("")
	filled.Value = "Maria Souza"
	pass := widget.NewInput("Senha")
	pass.Type, pass.Value = widget.TypePassword, "segredo"
	qty := widget.NewNumberInput(3, 0)
	qty.Min, qty.Max = 0, 99
	price := widget.NewNumberInput(19.9, 2)
	price.Prefix, price.DecimalSeparator, price.Step = "R$ ", ",", 0.5
	phone := widget.NewMaskedInput("(00) 00000-0000", "11987654321")
	search := widget.NewSearchInput("Buscar livros…")
	search.Value = "dom casmurro"
	empty := widget.NewSearchInput("Buscar livros…")
	return widget.NewColumn(16,
		captioned("texto (placeholder)", name), captioned("texto (preenchido)", filled),
		captioned("senha", pass), captioned("numérico, 0–99", qty),
		captioned("numérico com prefixo e decimais", price), captioned("máscara (00) 00000-0000", phone),
		captioned("busca, preenchida (limpar + lupa)", search), captioned("busca, vazia", empty))
}

func textareaExample() widget.Widget {
	filled := widget.NewTextArea("", 3)
	filled.Value = "Linha 1 do texto.\nO texto longo quebra de linha automaticamente quando passa da largura."
	empty := widget.NewTextArea("Observações…", 2)
	return widget.NewColumn(16, captioned("com texto", filled), captioned("vazia (placeholder)", empty))
}

func tableExample() widget.Widget {
	t := widget.NewTable(4,
		widget.TableColumn{Title: "Livro", Flex: 3, Sortable: true},
		widget.TableColumn{Title: "Autor", Flex: 2, Sortable: true},
		widget.TableColumn{Title: "Págs", Flex: 1, Align: gfx.AlignEnd, Sortable: true})
	t.Rows = [][]string{
		{"Dom Casmurro", "Machado de Assis", "256"}, {"Vidas Secas", "Graciliano Ramos", "176"},
		{"Iracema", "José de Alencar", "160"}, {"Macunaíma", "Mário de Andrade", "224"},
		{"O Cortiço", "Aluísio Azevedo", "320"}, {"A Hora da Estrela", "Clarice Lispector", "88"},
	}
	t.Summary = func(from, to, total int) string { return itoa(from) + "–" + itoa(to) + " de " + itoa(total) }
	t.SortCol, t.SortDesc, t.Selected = 2, true, 4 // ordenada por "Págs" (maior primeiro), "O Cortiço" selecionado
	return t
}

func itoa(n int) string {
	const digits = "0123456789"
	if n < 10 {
		return string(digits[n])
	}
	return itoa(n/10) + string(digits[n%10])
}

func progressExample() widget.Widget {
	done := widget.NewProgress(100)
	done.Text = "Concluído"
	return widget.NewColumn(16, captioned("0%", widget.NewProgress(0)), captioned("35%", widget.NewProgress(35)),
		captioned("80%", widget.NewProgress(80)), captioned("texto próprio", done))
}

func pagerExample() widget.Widget {
	senders := []string{"Ana Souza", "Banco do Brasil", "Carlos Lima", "Diretoria", "Editora Nova", "Fernanda", "GitHub", "Helena"}
	var rows []widget.Widget
	for i := 0; i < 10; i++ {
		who := widget.NewLabel(senders[i%len(senders)])
		who.Style.Bold, who.Style.FlexGrow = true, 1
		subject := widget.NewLabel("Assunto " + string(rune('1'+i%9)))
		subject.Style.FlexGrow = 2
		row := widget.NewRow(16, who, subject)
		row.Style.Padding = geom.XY(14, 20)
		row.Style.BorderColor, row.Style.BorderWidth, row.Style.BorderRadius = widget.Mid, 2, widget.Radius
		rows = append(rows, row)
	}
	return widget.NewPager(4, rows...).Root
}

func paragraphPagerExample() widget.Widget {
	verses := []string{
		"No princípio, Deus criou os céus e a terra.",
		"A terra era sem forma e vazia; havia trevas sobre a face do abismo, e o Espírito de Deus pairava sobre as águas.",
		"Disse Deus: Haja luz. E houve luz.",
	}
	p := widget.NewParagraphPager(verses, []string{"1", "2", "3"})
	p.Lines = 7
	return p
}

func textPagerExample() widget.Widget {
	text := "Um texto longo se lê virando páginas, não rolando a tela. Cada página tem um número fixo de linhas, então nada se mexe enquanto você lê, e a tela do leitor é atualizada uma vez por página. "
	t := widget.NewTextPager(text + text + text + text)
	t.Lines = 6
	return t
}

func gaugeExample() widget.Widget {
	g1, g2 := widget.NewGauge(0, 100, 72, "%"), widget.NewGauge(0, 240, 135, " km/h")
	g1.Label, g2.Label = "Bateria", "Velocidade"
	g1.Style.Width, g2.Style.Width = widget.Px(270), widget.Px(270)
	return widget.NewRow(20, g1, g2)
}

func sliderExample() widget.Widget {
	left := widget.NewSlider(0, 100, 50, 0)
	center := widget.NewSlider(0, 100, 50, 50)
	right := widget.NewSlider(0, 100, 50, 100)
	small := widget.NewSlider(0, 10, 1, 3)
	small.ThumbSize, small.Style.Width = 40, widget.Px(300)
	big := widget.NewSlider(0, 100, 10, 70)
	big.ThumbSize = 84
	return widget.NewColumn(8,
		captioned("esquerda  O============", left), captioned("centro  ======O======", center),
		captioned("direita  ============O", right), captioned("tamanho: largura 300 px, botão 40", small),
		captioned("tamanho: botão 84", big))
}

func chipsExample() widget.Widget {
	multi := widget.NewChipGroup(false)
	for i, l := range []string{"Ficção", "Ciência", "História"} {
		c := widget.NewChip(l)
		c.Selected = i == 0
		multi.AppendChild(c)
	}
	tags := widget.NewChipGroup(false)
	for _, l := range []string{"#leitura", "#e-ink"} {
		c := widget.NewChip(l)
		c.Closable = true
		tags.AppendChild(c)
	}
	return widget.NewColumn(16, captioned("seleção (o chip marcado fica preenchido e com visto)", multi), captioned("removíveis (toque no ×)", tags))
}

func monthViewExample() widget.Widget {
	m := widget.NewMonthView(time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC))
	m.Locale = widget.LocalePTBR
	m.Today = time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	m.Entries = func(day time.Time) []string {
		switch {
		case day.Month() != time.November:
			return nil
		case day.Day() == 18:
			return []string{"09:00 Reunião", "14:00 Dentista", "Feriado", "17:00 Yoga", "19:00 Jantar"}
		case day.Weekday() == time.Friday:
			return []string{"10:00 Revisão"}
		}
		return nil
	}
	return m
}

func timeGridExample() widget.Widget {
	monday := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	days := make([]time.Time, 5)
	for i := range days {
		days[i] = monday.AddDate(0, 0, i)
	}
	g := widget.NewTimeGrid(days...)
	g.Locale = widget.LocalePTBR
	g.Today = days[2]
	g.FirstHour, g.LastHour = 8, 14
	g.Events = []widget.GridEvent{
		{ID: "a", Title: "Reunião", Detail: "09:00–10:30", Start: monday.Add(9 * time.Hour), End: monday.Add(10*time.Hour + 30*time.Minute)},
		{ID: "b", Title: "Dentista", Detail: "10:00–11:00", Start: monday.Add(10 * time.Hour), End: monday.Add(11 * time.Hour)},
		{ID: "c", Title: "Almoço", Start: days[2].Add(12 * time.Hour), End: days[2].Add(13 * time.Hour)},
		{ID: "d", Title: "Feriado", Start: days[3], End: days[3].AddDate(0, 0, 1), AllDay: true},
	}
	return g
}

func calendarExample() widget.Widget {
	cal := widget.NewCalendar(time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC))
	cal.Locale = widget.LocalePTBR
	cal.Today = time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	return cal
}

// imageExample shows the same bitmap at its natural size and with each fit. A
// column in a row takes the whole width unless it has one, so each gets its own.
func imageExample() widget.Widget {
	icon := showcase.RenderIcon(128, false)
	column := func(caption string, width float64, img *widget.Image) widget.Widget {
		c := captioned(caption, img)
		c.Elem().Style.Width = widget.Px(width)
		return c
	}
	wide := func(caption string, fit gfx.Fit) widget.Widget {
		img := widget.NewImage(icon)
		img.Style.Width, img.Style.Height = widget.Px(170), widget.Px(96)
		img.Style.BorderColor, img.Style.BorderWidth = widget.Mid, 2
		img.Fit = fit
		return column(caption, 170, img)
	}
	row := widget.NewRow(14, column("natural", 140, widget.NewImage(icon)),
		wide("contain", gfx.FitContain), wide("cover", gfx.FitCover), wide("fill", gfx.FitFill))
	row.Style.AlignItems = widget.AlignStart
	return row
}

// modalExample draws the dialog inline (the library draws the same widget over
// the page when it is opened with Modal.Open).
func modalExample() widget.Widget {
	nick := widget.NewInput("Apelido")
	m := widget.NewModal("Editar perfil", widget.NewLabel("Escolha como você aparece para os outros."), nick)
	m.OKText, m.CancelText = "Salvar", "Cancelar"
	m.SetButtons(widget.ModalOKCancel)
	m.SetControls(widget.ModalClose | widget.ModalMaximize | widget.ModalMinimize)
	return m
}

func cardExample() widget.Widget {
	ok := widget.NewButton("Saiba mais")
	ok.SetVariant(widget.VariantPrimary)
	later := widget.NewButton("Depois")
	return widget.NewCard("Título do card", "Subtítulo opcional",
		widget.NewLabel("Um card agrupa conteúdo relacionado: título, subtítulo, corpo e ações."),
		widget.NewRow(14, ok, later))
}

func catalogRecipe() widget.Widget {
	return recipes.NewCatalog([][]string{
		{"Dom Casmurro", "Machado de Assis", "256"}, {"Vidas Secas", "Graciliano Ramos", "176"},
		{"Iracema", "José de Alencar", "160"}, {"Macunaíma", "Mário de Andrade", "224"},
		{"O Cortiço", "Aluísio Azevedo", "320"},
	}).Root
}

func datepickerRecipe() widget.Widget {
	p := recipes.NewDatePicker(time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC), noDate)
	p.Calendar.Today = time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	return p.Root
}

func customRecipe() widget.Widget {
	counter := custom.NewTapCounter()
	counter.Count = 2
	badge := custom.NewBadge(3)
	return widget.NewRow(20, badge, counter)
}

// The pictures only show the screens, so their callbacks have nothing to do.

func noAction() {
	// Intentionally empty: a picture has nobody to notify.
}

func noLogin(string, bool) {
	// Intentionally empty: a picture has nobody to sign in.
}

func noDate(time.Time) {
	// Intentionally empty: a picture has nobody to notify.
}
