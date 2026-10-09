# Contributing

Keep this a focused local CSV diagnosis tool. Use synthetic fixtures only. Prefer a reproducible bug and a regression test over automatic guesses about malformed data. Cell text, records and extra fields must be preserved unless a clearly described opt-in operation changes them.

Run frontend tests/build, `go vet ./...` and `go test ./...`. For Windows packaging use the Wails CLI version in go.mod and `-webview2 browser`; WebView2 must remain a separate dependency. Do not add telemetry, automatic downloads or networking to the application.
