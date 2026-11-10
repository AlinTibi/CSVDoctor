# Validation

Validated on Windows x64 with Node 22.23.3, Go 1.27.0 and Wails 2.16.0.

- Frontend: `npm ci`, 3 tests, TypeScript checking and Vite production build pass.
- Go: `go vet ./...` and 50 passing test cases/subtests, including round-trip fuzz seeds. Engine statement coverage: 94.0%.
- A 20-second real fuzz run completed 850,093 executions without a cell round-trip failure.
- Production build: `wails build -clean -s -webview2 browser` passes.
- Real Windows app: startup, synthetic CSV diagnosis, opt-in header/padding review and before/after comparison verified.
- Native Open/Save dialogs: exported a new UTF-8 BOM/CRLF copy and adjacent report; reopened it successfully. All six records, leading-zero/long integer text, Unicode and embedded LF survived. The original SHA-256 remained equal to the report source hash. Canceling export created no file.
- Fresh portable extraction launches and renders diagnostics and reviewed changes correctly. The screenshot is a direct application capture using synthetic data, without a cursor or compositing.

Automated tests additionally cover all four delimiters, UTF-8/BOM/UTF-16/Windows-1252, invalid encodings, malformed quotes, Unicode, multiline cells, uneven rows, formula protection, header collisions, output overwrite refusal and stale review tokens. Reports and reopened copies agree on blank-record counts.

Completed manual validation:

- Explorer drag/drop: PASS. CSV, TSV and TXT opened successfully; an unsupported BIN file was rejected clearly. Replacing the loaded file left no stale data or application freeze. These Explorer-to-application checks were performed and confirmed by the tester.
- Missing WebView2 in Windows Sandbox: PASS. With WebView2 registration unavailable to the runtime loader, the packaged executable showed a controlled message identifying Microsoft Edge WebView2 Runtime and its official Microsoft download page, then exited cleanly. Nothing was downloaded or installed automatically. The host runtime was not removed or changed; this tested runtime unavailability rather than a physical uninstall.
- Clean extraction / spaced path / non-project working directory: PASS. RC.6 launched from a freshly extracted folder containing spaces with a working directory outside the project.
- GUI smoke test: PASS. Opened a synthetic CSV, loaded diagnostics, reviewed opt-in header/padding changes, exported a repaired copy and JSON report, then reopened the copy successfully.
- Original input byte-for-byte preservation: PASS. The input SHA-256 was unchanged after export; independent comparison confirmed original data cell text was preserved, including leading zeros, long numeric identifiers, formula text and multiline content.

Not yet independently verified: taskbar/Alt+Tab icon inspection and a fully network-isolated environment. No host network/security settings were changed to simulate those conditions.

Binaries are not Authenticode signed. No website product page or WinGet package is created.
