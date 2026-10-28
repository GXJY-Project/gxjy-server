$ErrorActionPreference = 'Stop'
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$GoCommand = Get-Command go -ErrorAction SilentlyContinue
$GoPath = if ($GoCommand) { $GoCommand.Source } else { Join-Path $env:ProgramFiles 'Go\bin\go.exe' }
$GofmtPath = Join-Path (Split-Path -Parent $GoPath) 'gofmt.exe'

if (-not (Test-Path $GoPath)) {
    throw 'Go was not found. Install Go or add it to PATH.'
}

Push-Location $ProjectRoot
try {
    $SourceDirs = @('config', 'controllers', 'flag', 'global', 'middlewares', 'models', 'repositories', 'router', 'internal')
    $Packages = @('.') + @($SourceDirs | ForEach-Object { './' + $_ + '/...' })
    $GoFiles = @(Join-Path $ProjectRoot 'main.go') + @(
        foreach ($SourceDir in $SourceDirs) {
            Get-ChildItem -LiteralPath (Join-Path $ProjectRoot $SourceDir) -Recurse -Filter '*.go' | ForEach-Object FullName
        }
    )
    $Unformatted = @(& $GofmtPath -l $GoFiles)
    if ($LASTEXITCODE -ne 0) { throw 'gofmt failed.' }
    if ($Unformatted.Count -gt 0) {
        $Unformatted | Write-Output
        throw 'gofmt check failed.'
    }
    Write-Output 'gofmt: passed'

    & $GoPath test @Packages
    if ($LASTEXITCODE -ne 0) { throw 'go test failed.' }

    & $GoPath vet @Packages
    if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }
    Write-Output 'go vet: passed'

    & $GoPath build @Packages
    if ($LASTEXITCODE -ne 0) { throw 'go build failed.' }
    Write-Output 'go build: passed'
}
finally {
    Pop-Location
}
