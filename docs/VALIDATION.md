# Candidate validation

Validated on Windows x64 with Node 22.23.3, Go 1.27.0 and Wails 2.16.0.

- Frontend: `npm ci`, 3 tests, TypeScript checking and Vite production build pass.
- Go: `go vet ./...` and 50 passing test cases/subtests, including round-trip fuzz seeds. Engine statement coverage: 94.0%.
- A 20-second real fuzz run completed 850,093 executions without a cell round-trip failure.
- Production build: `wails build -clean -s -webview2 browser` passes.
- Real Windows app: startup, synthetic CSV diagnosis, opt-in header/padding review and before/after comparison verified.
- Native Open/Save dialogs: exported a new UTF-8 BOM/CRLF copy and adjacent report; reopened it successfully. All six records, leading-zero/long integer text, Unicode and embedded LF survived. The original SHA-256 remained equal to the report source hash. Canceling export created no file.
- Fresh portable extraction launches and renders diagnostics and reviewed changes correctly. The screenshot is a direct application capture using synthetic data, without a cursor or compositing.

Automated tests additionally cover all four delimiters, UTF-8/BOM/UTF-16/Windows-1252, invalid encodings, malformed quotes, Unicode, multiline cells, uneven rows, formula protection, header collisions, output overwrite refusal and stale review tokens. Reports and reopened copies agree on blank-record counts.

Not yet independently verified: drag-and-drop using Explorer, behavior on a machine without WebView2, taskbar/Alt+Tab icon inspection and a fully network-isolated environment. No host dependencies or network/security settings were removed or changed to simulate those conditions.

This is a functional candidate, not a published release. Binaries are not Authenticode signed. No website product page or WinGet package is created.
