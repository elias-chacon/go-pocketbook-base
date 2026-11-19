// Package application holds the use cases of the Accounts context: add an
// account with its secret, remove it with everything it left behind, and hand the
// credentials to the app that needs them.
package application

import (
	"errors"
	"fmt"

	"go-pocketbook-base/internal/accounts/domain"
)

// Accounts is the use case of managing accounts.
type Accounts struct {
	Store  domain.AccountStore
	Vault  domain.Vault
	Eraser domain.Eraser // optional: removes an account's cached data
}

// Add validates an account and keeps it with its secret. The ID must be free. If
// the secret cannot be kept the account is not left half-added.
func (u Accounts) Add(a domain.Account, s domain.Secret) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if _, err := u.Store.Get(a.ID); err == nil {
		return domain.ErrExists
	} else if !errors.Is(err, domain.ErrNotFound) {
		return err
	}
	if err := u.Vault.Put(a.ID, s); err != nil {
		return fmt.Errorf("accounts: keeping the secret: %w", err)
	}
	if err := u.Store.Save(a); err != nil {
		u.Vault.Delete(a.ID) // do not leave a secret without an account
		return err
	}
	return nil
}

// Update replaces an account's settings (not its secret).
func (u Accounts) Update(a domain.Account) error {
	if err := a.Validate(); err != nil {
		return err
	}
	if _, err := u.Store.Get(a.ID); err != nil {
		return err
	}
	return u.Store.Save(a)
}

// Remove deletes an account, its secret and its cached data. Every step is tried;
// the first failure is returned, so a half-removed account can be removed again.
func (u Accounts) Remove(id string) error {
	if _, err := u.Store.Get(id); err != nil {
		return err
	}
	var first error
	keep := func(err error) {
		if err != nil && !errors.Is(err, domain.ErrNotFound) && first == nil {
			first = err
		}
	}
	keep(u.Vault.Delete(id))
	if u.Eraser != nil {
		keep(u.Eraser.Erase(id))
	}
	if first == nil {
		keep(u.Store.Delete(id)) // the account goes last: while it exists, Remove can be retried
	}
	return first
}

// List returns the accounts that provide a service ("" for all), by name.
func (u Accounts) List(service domain.Service) ([]domain.Account, error) {
	all, err := u.Store.List()
	if err != nil {
		return nil, err
	}
	var out []domain.Account
	for _, a := range all {
		if service == "" || a.Has(service) {
			out = append(out, a)
		}
	}
	return out, nil
}

// Credentials returns an account with its secret, for an app about to connect.
func (u Accounts) Credentials(id string) (domain.Account, domain.Secret, error) {
	a, err := u.Store.Get(id)
	if err != nil {
		return domain.Account{}, domain.Secret{}, err
	}
	s, err := u.Vault.Get(id)
	if err != nil {
		return domain.Account{}, domain.Secret{}, err
	}
	return a, s, nil
}

// Refresh keeps new OAuth2 tokens of an account (after the access token expired
// and was renewed).
func (u Accounts) Refresh(id string, s domain.Secret) error {
	if _, err := u.Store.Get(id); err != nil {
		return err
	}
	return u.Vault.Put(id, s)
}
