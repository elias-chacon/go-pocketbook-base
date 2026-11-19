package securefile

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"go-pocketbook-base/internal/accounts/domain"
)

func init() { Iterations = 50 } // keep the tests quick

func TestPBKDF2MatchesTheKnownVectorsOfRFC7914(t *testing.T) {
	cases := []struct {
		iterations int
		want       string
	}{
		{1, "120fb6cffcf8b32c43e7225256c4f837a86548c92ccc35480805987cb70be17b"},
		{2, "ae4d0c95af6b46d32d0adff928f06dd02a303f8ef3c251dfd6e2d85a95474c43"},
		{4096, "c5e478d59288c841aa530db6845c4c8d962893a001ce4e11a4963873aa98134a"},
	}
	for _, c := range cases {
		if got := hex.EncodeToString(pbkdf2([]byte("password"), []byte("salt"), c.iterations, 32)); got != c.want {
			t.Errorf("c=%d: %s", c.iterations, got)
		}
	}
	if got := pbkdf2([]byte("p"), []byte("s"), 1, 50); len(got) != 50 {
		t.Errorf("more than one block: %d bytes", len(got))
	}
}

func TestSecretsRoundTripAndAreNotInThePlainFile(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	if _, err := v.Get("a"); err != domain.ErrNotFound {
		t.Errorf("an empty vault: %v", err)
	}
	if err := v.Put("a", domain.Secret{Password: "hunter2", AccessToken: "tok-123"}); err != nil {
		t.Fatal(err)
	}
	v.Put("b", domain.Secret{RefreshToken: "ref-456"})

	// a new Vault on the same folder (a new run of the app) reads them back
	got, err := New(dir).Get("a")
	if err != nil || got.Password != "hunter2" || got.AccessToken != "tok-123" {
		t.Fatalf("%v %v", got, err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "secrets.bin"))
	for _, secret := range []string{"hunter2", "tok-123", "ref-456", "password"} {
		if bytes.Contains(raw, []byte(secret)) {
			t.Errorf("%q is in the file in plain text", secret)
		}
	}
	if !bytes.HasPrefix(raw, []byte("PBGS1")) {
		t.Error("the file starts with the magic")
	}

	if err := v.Delete("a"); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Get("a"); err != domain.ErrNotFound {
		t.Error("deleted")
	}
	if err := v.Delete("a"); err != domain.ErrNotFound {
		t.Errorf("deleting twice: %v", err)
	}
	if s, _ := v.Get("b"); s.RefreshToken != "ref-456" {
		t.Error("the other secret stays")
	}
}

func TestEverySaveUsesAFreshNonce(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	v.Put("a", domain.Secret{Password: "x"})
	first, _ := os.ReadFile(filepath.Join(dir, "secrets.bin"))
	v.Put("a", domain.Secret{Password: "x"})
	second, _ := os.ReadFile(filepath.Join(dir, "secrets.bin"))
	if bytes.Equal(first, second) {
		t.Error("the same content sealed twice must not give the same bytes")
	}
}

func TestWithoutTheDeviceKeyOrWithAChangedFileTheVaultStaysClosed(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	v.Put("a", domain.Secret{Password: "x"})

	raw, _ := os.ReadFile(filepath.Join(dir, "secrets.bin"))
	raw[len(raw)-1] ^= 0xff // a flipped bit
	os.WriteFile(filepath.Join(dir, "secrets.bin"), raw, 0o600)
	if _, err := New(dir).Get("a"); err != domain.ErrLocked {
		t.Errorf("a tampered file is refused: %v", err)
	}

	raw[len(raw)-1] ^= 0xff
	os.WriteFile(filepath.Join(dir, "secrets.bin"), raw, 0o600)
	os.Remove(filepath.Join(dir, "device.key"))
	if _, err := New(dir).Get("a"); err != domain.ErrLocked {
		t.Errorf("without its key the secrets stay closed: %v", err)
	}
	if err := New(dir).Put("a", domain.Secret{}); err != domain.ErrLocked {
		t.Errorf("and it does not overwrite them with a new key: %v", err)
	}
}

func TestAPINLocksTheVaultAndOnlyTheRightPINOpensIt(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	v.Put("a", domain.Secret{Password: "hunter2"})
	if v.Locked() {
		t.Error("a vault without a PIN is open")
	}
	if err := v.SetPIN("1234"); err != nil {
		t.Fatal(err)
	}
	if s, err := v.Get("a"); err != nil || s.Password != "hunter2" {
		t.Errorf("still open right after setting it: %v %v", s, err)
	}

	next := New(dir) // a new run of the app
	if !next.Locked() {
		t.Error("a vault with a PIN starts locked")
	}
	if _, err := next.Get("a"); err != domain.ErrLocked {
		t.Errorf("locked: %v", err)
	}
	if err := next.Put("b", domain.Secret{}); err != domain.ErrLocked {
		t.Errorf("a locked vault takes no writes: %v", err)
	}
	if err := next.Unlock("0000"); err != domain.ErrLocked || !next.Locked() {
		t.Errorf("a wrong PIN: %v", err)
	}
	if err := next.Unlock("1234"); err != nil || next.Locked() {
		t.Fatalf("the right PIN: %v", err)
	}
	if s, _ := next.Get("a"); s.Password != "hunter2" {
		t.Error("readable after unlocking")
	}
	if err := next.Put("b", domain.Secret{Password: "y"}); err != nil {
		t.Errorf("writable after unlocking: %v", err)
	}
}

func TestChangingAndClearingThePIN(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	v.Put("a", domain.Secret{Password: "p"})
	v.SetPIN("1111")
	if err := v.SetPIN("2222"); err != nil {
		t.Fatal(err)
	}
	next := New(dir)
	if next.Unlock("1111") == nil {
		t.Error("the old PIN no longer works")
	}
	if next.Unlock("2222") != nil {
		t.Error("the new one does")
	}
	if err := next.ClearPIN(); err != nil {
		t.Fatal(err)
	}
	if s, err := New(dir).Get("a"); err != nil || s.Password != "p" {
		t.Errorf("without a PIN again: %v %v", s, err)
	}
	if err := v.SetPIN(""); err == nil {
		t.Error("an empty PIN is refused")
	}
	if err := New(dir).Unlock("whatever"); err != nil {
		t.Errorf("nothing to unlock: %v", err)
	}
}

func TestSettingOrClearingThePINNeedsAnOpenVault(t *testing.T) {
	dir := t.TempDir()
	v := New(dir)
	v.Put("a", domain.Secret{Password: "p"})
	v.SetPIN("1234")
	locked := New(dir)
	if err := locked.SetPIN("9999"); err != domain.ErrLocked {
		t.Errorf("%v", err)
	}
	if err := locked.ClearPIN(); err != domain.ErrLocked {
		t.Errorf("%v", err)
	}
}

func TestAFileThatIsNotAVaultIsReported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.bin")
	for name, content := range map[string][]byte{
		"too short":   []byte("PBGS"),
		"wrong magic": append([]byte("XXXXX\x00"), make([]byte, 40)...),
		"wrong mode":  append([]byte("PBGS1\x07"), make([]byte, 40)...),
		"no nonce":    append([]byte("PBGS1\x00"), make([]byte, 16)...),
	} {
		os.WriteFile(path, content, 0o600)
		if _, err := New(dir).Get("a"); err == nil {
			t.Errorf("%s must be an error", name)
		}
		if New(dir).Locked() {
			t.Errorf("%s is not a locked vault", name)
		}
	}
}

func TestAStorageThatCannotBeWrittenIsReported(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	if err := New(filepath.Join(file, "sub")).Put("a", domain.Secret{}); err == nil {
		t.Error("a folder under a file cannot be made")
	}
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "secrets.bin", "x"), 0o755) // the target is a folder
	if err := New(dir).Put("a", domain.Secret{}); err == nil {
		t.Error("a target that cannot be replaced")
	}
	if _, err := os.Stat(filepath.Join(dir, "secrets.bin.tmp")); err == nil {
		t.Error("no temporary file is left")
	}
}
