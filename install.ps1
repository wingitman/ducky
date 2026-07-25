#Requires -Version 5.1
<#
.SYNOPSIS
    Installs ducky on Windows.
.PARAMETER BuildAll
    Cross-compiles the supported release binaries and exits.
##>
param([switch]$BuildAll)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$BinaryName = 'ducky.exe'
$InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\ducky'
$BuildDir = Join-Path $PSScriptRoot 'bin'
$ReleaseBin = Join-Path $PSScriptRoot 'releases\windows\ducky.exe'

function Step([string]$Message) { Write-Host "==> $Message" -ForegroundColor Cyan }

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    if ($BuildAll -or -not (Test-Path $ReleaseBin)) {
        throw 'Go is required. Install Go from https://go.dev/dl/ and retry.'
    }
    $SourceBin = $ReleaseBin
} else {
    $Commit = (git rev-parse HEAD 2>$null)
    if (-not $Commit) { $Commit = 'dev' }
    if ($BuildAll) {
        $targets = @(
            @{ OS = 'linux'; Arch = 'amd64'; Out = 'releases\linux\amd64\ducky' },
            @{ OS = 'linux'; Arch = 'arm64'; Out = 'releases\linux\arm64\ducky' },
            @{ OS = 'darwin'; Arch = 'amd64'; Out = 'releases\darwin\amd64\ducky' },
            @{ OS = 'darwin'; Arch = 'arm64'; Out = 'releases\darwin\arm64\ducky' },
            @{ OS = 'windows'; Arch = 'amd64'; Out = 'releases\windows\ducky.exe' }
        )
        foreach ($target in $targets) {
            Step "Building $($target.OS)/$($target.Arch)"
            $out = Join-Path $PSScriptRoot $target.Out
            New-Item -ItemType Directory -Path (Split-Path $out) -Force | Out-Null
            $env:GOOS = $target.OS; $env:GOARCH = $target.Arch; $env:CGO_ENABLED = '0'
            & go build -ldflags="-s -w -X github.com/wingitman/ducky/internal/version.Commit=$Commit" -o $out ./cmd/ducky
            if ($LASTEXITCODE -ne 0) { throw "build failed for $($target.OS)/$($target.Arch)" }
        }
        $env:GOOS = $null; $env:GOARCH = $null; $env:CGO_ENABLED = $null
        Write-Host 'Pre-built binaries written to releases\' -ForegroundColor Green
        exit 0
    }
    Step 'Building ducky from source'
    New-Item -ItemType Directory -Path $BuildDir -Force | Out-Null
    $built = Join-Path $BuildDir $BinaryName
    & go build -ldflags="-s -w -X github.com/wingitman/ducky/internal/version.Commit=$Commit" -o $built ./cmd/ducky
    if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
    $SourceBin = $built
}

Step 'Installing ducky'
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item $SourceBin (Join-Path $InstallDir $BinaryName) -Force
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (($userPath -split ';') -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable('Path', (($userPath + ';' + $InstallDir).Trim(';')), 'User')
}
$env:Path = "$env:Path;$InstallDir"
Write-Host "Installed: $(Join-Path $InstallDir $BinaryName)" -ForegroundColor Green
