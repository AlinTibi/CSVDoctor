param([string]$Version = '1.0.0-rc.1')
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^\d+\.\d+\.\d+-rc\.\d+$') { throw 'Use a candidate version such as 1.0.0-rc.1' }
$ProjectRoot = Split-Path $PSScriptRoot -Parent
$ArtifactRoot = Join-Path $ProjectRoot 'artifacts'
New-Item -ItemType Directory -Path $ArtifactRoot -Force | Out-Null
$Stage = Join-Path ([IO.Path]::GetTempPath()) ('csvdoctor-package-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $Stage | Out-Null
Copy-Item -LiteralPath (Join-Path $ProjectRoot 'build/bin/CSVDoctor.exe') -Destination $Stage
foreach ($Name in @('LICENSE','README.md','RELEASE_NOTES.md','SECURITY.md')) { Copy-Item -LiteralPath (Join-Path $ProjectRoot $Name) -Destination $Stage }
$Notices = [Collections.Generic.List[string]]::new()
$Notices.Add('Third-party license texts for dependencies linked into CSV Doctor.')
$GoRoot = go env GOROOT
$Notices.Add('Go standard library and runtime'); $Notices.Add((Get-Content -LiteralPath (Join-Path $GoRoot 'LICENSE') -Raw))
$ModuleNames = go list -deps -f '{{with .Module}}{{.Path}}{{end}}' . | Sort-Object -Unique
if ($LASTEXITCODE -ne 0) { throw 'Cannot determine linked dependencies' }
foreach ($ModuleName in $ModuleNames) {
 if (-not $ModuleName -or $ModuleName -eq 'github.com/AlinTibi/CSVDoctor') { continue }
 $Module = go list -m -json $ModuleName | ConvertFrom-Json
 if ($LASTEXITCODE -ne 0) { throw "Cannot locate dependency $ModuleName" }
 $LicenseFiles = Get-ChildItem -LiteralPath $Module.Dir -File | Where-Object { $_.Name -match '^(LICENSE|COPYING|NOTICE)(\.|$)' }
 if (-not $LicenseFiles) { throw "No license text found for $ModuleName" }
 $Notices.Add("`n$ModuleName $($Module.Version)")
 foreach ($LicenseFile in $LicenseFiles) { $Notices.Add((Get-Content -LiteralPath $LicenseFile.FullName -Raw)) }
}
$Notices -join "`n" | Set-Content -LiteralPath (Join-Path $Stage 'THIRD_PARTY_NOTICES.txt') -Encoding utf8
$Zip = Join-Path $ArtifactRoot "CSVDoctor-v$Version-win-x64.zip"
if (Test-Path -LiteralPath $Zip) { throw 'Candidate ZIP already exists; choose a new candidate number or remove only your disposable old artifact explicitly' }
Compress-Archive -Path (Join-Path $Stage '*') -DestinationPath $Zip -CompressionLevel Optimal
$Hash = (Get-FileHash -LiteralPath $Zip -Algorithm SHA256).Hash.ToLowerInvariant()
"$Hash  $([IO.Path]::GetFileName($Zip))" | Set-Content -LiteralPath "$Zip.sha256" -Encoding ascii
Write-Output "Candidate: $([IO.Path]::GetFileName($Zip))"
Write-Output "SHA-256: $Hash"
