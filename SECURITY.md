# Security

Report vulnerabilities privately to security@almarfeld.com. Do not attach personal or customer data; reproduce with synthetic CSV fixtures.

CSV Doctor reads local data and does not evaluate formulas or launch content from cells. Optional formula prefixes are not universal protection against spreadsheet formula execution. No revocation/security scan or Authenticode signing is provided. Keep Windows, WebView2 and spreadsheet applications updated independently.

The v1 candidate applies explicit size/record/field limits, requires successful parsing before export, and refuses existing destination files. Changes should retain these guarantees and be covered by regression tests.
