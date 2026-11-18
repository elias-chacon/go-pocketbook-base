package widget

import "testing"

func TestApplyMask(t *testing.T) {
	cases := []struct{ mask, digits, want string }{
		{"(00) 00000-0000", "11987654321", "(11) 98765-4321"},
		{"(00) 00000-0000", "1198", "(11) 98"},
		{"(00) 00000-0000", "", ""},
		{"000.000.000-00", "12345678901", "123.456.789-01"},
		{"000.000.000-00", "1234x5", "123.45"},
		{"##/##", "1234", "12/34"},
	}
	for _, c := range cases {
		if got := ApplyMask(c.mask, c.digits); got != c.want {
			t.Errorf("ApplyMask(%q, %q) = %q, want %q", c.mask, c.digits, got, c.want)
		}
	}
	if got := TrimToMask("00-00", "123456"); got != "1234" {
		t.Errorf("TrimToMask = %q", got)
	}
	if MaskSlots("(00) 0") != 3 {
		t.Error("MaskSlots")
	}
}

func TestNumberInputStepsClampsAndRounds(t *testing.T) {
	in := NewNumberInput(0.5, 1)
	in.Min, in.Max, in.Step = 0, 1, 0.1
	in.StepUp()
	in.StepUp()
	if in.Value != "0.7" {
		t.Errorf("after 2 steps Value = %q, want 0.7", in.Value)
	}
	for i := 0; i < 10; i++ {
		in.StepUp()
	}
	if in.Value != "1.0" {
		t.Errorf("clamped Value = %q, want 1.0", in.Value)
	}
	for i := 0; i < 20; i++ {
		in.StepDown()
	}
	if in.Value != "0.0" {
		t.Errorf("clamped Value = %q, want 0.0", in.Value)
	}
}

func TestNumberInputDisplayAndMask(t *testing.T) {
	price := NewNumberInput(19.9, 2)
	price.Prefix, price.DecimalSeparator = "R$ ", ","
	if got := price.Display(); got != "R$ 19,90" {
		t.Errorf("Display = %q", got)
	}
	phone := NewMaskedInput("(00) 00000-0000", "11987654321")
	if got := phone.Display(); got != "(11) 98765-4321" {
		t.Errorf("masked Display = %q", got)
	}
	phone.commit("(21) 3333-4444 extra digits 99")
	if phone.Value != "21333344449" { // trimmed to the 11 slots
		t.Errorf("masked Value = %q", phone.Value)
	}
}

func TestTextInputWithKeyboard(t *testing.T) {
	kb := &FakeKeyboard{Reply: "Maria"}
	in := NewInput("Nome")
	changes := &counter{}
	in.OnChange = changes.handler()
	d := testDoc(in)
	d.Keyboard = kb

	tap(d, in)
	if in.Value != "Maria" || changes.n != 1 {
		t.Fatalf("Value=%q changes=%d", in.Value, changes.n)
	}
	if kb.Last.Title != "Nome" || kb.Last.Kind != KeyboardText {
		t.Errorf("request = %+v", kb.Last)
	}

	kb.Cancel = true
	kb.Reply = "ignored"
	tap(d, in)
	if in.Value != "Maria" || changes.n != 1 {
		t.Errorf("cancel must keep the value: %q (%d changes)", in.Value, changes.n)
	}

	in.MaxLength = 3
	kb.Cancel, kb.Reply = false, "abcdef"
	tap(d, in)
	if in.Value != "abc" {
		t.Errorf("MaxLength not applied: %q", in.Value)
	}
}

func TestNumberInputSteppersAndKeyboard(t *testing.T) {
	in := NewNumberInput(5, 0)
	d := testDoc(in)
	kb := &FakeKeyboard{Reply: "42,5"}
	d.Keyboard = kb

	ads := in.adornments() // plus, minus (right to left)
	tapAt(d, ads[0].rect.Center())
	tapAt(d, ads[0].rect.Center())
	tapAt(d, ads[1].rect.Center())
	if in.Value != "6" || kb.Last.Title != "" {
		t.Errorf("steppers: Value=%q (keyboard must stay closed)", in.Value)
	}

	tapAt(d, geomPt(in.textRect().X+20, in.Bounds.Center().Y))
	if kb.Last.Kind != KeyboardNumeric || in.Value != "43" { // 42.5 rounds to 43 with 0 decimals
		t.Errorf("keyboard kind=%v Value=%q", kb.Last.Kind, in.Value)
	}

	kb.Reply = "abc"
	tapAt(d, geomPt(in.textRect().X+20, in.Bounds.Center().Y))
	if in.Value != "43" {
		t.Errorf("invalid number must be ignored, Value=%q", in.Value)
	}
}

func TestSearchInputButtons(t *testing.T) {
	in := NewSearchInput("Buscar")
	submits := &counter{}
	in.OnSubmit = submits.handler()
	d := testDoc(in)
	d.Keyboard = &FakeKeyboard{Reply: "golang"}

	tap(d, in) // typing a query submits it, like pressing OK on the keyboard
	if in.Value != "golang" || submits.n != 1 {
		t.Fatalf("Value=%q submits=%d", in.Value, submits.n)
	}
	ads := in.adornments() // magnifier, clear
	if len(ads) != 2 {
		t.Fatalf("want magnifier + clear, got %d", len(ads))
	}
	tapAt(d, ads[0].rect.Center())
	if submits.n != 2 {
		t.Errorf("magnifier must submit, submits=%d", submits.n)
	}
	tapAt(d, ads[1].rect.Center())
	if in.Value != "" || len(in.adornments()) != 1 {
		t.Errorf("clear: Value=%q adornments=%d", in.Value, len(in.adornments()))
	}
}

func TestTextAreaKeyboard(t *testing.T) {
	ta := NewTextArea("Notas", 3)
	d := testDoc(ta)
	d.Keyboard = &FakeKeyboard{Reply: "linha 1\nlinha 2"}
	tap(d, ta)
	if ta.Value != "linha 1\nlinha 2" {
		t.Errorf("Value = %q", ta.Value)
	}
	if got := len(wrapText(ta.Value, 1000, FontBody, false)); got != 2 {
		t.Errorf("lines = %d, want 2", got)
	}
}

func TestDisabledInputIgnoresTouch(t *testing.T) {
	in := NewInput("x")
	in.Disabled = true
	d := testDoc(in)
	kb := &FakeKeyboard{Reply: "nope"}
	d.Keyboard = kb
	tap(d, in)
	if in.Value != "" || kb.Last.Title != "" {
		t.Error("a disabled input must not react")
	}
}
