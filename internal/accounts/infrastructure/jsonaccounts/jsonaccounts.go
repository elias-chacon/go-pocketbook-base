// Package jsonaccounts keeps the accounts (not their secrets) in a JSON file
// (applications/.pbgo/accounts.json): the domain.AccountStore of the reader.
package jsonaccounts

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"go-pocketbook-base/internal/accounts/domain"
)

// Store implements domain.AccountStore.
type Store struct {
	Path string
	mu   sync.Mutex
}

// New returns the store kept in dir (applications/.pbgo of the reader).
func New(dir string) *Store { return &Store{Path: filepath.Join(dir, "accounts.json")} }

type file struct {
	Accounts []domain.Account `json:"accounts"`
}

func (s *Store) read() ([]domain.Account, error) {
	data, err := os.ReadFile(s.Path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Accounts, nil
}

func (s *Store) write(accounts []domain.Account) error {
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Name < accounts[j].Name })
	data, err := json.MarshalIndent(file{Accounts: accounts}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// List implements domain.AccountStore, by name.
func (s *Store) List() ([]domain.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.read()
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Name < accounts[j].Name })
	return accounts, err
}

// Get implements domain.AccountStore.
func (s *Store) Get(id string) (domain.Account, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.read()
	if err != nil {
		return domain.Account{}, err
	}
	for _, a := range accounts {
		if a.ID == id {
			return a, nil
		}
	}
	return domain.Account{}, domain.ErrNotFound
}

// Save implements domain.AccountStore: it creates the account or replaces the one with the same ID.
func (s *Store) Save(a domain.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.read()
	if err != nil {
		return err
	}
	replaced := false
	for i := range accounts {
		if accounts[i].ID == a.ID {
			accounts[i], replaced = a, true
		}
	}
	if !replaced {
		accounts = append(accounts, a)
	}
	return s.write(accounts)
}

// Delete implements domain.AccountStore.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	accounts, err := s.read()
	if err != nil {
		return err
	}
	for i := range accounts {
		if accounts[i].ID == id {
			return s.write(append(accounts[:i], accounts[i+1:]...))
		}
	}
	return domain.ErrNotFound
}
