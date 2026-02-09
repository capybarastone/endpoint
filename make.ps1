$ErrorActionPreference = "Stop"

param(
    [string]$Command = "all"
)

$RootDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$BinDir = Join-Path $RootDir "bin"
$AppName = "capyendpoint"

function Show-Usage {
    Write-Output @"
Usage: .\make.ps1 [command]

Commands:
  format   Run go fmt ./...
  vet      Run go vet ./...
  test     Run go test ./...
  build    Build the project into .\bin
  clean    Remove .\bin
  all      Run format, vet, test, then build (default)
"@
}

function Invoke-Format {
    go fmt ./...
}

function Invoke-Vet {
    go vet ./...
}

function Invoke-Test {
    go test ./...
}

function Invoke-Build {
    New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
    go build -o (Join-Path $BinDir $AppName) .
}

function Invoke-Clean {
    if (Test-Path $BinDir) {
        Remove-Item -Path $BinDir -Recurse -Force
    }
}

function Invoke-All {
    Invoke-Format
    Invoke-Vet
    Invoke-Test
    Invoke-Build
}

switch ($Command.ToLowerInvariant()) {
    "format" { Invoke-Format }
    "vet"    { Invoke-Vet }
    "test"   { Invoke-Test }
    "build"  { Invoke-Build }
    "clean"  { Invoke-Clean }
    "all"    { Invoke-All }
    "help"   { Show-Usage }
    "-h"     { Show-Usage }
    "--help" { Show-Usage }
    default  {
        Write-Error "Unknown command: $Command"
        Show-Usage
        exit 1
    }
}
