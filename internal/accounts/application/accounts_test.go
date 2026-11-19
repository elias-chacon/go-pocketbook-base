package application

import (
	"errors"
	"sort"
	"testing"

	"go-pocketbook-base/internal/accounts/domain"
)

type store struct {
	m       map[string]domain.Account
	saveErr error
	listErr error
	delErr  error
}

func newStore() *store { return &store{m: map[string]domain.Account{}} }

func (s *store) List() ([]domain.Account, error) {
	var out []domain.Account
	for _, a := range s.m {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, s.listErr
}
func (s *store) Get(id string) (domain.Account, error) {
	if a, ok := s.m[id]; ok {
		return a, nil
	}
	return domain.Account{}, domain.ErrNotFound
}
func (s *store) Save(a domain.Account) error {
	if s.saveErr != nil {
		return s.saveErr
	}
	s.m[a.ID] = a
	return nil
}
func (s *store) Delete(id string) error {
	if s.delErr != nil {
		return s.delErr
	}
	delete(s.m, id)
	return nil
}

type vault struct {
	m      map[string]domain.Secret
	putErr error
	getErr error
	delErr error
}

func newVault() *vault { return &vault{m: map[string]domain.Secret{}} }

func (v *vault) Put(id string, s domain.Secret) error {
	if v.putErr != nil {
		return v.putErr
	}
	v.m[id] = s
	return nil
}
func (v *vault) Get(id string) (domain.Secret, error) {
	if v.getErr != nil {
		return domain.Secret{}, v.getErr
	}
	if s, ok := v.m[id]; ok {
		return s, nil
	}
	return domain.Secret{}, domain.ErrNotFound
}
func (v *vault) Delete(id string) error {
	if v.delErr != nil {
		return v.delErr
	}
	if _, ok := v.m[id]; !ok {
		return domain.ErrNotFound
	}
	delete(v.m, id)
	return nil
}

type eraser struct {
	erased []string
	err    error
}

func (e *eraser) Erase(id string) error { e.erased = append(e.erased, id); return e.err }

func account(name, email string) domain.Account {
	p, _ := domain.PresetFor(email)
	return p.Account(name, email)
}

func TestAddKeepsTheAccountAndItsSecret(t *testing.T) {
	st, vt := newStore(), newVault()
	u := Accounts{Store: st, Vault: vt}
	a := account("Pessoal", "ana@icloud.com")

	if err := u.Add(a, domain.Secret{Password: "app-pass"}); err != nil {
		t.Fatal(err)
	}
	got, secret, err := u.Credentials("pessoal")
	if err != nil || got.Email != "ana@icloud.com" || secret.Password != "app-pass" {
		t.Fatalf("%+v %v %v", got, secret, err)
	}
	if err := u.Add(a, domain.Secret{}); err != domain.ErrExists {
		t.Errorf("an id is taken only once: %v", err)
	}
	if err := u.Add(domain.Account{ID: "x"}, domain.Secret{}); err == nil {
		t.Error("an invalid account")
	}
}

func TestAddLeavesNothingHalfDone(t *testing.T) {
	st, vt := newStore(), newVault()
	u := Accounts{Store: st, Vault: vt}
	vt.putErr = errors.New("vault locked")
	if err := u.Add(account("A", "a@icloud.com"), domain.Secret{}); err == nil || len(st.m) != 0 {
		t.Errorf("no secret, no account: %v %v", err, st.m)
	}

	vt.putErr = nil
	st.saveErr = errors.New("disk full")
	if err := u.Add(account("B", "b@icloud.com"), domain.Secret{Password: "p"}); err == nil || len(vt.m) != 0 {
		t.Errorf("no account, no secret: %v %v", err, vt.m)
	}

	st.saveErr = nil
	st.listErr = nil
	broken := &brokenGet{store: st}
	if err := (Accounts{Store: broken, Vault: vt}).Add(account("C", "c@icloud.com"), domain.Secret{}); err == nil {
		t.Error("a store that fails to look the id up")
	}
}

type brokenGet struct{ *store }

func (brokenGet) Get(string) (domain.Account, error) { return domain.Account{}, errors.New("io error") }

func TestRemoveDeletesTheSecretTheDataAndTheAccountInThatOrder(t *testing.T) {
	st, vt, er := newStore(), newVault(), &eraser{}
	u := Accounts{Store: st, Vault: vt, Eraser: er}
	u.Add(account("A", "a@icloud.com"), domain.Secret{Password: "p"})

	if err := u.Remove("a"); err != nil {
		t.Fatal(err)
	}
	if len(st.m) != 0 || len(vt.m) != 0 || len(er.erased) != 1 || er.erased[0] != "a" {
		t.Errorf("everything is gone: %v %v %v", st.m, vt.m, er.erased)
	}
	if err := u.Remove("a"); err != domain.ErrNotFound {
		t.Errorf("removing what is not there: %v", err)
	}
}

func TestAFailedRemovalCanBeRetriedBecauseTheAccountStaysUntilTheEnd(t *testing.T) {
	st, vt, er := newStore(), newVault(), &eraser{err: errors.New("db busy")}
	u := Accounts{Store: st, Vault: vt, Eraser: er}
	u.Add(account("A", "a@icloud.com"), domain.Secret{Password: "p"})

	if err := u.Remove("a"); err == nil {
		t.Fatal("the eraser failed")
	}
	if len(st.m) != 1 {
		t.Error("the account stays so the removal can be retried")
	}
	er.err = nil
	if err := u.Remove("a"); err != nil || len(st.m) != 0 {
		t.Errorf("the retry finishes it: %v", err)
	}

	st2, vt2 := newStore(), newVault()
	u2 := Accounts{Store: st2, Vault: vt2}
	u2.Add(account("B", "b@icloud.com"), domain.Secret{})
	vt2.delErr = errors.New("locked")
	if err := u2.Remove("b"); err == nil || len(st2.m) != 1 {
		t.Errorf("a vault that cannot delete: %v", err)
	}
	vt2.delErr = nil
	st2.delErr = errors.New("read-only")
	if err := u2.Remove("b"); err == nil {
		t.Error("a store that cannot delete")
	}
}

func TestListFiltersByServiceAndUpdateReplacesTheSettings(t *testing.T) {
	st, vt := newStore(), newVault()
	u := Accounts{Store: st, Vault: vt}
	u.Add(account("Z mail", "z@icloud.com"), domain.Secret{})
	custom, _ := domain.PresetByID(domain.Custom)
	cal := custom.Account("Agenda", "a@example.org")
	cal.Services = map[domain.Service][]domain.Endpoint{domain.Calendar: {{Kind: domain.CalDAV, URL: "https://cal.example.org"}}}
	if err := u.Add(cal, domain.Secret{Password: "x"}); err != nil {
		t.Fatal(err)
	}

	all, _ := u.List("")
	mail, _ := u.List(domain.Mail)
	if len(all) != 2 || len(mail) != 1 || mail[0].ID != "z-mail" {
		t.Errorf("all=%d mail=%v", len(all), mail)
	}
	st.listErr = errors.New("io")
	if _, err := u.List(""); err == nil {
		t.Error("a store that fails to list")
	}

	cal.Name = "Agenda do trabalho"
	if err := u.Update(cal); err != nil || st.m["agenda"].Name != "Agenda do trabalho" {
		t.Errorf("%v", err)
	}
	if err := u.Update(domain.Account{ID: "ghost"}); err == nil {
		t.Error("an invalid account")
	}
	ghost := account("Ghost", "g@icloud.com")
	if err := u.Update(ghost); err != domain.ErrNotFound {
		t.Errorf("an account that does not exist: %v", err)
	}
}

func TestRefreshKeepsNewTokensAndCredentialsReportAMissingSecret(t *testing.T) {
	st, vt := newStore(), newVault()
	u := Accounts{Store: st, Vault: vt}
	out, _ := domain.PresetByID("outlook")
	u.Add(out.Account("MS", "m@outlook.com"), domain.Secret{AccessToken: "old", RefreshToken: "r"})

	if err := u.Refresh("ms", domain.Secret{AccessToken: "new", RefreshToken: "r", ExpiresAt: 99}); err != nil {
		t.Fatal(err)
	}
	if _, s, _ := u.Credentials("ms"); s.AccessToken != "new" {
		t.Error("the new token is kept")
	}
	if err := u.Refresh("nobody", domain.Secret{}); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
	if _, _, err := u.Credentials("nobody"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
	vt.getErr = domain.ErrLocked
	if _, _, err := u.Credentials("ms"); err != domain.ErrLocked {
		t.Errorf("a locked vault: %v", err)
	}
}
