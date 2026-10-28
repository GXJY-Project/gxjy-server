param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ServerArgs
)

$ErrorActionPreference = 'Stop'
$ProjectRoot = Split-Path -Parent $PSScriptRoot
$GoCommand = Get-Command go -ErrorAction SilentlyContinue
$GoPath = if ($GoCommand) { $GoCommand.Source } else { Join-Path $env:ProgramFiles 'Go\bin\go.exe' }

if (-not (Test-Path $GoPath)) {
    throw 'Go was not found. Install Go or add it to PATH.'
}

Push-Location $ProjectRoot
try {
    & $GoPath run . @ServerArgs
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
finally {
    Pop-Location
}
