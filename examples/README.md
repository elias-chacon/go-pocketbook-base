# examples

Code samples for the documentation (`docs/en`, `docs/pt-br`). Every sample compiles and is covered by tests, and
`go run ./tools/docsync` copies them into the pages, so the documentation never goes stale.

They are organized by the chapter (that is, the context) each one teaches:

| Folder | Chapter | What |
|---|---|---|
| `components/` | 6 | one file per component: `button.go`, `table.go`, `modal.go`… |
| `recipes/` | 8 | complete screens: login, settings, catalog, order, date picker, navigation, hello |
| `custom/` | 9 | writing your own element |
| `translation/` | 12 | i18n with JSON files |
| `harness/` | 10 | what the samples and their tests share: `Screen`, `SavePNG` and the test kit (`NewDoc`, `Tap`, `Inked`…) |

They may use the toolkit, the kernel, localization and the showcase, never the device (see
[Architecture](../docs/en/14-architecture.md)). The app template is `apps/hello`.
