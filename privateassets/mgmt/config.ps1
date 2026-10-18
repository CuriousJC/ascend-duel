# Dot-sourced by every script in this directory. Reads local.psd1 — this machine's own values,
# gitignored — so the scripts themselves can be committed. local.example.psd1 is the template.
#
# A script's own parameter wins over the file, so a one-off -Bucket or -AwsProfile still works.

$MgmtDir = $PSScriptRoot
$RepoRoot = Split-Path -Parent (Split-Path -Parent $MgmtDir)
$AudioDir = Join-Path $RepoRoot 'privateassets\audio'

function Read-LocalConfig {
    $path = Join-Path $MgmtDir 'local.psd1'
    if (-not (Test-Path $path)) {
        throw "no $path. Copy local.example.psd1 beside it to local.psd1 and fill it in."
    }
    # Import-PowerShellDataFile reads data only: a .psd1 cannot run code.
    return Import-PowerShellDataFile -Path $path
}

# Resolve picks the parameter if it was given, else the config value, else the default, and
# refuses an empty answer for a required setting.
function Resolve {
    param([string]$Given, [hashtable]$Config, [string]$Key, [string]$Default = '', [switch]$Required)
    $value = $Given
    if (-not $value -and $Config.ContainsKey($Key)) { $value = [string]$Config[$Key] }
    if (-not $value) { $value = $Default }
    if ($Required -and -not $value) {
        throw "$Key is not set: give it as a parameter or in local.psd1"
    }
    return $value
}
