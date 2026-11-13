package demo

import (
	"fmt"
	"time"

	"go-pocketbook-base/internal/gfx"
	"go-pocketbook-base/internal/widget"
)

// ---- Entradas: text, number, mask, search, text area ----

func (a *App) inputsPage() widget.Widget {
	name := widget.NewInput("Nome completo")
	name.OnChange = func(e *widget.Event) { a.say("Nome: %s", name.Value) }

	pass := widget.NewInput("Senha")
	pass.Type = widget.TypePassword

	search := widget.NewSearchInput("Buscar livros…")
	search.OnSubmit = func(e *widget.Event) { a.say("Buscando por “%s”", search.Value) }

	qty := widget.NewNumberInput(1, 0)
	qty.Min, qty.Max = 0, 99
	qty.OnChange = func(e *widget.Event) { a.say("Quantidade: %s", qty.Display()) }

	price := widget.NewNumberInput(19.9, 2)
	price.Prefix, price.Step, price.Min, price.Max, price.DecimalSeparator = "R$ ", 0.5, 0, 1000, ","
	price.OnChange = func(e *widget.Event) { a.say("Preço: %s", price.Display()) }

	phone := widget.NewMaskedInput("(00) 00000-0000", "11987654321")
	phone.OnChange = func(e *widget.Event) { a.say("Telefone: %s", phone.Display()) }

	cpf := widget.NewMaskedInput("000.000.000-00", "")
	cpf.NoSteppers, cpf.Placeholder = true, "000.000.000-00"

	notes := widget.NewTextArea("Observações…", 4)
	notes.OnChange = func(e *widget.Event) { a.say("Observações: %d caracteres", len([]rune(notes.Value))) }

	return twoColPage(
		widget.NewCard("Texto", "O teclado do sistema abre ao tocar",
			field("Nome", withID(name, "input-name")),
			field("Senha", withID(pass, "input-password")),
			field("Busca (lupa e limpar)", withID(search, "input-search"))),
		widget.NewCard("Numérico", "Botões − / + e máscara de dígitos",
			field("Quantidade (0–99)", withID(qty, "input-qty")),
			field("Preço (passo 0,5)", withID(price, "input-price")),
			field("Telefone (máscara)", withID(phone, "input-phone")),
			field("CPF (máscara)", withID(cpf, "input-cpf"))),
		span(widget.NewCard("Texto longo", "Text-area com quebra de linha",
			withID(notes, "textarea-notes"))),
	)
}

// ---- Escolhas: check, radio, switch, combo, dropdown, chips, sliders ----

func (a *App) choicesPage() widget.Widget {
	checks := widget.NewColumn(4)
	for i, l := range []string{"Receber e-mails", "Notificações push", "Modo offline"} {
		cb := widget.NewCheckbox(l, i == 0)
		cb.ID = fmt.Sprintf("check-%d", i+1)
		cb.OnChange = func(e *widget.Event) { a.say("%s: %v", cb.Label, cb.Checked) }
		checks.AppendChild(cb)
	}

	radios := widget.NewColumn(4)
	for i, l := range []string{"Gratuito", "Pro", "Empresa"} {
		r := widget.NewRadio(l, "plano", l)
		r.ID, r.Checked = fmt.Sprintf("radio-%d", i+1), i == 1
		r.OnChange = func(e *widget.Event) { a.say("Plano escolhido: %s", r.Value) }
		radios.AppendChild(r)
	}

	switches := widget.NewColumn(4)
	for i, l := range []string{"Wi-Fi", "Bluetooth", "Luz frontal"} {
		sw := widget.NewSwitch(l, i != 1)
		sw.ID = fmt.Sprintf("switch-%d", i+1)
		sw.OnChange = func(e *widget.Event) { a.say("%s: %v", sw.Label, sw.Checked) }
		switches.AppendChild(sw)
	}

	city := widget.NewComboBox("Selecione a cidade",
		widget.Option{Label: "São Paulo", Value: "sp"}, widget.Option{Label: "Rio de Janeiro", Value: "rj"},
		widget.Option{Label: "Belo Horizonte", Value: "bh"}, widget.Option{Label: "Curitiba", Value: "cwb"})
	city.OnChange = func(e *widget.Event) { a.say("Cidade: %s (%s)", city.Options[city.Index].Label, city.Value()) }

	actions := widget.NewDropdown("Ações",
		widget.MenuItem{Label: "Duplicar", Icon: gfx.IconPlus, OnSelect: func() { a.say("Duplicado") }},
		widget.MenuItem{Label: "Renomear", OnSelect: func() { a.say("Renomear…") }},
		widget.MenuItem{Separator: true},
		widget.MenuItem{Label: "Excluir", Icon: gfx.IconClose, OnSelect: func() { a.say("Excluído") }},
		widget.MenuItem{Label: "Arquivar (indisponível)", Disabled: true})

	return twoColPage(
		widget.NewCard("Checkbox", "Várias opções independentes", checks),
		widget.NewCard("Radio button", "Uma opção do grupo “plano”", radios),
		widget.NewCard("Switch", "Liga / desliga", switches),
		widget.NewCard("Combo-box e dropdown", "Lista de valores × menu de ações",
			field("Combo-box", withID(city, "combo-city")),
			field("Dropdown", withID(actions, "dropdown-actions"))),
		widget.NewCard("Chips", "Seleção múltipla, exclusiva e removíveis", a.chips()),
		widget.NewCard("Slider", "Tamanho configurável", a.sliders()),
	)
}

func (a *App) chips() widget.Widget {
	multi := widget.NewChipGroup(false)
	for i, l := range []string{"Ficção", "Ciência", "História"} {
		c := widget.NewChip(l)
		c.ID, c.Selected = fmt.Sprintf("chip-%d", i+1), i == 0
		c.OnChange = func(e *widget.Event) { a.say("Categorias: %v", multi.SelectedLabels()) }
		multi.AppendChild(c)
	}
	order := widget.NewChipGroup(true)
	for i, l := range []string{"Título", "Autor", "Data"} {
		c := widget.NewChip(l)
		c.ID, c.Selected = fmt.Sprintf("chip-order-%d", i+1), i == 0
		c.OnChange = func(e *widget.Event) { a.say("Ordenar por %v", order.SelectedLabels()) }
		order.AppendChild(c)
	}
	tags := widget.NewChipGroup(false)
	for i, l := range []string{"#leitura", "#e-ink"} {
		c := widget.NewChip(l)
		c.ID, c.Closable = fmt.Sprintf("chip-tag-%d", i+1), true
		tags.AppendChild(c)
	}
	return widget.NewColumn(18, field("Múltipla", multi), field("Exclusiva", order), field("Removíveis (toque no ×)", tags))
}

func (a *App) sliders() widget.Widget {
	val := widget.NewLabel("40")
	val.Style.Bold, val.Style.TextAlign, val.Style.Width = true, gfx.AlignEnd, widget.Px(90)

	basic := widget.NewSlider(0, 100, 5, 40)
	basic.ID, basic.Style.FlexGrow = "slider-basic", 1
	basic.OnInput = func(e *widget.Event) { val.Text = fmt.Sprintf("%.0f", basic.Value) }
	basic.OnChange = func(e *widget.Event) { a.say("Volume: %.0f", basic.Value) }

	thin := widget.NewSlider(0, 10, 1, 3)
	thin.ID, thin.ThumbSize, thin.Style.Width = "slider-small", 40, widget.Px(300)
	thin.OnChange = func(e *widget.Event) { a.say("Pequeno (300 px, botão 40): %.0f", thin.Value) }

	// Slide button with three positions: O====== (left), ===O=== (center), ======O (right).
	pos := widget.NewSlider(0, 100, 50, 50)
	pos.ID, pos.Style.Width = "slider-position", widget.Px(420)
	pos.OnChange = func(e *widget.Event) {
		a.say("Posição: %s", map[float64]string{0: "esquerda", 50: "centro", 100: "direita"}[pos.Value])
	}

	big := widget.NewSlider(0, 100, 10, 70)
	big.ID, big.ThumbSize = "slider-big", 84
	big.OnChange = func(e *widget.Event) { a.say("Grande (botão 84): %.0f", big.Value) }

	return widget.NewColumn(18,
		field("Largura total", widget.NewRow(12, basic, val)),
		field("Slide button: esquerda · centro · direita", pos),
		field("Pequeno: 300 px, botão 40", thin),
		field("Grande: botão 84", big))
}

// ---- Dados: table, progress, gauge, card ----

// machado is an author that appears several times in the sample table.
const machado = "Machado de Assis"

func (a *App) dataPage() widget.Widget {
	t := widget.NewTable(5,
		widget.TableColumn{Title: "Livro", Flex: 3, Sortable: true},
		widget.TableColumn{Title: "Autor", Flex: 2, Sortable: true},
		widget.TableColumn{Title: "Págs", Flex: 1, Align: gfx.AlignEnd, Sortable: true})
	t.ID = "table-books"
	t.Summary = func(from, to, total int) string { return fmt.Sprintf("%d–%d de %d", from, to, total) }
	t.Rows = [][]string{
		{"Dom Casmurro", machado, "256"}, {"Grande Sertão: Veredas", "Guimarães Rosa", "608"},
		{"Memórias Póstumas", machado, "368"}, {"Capitães da Areia", "Jorge Amado", "280"},
		{"A Hora da Estrela", "Clarice Lispector", "88"}, {"Vidas Secas", "Graciliano Ramos", "176"},
		{"O Cortiço", "Aluísio Azevedo", "320"}, {"Iracema", "José de Alencar", "160"},
		{"O Guarani", "José de Alencar", "432"}, {"Macunaíma", "Mário de Andrade", "224"},
		{"Quincas Borba", machado, "304"}, {"São Bernardo", "Graciliano Ramos", "208"},
	}
	t.OnSelect = func(e *widget.Event) { a.say("Livro: %s", t.Rows[t.Selected][0]) }

	bar := widget.NewProgress(35)
	bar.ID = "progress-download"
	more := widget.NewButton("Avançar 15%")
	more.ID = "button-advance"
	more.OnClick = func(e *widget.Event) {
		bar.Value = float64(int(bar.Value+15) % 115)
		a.say("Download: %.0f%%", bar.Value)
	}
	done := widget.NewProgress(100)
	done.Text = "Concluído"

	g1, g2 := widget.NewGauge(0, 100, 72, "%"), widget.NewGauge(0, 240, 135, " km/h")
	g1.Label, g2.Label = "Bateria", "Velocidade"
	g1.Style.Width, g2.Style.Width = widget.Px(260), widget.Px(260)

	about := widget.NewButton("Saiba mais")
	about.SetVariant(widget.VariantPrimary)
	about.OnClick = func(e *widget.Event) { a.say("Card: Saiba mais") }
	later := widget.NewButton("Depois")
	later.OnClick = func(e *widget.Event) { a.say("Card: Depois") }
	body := widget.NewLabel("Um card agrupa conteúdo relacionado com título, subtítulo, corpo e ações.")

	return widget.NewGrid(24, twoColumns,
		span(widget.NewCard("Tabela de dados", "Toque no cabeçalho para ordenar e nas setas para paginar", t)),
		widget.NewCard("Barra de progresso", "O texto inverte sobre o preenchimento",
			field("Download", bar), more, field("Sincronização", done)),
		widget.NewCard("Gauge", "Medidor semicircular",
			widget.NewRow(16, g1, g2)),
		span(widget.NewCard("Card", "Título, subtítulo, corpo e ações", body, widget.NewRow(16, about, later))),
	)
}

// ---- Calendário ----

func (a *App) calendarPage() widget.Widget {
	cal := widget.NewCalendar(time.Now())
	cal.ID, cal.Locale = "calendar", widget.LocalePTBR

	chosen := widget.NewLabel("")
	chosen.Style.FontSize, chosen.Style.Bold = widget.FontTitle, true
	describe := func() {
		if cal.Selected.IsZero() {
			chosen.Text = "Nenhuma data"
			return
		}
		chosen.Text = fmt.Sprintf("%02d/%02d/%d", cal.Selected.Day(), int(cal.Selected.Month()), cal.Selected.Year())
	}
	describe()
	cal.OnChange = func(e *widget.Event) { describe(); a.say("Data: %s", chosen.Text) }

	jump := func(label string, f func()) *widget.Button {
		b := widget.NewButton(label)
		b.OnClick = func(e *widget.Event) { f(); describe() }
		return b
	}
	pick := func(t time.Time) { cal.Selected, cal.Year, cal.Month = t, t.Year(), t.Month() }
	today := jump("Hoje", func() { pick(time.Now()) })
	week := jump("+ 7 dias", func() { pick(time.Now().AddDate(0, 0, 7)) })
	clear := jump("Limpar", func() { cal.Selected = time.Time{} })
	for i, b := range []*widget.Button{today, week, clear} {
		b.ID = fmt.Sprintf("cal-shortcut-%d", i+1)
	}

	return twoColPage(
		cal,
		widget.NewColumn(24,
			widget.NewCard("Data selecionada", "", chosen),
			widget.NewCard("Atalhos", "Botões que alteram o calendário", widget.NewRow(14, today, week, clear))),
	)
}
