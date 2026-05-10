# notebooklm-mcp Windows installer
# Usage:
#   iwr -useb https://raw.githubusercontent.com/vankcdhv/notebook-mcp/main/install.ps1 | iex
#
# Environment overrides:
#   NOTEBOOKLM_MCP_INSTALL_DIR  (default: $env:LOCALAPPDATA\notebooklm-mcp\bin)
#   NOTEBOOKLM_MCP_VERSION      (default: latest)

$ErrorActionPreference = "Stop"

$Repo = "vankcdhv/notebook-mcp"
$BinName = "notebooklm-mcp.exe"
$InstallDir = if ($env:NOTEBOOKLM_MCP_INSTALL_DIR) { $env:NOTEBOOKLM_MCP_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "notebooklm-mcp\bin" }

function Fail($Message) {
    Write-Error "error: $Message"
    exit 1
}

switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()) {
    "X64" { $Arch = "amd64" }
    "Arm64" { $Arch = "arm64" }
    default { Fail "unsupported architecture: $([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture)" }
}

$Version = if ($args.Count -gt 0) { $args[0] } elseif ($env:NOTEBOOKLM_MCP_VERSION) { $env:NOTEBOOKLM_MCP_VERSION } else { "latest" }
if ($Version -eq "latest") {
    Write-Host "==> Resolving latest release"
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -Headers @{ "User-Agent" = "notebooklm-mcp-installer" }
    $Version = $Release.tag_name
}
$Version = $Version.TrimStart("v")

$Archive = "notebooklm-mcp_${Version}_windows_${Arch}.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/v${Version}/$Archive"
$ChecksumsUrl = "https://github.com/$Repo/releases/download/v${Version}/checksums.txt"
$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("notebooklm-mcp-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $TempDir | Out-Null

try {
    $ArchivePath = Join-Path $TempDir $Archive
    $ChecksumsPath = Join-Path $TempDir "checksums.txt"

    Write-Host "==> Downloading $Archive"
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $ArchivePath -Headers @{ "User-Agent" = "notebooklm-mcp-installer" }

    Write-Host "==> Verifying checksum"
    Invoke-WebRequest -Uri $ChecksumsUrl -OutFile $ChecksumsPath -Headers @{ "User-Agent" = "notebooklm-mcp-installer" }
    $Expected = (Select-String -Path $ChecksumsPath -Pattern ([regex]::Escape($Archive) + '$')).Line.Split()[0]
    if (-not $Expected) { Fail "no checksum entry for $Archive" }
    $Actual = (Get-FileHash -Algorithm SHA256 -Path $ArchivePath).Hash.ToLowerInvariant()
    if ($Actual -ne $Expected.ToLowerInvariant()) { Fail "checksum mismatch (expected $Expected, got $Actual)" }

    Write-Host "==> Installing to $(Join-Path $InstallDir $BinName)"
    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force
    $Source = Join-Path $TempDir $BinName
    if (-not (Test-Path $Source)) {
        $Source = Get-ChildItem -Path $TempDir -Recurse -Filter $BinName | Select-Object -First 1 -ExpandProperty FullName
    }
    if (-not $Source -or -not (Test-Path $Source)) { Fail "binary not found in archive" }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -Force $Source (Join-Path $InstallDir $BinName)

    $BinPath = Join-Path $InstallDir $BinName
    Write-Host ""
    Write-Host "✓ Installed notebooklm-mcp v$Version -> $BinPath" -ForegroundColor Green
    Write-Host ""

    $ProfilePath = Join-Path $env:USERPROFILE ".notebooklm-mcp\profile.json"
    if (Test-Path $ProfilePath) {
        Write-Host "Existing login profile detected — no setup needed."
        Write-Host ""
        Write-Host "If you have not registered the MCP yet, run:"
        Write-Host "  claude mcp add -s user notebooklm-mcp `"$BinPath`""
    } else {
        Write-Host "Next steps:"
        Write-Host "  1. First-time setup (installs Playwright browser, opens Google login):"
        Write-Host "       & `"$BinPath`" setup"
        Write-Host ""
        Write-Host "  2. Register with Claude Code (user scope = global):"
        Write-Host "       claude mcp add -s user notebooklm-mcp `"$BinPath`""
        Write-Host ""
        Write-Host "  3. Restart your Claude Code session."
    }
} finally {
    Remove-Item -Recurse -Force $TempDir -ErrorAction SilentlyContinue
}
