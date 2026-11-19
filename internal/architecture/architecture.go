// Package architecture is the architecture of the repository, written as code:
// the bounded contexts, their layers and the one rule that keeps them apart. It
// has no behavior at run time; the test next to it reads every import of the
// repository and fails when the rule is broken, so the structure cannot rot.
//
// # Bounded contexts (internal/)
//
//	kernel        the shared kernel: tiny value objects every context uses
//	              (geom: Rect, Point, Insets · version: MAJOR.MINOR.BUILD).
//	              It depends on nothing.
//	toolkit       the UI toolkit, the core of the base library: gfx (vector
//	              drawing) and widget (the Document aggregate, elements, layout,
//	              touch). It knows nothing of the reader or of apps.
//	device        the e-reader as an outside system: inkview (the cgo binding)
//	              desktop (registering an icon on the home screen) and network
//	              (an HTTP client that works on the reader's Linux). The
//	              anti-corruption layer: the rest of the code never sees InkView.
//	localization  languages: i18n (pt-BR first, English fallback).
//	platform      what an app of this project is. Three layers:
//	                domain          Manifest, InstalledApp, Settings and the ports
//	                application     use cases: Runtime (run an app), FirstRun
//	                infrastructure  adapters of the ports: jsonfiles, reader
//	              plus the package platform itself, the composition root that
//	              wires them and the one import an app needs.
//	launcher      the App Launcher, the second context of the ecosystem. Same layers:
//	                domain          Tile, Grid, paging, search matching, the AppLauncher and Catalog ports
//	                application     the Home use case (list the tiles, open one) and Search (installed + available)
//	                infrastructure  processlauncher (starts an app by replacing the process)
//	                presentation    the screens, built from the toolkit's widgets
//	              It depends on the platform only for its vocabulary (InstalledApp).
//	installer     the installer for people, the third context: put the apps on a
//	              PocketBook connected to the computer (cmd/pbinstaller). Layers:
//	                domain          Reader, Package, Outcome and the ports
//	                application     the Install use case
//	                infrastructure  drives (find the reader), embedded (the apps inside
//	                                the executable), readerfs (write to the reader), locale
//	                presentation    the console conversation
//	              It does not depend on the platform: the composition root hands it
//	              the registration (platform.RegisterOnDrive).
//	accounts      the accounts of the person (an address, what it gives access to, the
//	              secrets) for the mail, tasks and calendar apps to share. Layers:
//	                domain          Account, Endpoint, Secret, presets of providers, ports
//	                application     Accounts: add, remove, hand out the credentials
//	                infrastructure  jsonaccounts (accounts.json) and securefile (the
//	                                encrypted vault of secrets)
//	              It depends on nothing but the kernel.
//	mail          the e-mail of the future mail app: messages, folders, drafts, the rules
//	              of replying and forwarding, and the MIME reader and writer. Layers:
//	                domain          Message, Folder, Draft, Address, Reply, Forward
//	                infrastructure  mime (RFC 5322 and MIME on the wire) and sqlstore (the
//	                                messages in the shared database)
//	              It depends on the kernel and, for the store, on storage.
//	calendar      the events of the future calendar app: events, recurrence, reminders,
//	              the occurrences of a range, and the iCalendar (.ics) reader and writer.
//	              Layers:
//	                domain          Event, Occurrence, Calendar, Occurrences, Expand
//	                infrastructure  icalendar (RFC 5545 on the wire) and sqlstore (the
//	                                events in the shared database)
//	              It depends on the kernel and, for the store, on storage.
//	tasks         the to-do list of the future tasks app: tasks, lists, steps, due dates,
//	              repetition and the smart views (My Day, Important, Planned). Layer:
//	                domain          Task, List, Step, View, Complete, Filter, Count
//	                infrastructure  sqlstore (the tasks in the shared database)
//	              It depends on the kernel and, for the store, on storage.
//	storage       the shared local database (SQLite, pure Go) of the productivity apps:
//	              open, migrate per module, erase an account. A single package, depending
//	              on nothing but the kernel; each context keeps its own tables in it.
//	showcase      the content of the hello app: every widget on four pages.
//
// Outside internal/: apps/ (one composition root per app, free to wire any
// context), examples/ (documentation samples, by the context they teach), tools/
// and cmd/ (development tools).
//
// # Ubiquitous language
//
//	Document      the tree of elements of one screen; the aggregate root of the toolkit
//	Element       a node of the Document (button, input, panel…); implements widget.Widget
//	Design pixel  the unit of every measure (reference width 1404), scaled by the Document
//	App           a program of the ecosystem: one folder under apps/, one <id>.app file
//	Manifest      the app's identity, in app.json (id, titles, description)
//	InstalledApp  an App that has run on this reader: manifest + version + files
//	Settings      preferences shared by every App (today the language)
//	Port          an interface the domain defines for something it needs from outside
//	Adapter       the implementation of a port, in infrastructure
//	Plugin        an App that the Launcher lists, installs and runs
//
// # The dependency rule
//
// Dependencies point inwards: infrastructure → application → domain, and every
// context may use the kernel. A context never reaches into another's insides, and
// nothing in internal/ imports apps/, examples/, tools/ or cmd/. The exact table
// is Rules below.
package architecture

import "strings"

// Rule says which internal packages the packages under From may import.
type Rule struct {
	From  string   // directory (module-relative), the package itself and everything below it
	Allow []string // directories the packages may import, each with everything below it
}

// The names of the layers, the same in every context.
const (
	domainLayer         = "/domain"
	applicationLayer    = "/application"
	infrastructureLayer = "/infrastructure"
	presentationLayer   = "/presentation"
)

// The directories the rule talks about (module-relative).
const (
	kernelDir         = "internal/kernel"
	localizationDir   = "internal/localization"
	toolkitDir        = "internal/toolkit"
	deviceDir         = "internal/device"
	platformDir       = "internal/platform"
	domainDir         = platformDir + domainLayer
	applicationDir    = platformDir + applicationLayer
	infrastructureDir = platformDir + infrastructureLayer
	showcaseDir       = "internal/showcase"
	launcherDir       = "internal/launcher"
	launcherDomain    = launcherDir + domainLayer
	launcherApp       = launcherDir + applicationLayer
	launcherInfra     = launcherDir + infrastructureLayer
	launcherUI        = launcherDir + presentationLayer
	accountsDir       = "internal/accounts"
	calendarDir       = "internal/calendar"
	calendarDomain    = calendarDir + domainLayer
	calendarInfra     = calendarDir + infrastructureLayer
	tasksDir          = "internal/tasks"
	storageDir        = "internal/storage"
	tasksDomain       = tasksDir + domainLayer
	tasksInfra        = tasksDir + infrastructureLayer
	mailDir           = "internal/mail"
	mailDomain        = mailDir + domainLayer
	mailInfra         = mailDir + infrastructureLayer
	accountsDomain    = accountsDir + domainLayer
	accountsApp       = accountsDir + applicationLayer
	accountsInfra     = accountsDir + infrastructureLayer
	installerDir      = "internal/installer"
	installerDomain   = installerDir + domainLayer
	installerApp      = installerDir + applicationLayer
	installerInfra    = installerDir + infrastructureLayer
	installerUI       = installerDir + presentationLayer
	processDir        = infrastructureDir + "/process"
	architectureDir   = "internal/architecture"
	examplesDir       = "examples"
)

// Rules is the dependency rule. The most specific From wins; packages under
// apps/, tools/ and cmd/ (the composition roots and the tools) are not limited.
var Rules = []Rule{
	{kernelDir, nil},
	{localizationDir, []string{kernelDir}},
	{toolkitDir, []string{kernelDir, toolkitDir}},
	{deviceDir, []string{kernelDir, toolkitDir, deviceDir}},
	{domainDir, []string{kernelDir, localizationDir, domainDir}},
	{applicationDir, []string{kernelDir, localizationDir, toolkitDir, domainDir, applicationDir}},
	{infrastructureDir, []string{kernelDir, localizationDir, toolkitDir, deviceDir, domainDir, applicationDir, infrastructureDir}},
	// The package platform itself: the composition root of the context.
	{platformDir, []string{kernelDir, localizationDir, platformDir}},
	{showcaseDir, []string{kernelDir, localizationDir, toolkitDir, showcaseDir}},
	// The Launcher: its own vocabulary, and the platform's only through the domain.
	{launcherDomain, []string{kernelDir, localizationDir, domainDir, launcherDomain}},
	{launcherApp, []string{kernelDir, localizationDir, domainDir, launcherDomain, launcherApp}},
	{launcherInfra, []string{kernelDir, domainDir, processDir, launcherDomain, launcherApp, launcherInfra}},
	{launcherUI, []string{kernelDir, localizationDir, toolkitDir, launcherDomain, launcherApp, launcherUI}},
	// The Installer: its own vocabulary only; the platform reaches it through the composition root.
	{installerDomain, []string{kernelDir, installerDomain}},
	{installerApp, []string{kernelDir, installerDomain, installerApp}},
	{installerInfra, []string{kernelDir, installerDomain, installerInfra}},
	{installerUI, []string{kernelDir, localizationDir, installerDomain, installerApp, installerUI}},
	// The Accounts: its own vocabulary only.
	{accountsDomain, []string{kernelDir, accountsDomain}},
	{accountsApp, []string{kernelDir, accountsDomain, accountsApp}},
	{accountsInfra, []string{kernelDir, accountsDomain, accountsInfra}},
	// The Mail: its own vocabulary only.
	{mailDomain, []string{kernelDir, mailDomain}},
	{mailInfra, []string{kernelDir, mailDomain, mailInfra, storageDir}}, // sqlstore keeps the messages in the shared database
	// The Calendar: its own vocabulary only.
	{calendarDomain, []string{kernelDir, calendarDomain}},
	{calendarInfra, []string{kernelDir, calendarDomain, calendarInfra, storageDir}}, // sqlstore keeps the events in the shared database
	// The Tasks: its own vocabulary only.
	{tasksDomain, []string{kernelDir, tasksDomain}},
	{tasksInfra, []string{kernelDir, tasksDomain, tasksInfra, storageDir}}, // sqlstore keeps the tasks in the shared database
	{storageDir, []string{kernelDir}},
	{architectureDir, nil},
	{examplesDir, []string{kernelDir, localizationDir, toolkitDir, showcaseDir, examplesDir}},
}

// Contexts are the folders allowed directly under internal/. A new one must be
// added here (and to the documentation above) on purpose.
var Contexts = []string{"architecture", "accounts", "calendar", "device", "installer", "kernel", "launcher", "localization", "mail", "platform", "showcase", "storage", "tasks", "toolkit"}

// within reports whether dir is base or a folder below it.
func within(dir, base string) bool { return dir == base || strings.HasPrefix(dir, base+"/") }

// RuleFor returns the rule that applies to the package in dir, or false when
// the package is not limited.
func RuleFor(dir string) (Rule, bool) {
	best, found := Rule{}, false
	for _, r := range Rules {
		if within(dir, r.From) && (!found || len(r.From) > len(best.From)) {
			best, found = r, true
		}
	}
	return best, found
}

// Allowed reports whether the package in fromDir may import the package in
// toDir (both module-relative directories of this module's packages).
func Allowed(fromDir, toDir string) bool {
	r, limited := RuleFor(fromDir)
	if !limited {
		return true
	}
	if within(toDir, fromDir) {
		return true // a package may use its own sub-packages
	}
	for _, a := range r.Allow {
		if within(toDir, a) {
			return true
		}
	}
	return false
}
