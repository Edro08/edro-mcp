#Requires -Version 5.1
[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$ForwardArgs
)

$ErrorActionPreference = 'Stop'

$repoRoot = $PSScriptRoot
$userHome = [Environment]::GetFolderPath('UserProfile')
if ([string]::IsNullOrWhiteSpace($userHome)) {
    $userHome = $HOME
}

$binDir = Join-Path (Join-Path $userHome '.mcp') 'bin'
if (-not (Test-Path -LiteralPath $binDir)) {
    New-Item -ItemType Directory -Path $binDir -Force | Out-Null
}

$exeName = if ($IsWindows -or $env:OS -eq 'Windows_NT') { 'edro-mcp.exe' } else { 'edro-mcp' }
$cliBin = Join-Path $binDir $exeName

Write-Host "[edro-mcp] Compilando CLI en $cliBin..." -ForegroundColor Cyan
Push-Location $repoRoot
try {
    & go build -o $cliBin ./cmd/edro-mcp
    if ($LASTEXITCODE -ne 0) {
        throw "Error al compilar edro-mcp (exit code $LASTEXITCODE)."
    }

    if ($ForwardArgs -and $ForwardArgs.Count -gt 0) {
        & $cliBin @ForwardArgs
    } else {
        & $cliBin install
    }
    exit $LASTEXITCODE
}
finally {
    Pop-Location
}
