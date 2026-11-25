// Package oauthconfig has the client ids of the apps registered at the providers. The ones of this
// project are built in (Default), so a person who installs the apps signs in with nothing to set up;
// oauth.json next to the accounts overrides them, for whoever registers their own app:
//
//	{"microsoft": "<client id>", "google": "<client id>", "google_secret": "<client secret>"}
//
// None of them is a secret in the sense of a password: they identify the app, which is a
// public client (a desktop application), and Google asks for its "secret" anyway.
package oauthconfig

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthcode"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthdevice"
)

// File is the name of the file.
const File = "oauth.json"

// Default is the registration of this project, built into the apps: the Microsoft one is the
// "Go-PocketBook-Base Apps" registration in Entra ID (a public client with the device code flow
// allowed, any directory and personal accounts). Google has none yet: it needs the oauth.json of
// whoever creates a Cloud project. A fork puts its own ids here.
var Default = Config{Microsoft: "a628414c-889d-4436-b259-c583bf213757"}

// Config holds the client ids ("" for a provider that is not set).
type Config struct {
	Microsoft    string `json:"microsoft"`
	Google       string `json:"google"`
	GoogleSecret string `json:"google_secret"`
}

// Load is Default with what the file of a data folder sets on top of it. A missing or broken
// file changes nothing; an id left empty in the file keeps the built-in one.
func Load(dir string) Config {
	c := Default
	data, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil {
		return c
	}
	var f Config
	if json.Unmarshal(data, &f) != nil {
		return c
	}
	if v := strings.TrimSpace(f.Microsoft); v != "" {
		c.Microsoft = v
	}
	if v := strings.TrimSpace(f.Google); v != "" {
		c.Google, c.GoogleSecret = v, strings.TrimSpace(f.GoogleSecret)
	}
	return c
}

// Sources builds the sign-in of each provider that has a client id in the data folder: Microsoft
// by device code, Google by authorization code. A provider that is not set is nil (the apps say
// how to set it). web is the network client of the platform.
func Sources(dir string, web *http.Client) (microsoft domain.Authorizer, google domain.CodeAuthorizer) {
	cfg := Load(dir)
	if cfg.Microsoft != "" {
		microsoft = oauthdevice.Client{HTTP: web, Provider: oauthdevice.Microsoft(cfg.Microsoft)}
	}
	if cfg.Google != "" {
		google = oauthcode.Client{HTTP: web, Provider: oauthcode.Google(cfg.Google, cfg.GoogleSecret)}
	}
	return microsoft, google
}
