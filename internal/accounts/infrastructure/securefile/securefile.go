// Package securefile is the domain.Vault of the reader: the secrets of the
// accounts in one encrypted file (applications/.pbgo/secrets.bin), AES-256-GCM.
//
// What it protects, honestly:
//
//   - With a PIN (SetPIN) the key is derived from the PIN (PBKDF2-HMAC-SHA-256):
//     the file is useless without it, and the app asks for it (Unlock).
//   - Without a PIN the key is a random file next to the vault (device.key): it
//     only protects against reading the secrets by accident or copying secrets.bin
//     alone. Anyone who can read both files can open the vault: the reader has no
//     keystore. The apps say so on the screen.
package securefile

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"go-pocketbook-base/internal/accounts/domain"
)

const (
	magic         = "PBGS1"
	modeDevice    = byte(0)
	modePIN       = byte(1)
	saltSize      = 16
	keySize       = 32
	headerSize    = len(magic) + 1 + saltSize
	vaultFile     = "secrets.bin"
	deviceKeyFile = "device.key"
)

// Iterations is the cost of deriving a key from a PIN. It is a variable so tests
// can lower it; the default takes a fraction of a second on the reader.
var Iterations = 20000

// ErrCorrupt is returned for a vault file that is not one of ours.
var ErrCorrupt = errors.New("securefile: the vault file is damaged")

// Vault implements domain.Vault.
type Vault struct {
	dir string

	mu  sync.Mutex
	key []byte // the PIN's key once unlocked
}

// New returns the vault kept in dir (applications/.pbgo of the reader).
func New(dir string) *Vault { return &Vault{dir: dir} }

func (v *Vault) path() string { return filepath.Join(v.dir, vaultFile) }

// state is what the file says about itself.
type state struct {
	mode byte
	salt []byte
	data []byte // nonce + ciphertext
	new  bool   // there is no file yet
}

func (v *Vault) readState() (state, error) {
	raw, err := os.ReadFile(v.path())
	if os.IsNotExist(err) {
		return state{mode: modeDevice, new: true}, nil
	}
	if err != nil {
		return state{}, err
	}
	if len(raw) < headerSize || !bytes.HasPrefix(raw, []byte(magic)) {
		return state{}, ErrCorrupt
	}
	mode := raw[len(magic)]
	if mode != modeDevice && mode != modePIN {
		return state{}, ErrCorrupt
	}
	return state{mode: mode, salt: raw[len(magic)+1 : headerSize], data: raw[headerSize:]}, nil
}

// keyFor is the key that opens (or seals) a vault in this state.
func (v *Vault) keyFor(st state, create bool) ([]byte, error) {
	if st.mode == modePIN {
		if v.key == nil {
			return nil, domain.ErrLocked
		}
		return v.key, nil
	}
	return v.deviceKey(create)
}

func (v *Vault) deviceKey(create bool) ([]byte, error) {
	p := filepath.Join(v.dir, deviceKeyFile)
	if key, err := os.ReadFile(p); err == nil && len(key) == keySize {
		return key, nil
	}
	if !create {
		return nil, domain.ErrLocked // the secrets are there but their key is not
	}
	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(v.dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(p, key, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

func (v *Vault) open(st state) (map[string]domain.Secret, error) {
	secrets := map[string]domain.Secret{}
	if st.new {
		return secrets, nil
	}
	key, err := v.keyFor(st, false)
	if err != nil {
		return nil, err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(st.data) < gcm.NonceSize() {
		return nil, ErrCorrupt
	}
	nonce, sealed := st.data[:gcm.NonceSize()], st.data[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, sealed, append([]byte(magic), st.mode))
	if err != nil {
		return nil, domain.ErrLocked // a wrong key, or a file that was changed
	}
	if err := json.Unmarshal(plain, &secrets); err != nil {
		return nil, ErrCorrupt
	}
	return secrets, nil
}

func (v *Vault) seal(secrets map[string]domain.Secret, mode byte, salt, key []byte) error {
	plain, err := json.Marshal(secrets)
	if err != nil {
		return err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}
	var out bytes.Buffer
	out.WriteString(magic)
	out.WriteByte(mode)
	out.Write(salt)
	out.Write(nonce)
	out.Write(gcm.Seal(nil, nonce, plain, append([]byte(magic), mode)))

	if err := os.MkdirAll(v.dir, 0o755); err != nil {
		return err
	}
	tmp := v.path() + ".tmp"
	if err := os.WriteFile(tmp, out.Bytes(), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, v.path()); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// update opens the vault, lets change edit the secrets and seals it again.
func (v *Vault) update(change func(map[string]domain.Secret) error) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	st, err := v.readState()
	if err != nil {
		return err
	}
	secrets, err := v.open(st)
	if err != nil {
		return err
	}
	if err := change(secrets); err != nil {
		return err
	}
	key, err := v.keyFor(st, true)
	if err != nil {
		return err
	}
	salt := st.salt
	if st.new {
		salt = make([]byte, saltSize)
	}
	return v.seal(secrets, st.mode, salt, key)
}

// Put implements domain.Vault.
func (v *Vault) Put(accountID string, s domain.Secret) error {
	return v.update(func(m map[string]domain.Secret) error { m[accountID] = s; return nil })
}

// Get implements domain.Vault: ErrNotFound, or ErrLocked when a PIN is needed.
func (v *Vault) Get(accountID string) (domain.Secret, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	st, err := v.readState()
	if err != nil {
		return domain.Secret{}, err
	}
	secrets, err := v.open(st)
	if err != nil {
		return domain.Secret{}, err
	}
	s, ok := secrets[accountID]
	if !ok {
		return domain.Secret{}, domain.ErrNotFound
	}
	return s, nil
}

// Delete implements domain.Vault.
func (v *Vault) Delete(accountID string) error {
	return v.update(func(m map[string]domain.Secret) error {
		if _, ok := m[accountID]; !ok {
			return domain.ErrNotFound
		}
		delete(m, accountID)
		return nil
	})
}

// Locked reports whether the vault needs its PIN before it can be read.
func (v *Vault) Locked() bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	st, err := v.readState()
	return err == nil && st.mode == modePIN && v.key == nil
}

// Unlock opens a vault that has a PIN. A wrong PIN is ErrLocked.
func (v *Vault) Unlock(pin string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	st, err := v.readState()
	if err != nil {
		return err
	}
	if st.mode != modePIN {
		return nil // nothing to unlock
	}
	v.key = pbkdf2([]byte(pin), st.salt, Iterations, keySize)
	if _, err := v.open(st); err != nil {
		v.key = nil
		return err
	}
	return nil
}

// SetPIN protects the vault with a PIN (or changes it): the secrets are sealed
// again with a key derived from it. The vault must be open.
func (v *Vault) SetPIN(pin string) error {
	if pin == "" {
		return fmt.Errorf("securefile: the PIN cannot be empty (use ClearPIN to remove it)")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	st, err := v.readState()
	if err != nil {
		return err
	}
	secrets, err := v.open(st)
	if err != nil {
		return err
	}
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return err
	}
	key := pbkdf2([]byte(pin), salt, Iterations, keySize)
	if err := v.seal(secrets, modePIN, salt, key); err != nil {
		return err
	}
	v.key = key
	return nil
}

// ClearPIN removes the PIN: the secrets are sealed with the device key again.
// The vault must be open.
func (v *Vault) ClearPIN() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	st, err := v.readState()
	if err != nil {
		return err
	}
	secrets, err := v.open(st)
	if err != nil {
		return err
	}
	key, err := v.deviceKey(true)
	if err != nil {
		return err
	}
	if err := v.seal(secrets, modeDevice, make([]byte, saltSize), key); err != nil {
		return err
	}
	v.key = nil
	return nil
}
