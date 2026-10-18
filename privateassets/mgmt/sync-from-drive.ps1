# Mirrors <SourcePath>\sounds into privateassets/audio/, then rewrites the manifest.
#
#   powershell -File privateassets/mgmt/sync-from-drive.ps1 [-SourcePath <ascend_assets folder>]
#
# The source is the truth and audio/ is a mirror of it: one folder per pack plus
# bundle-names.txt, and a file removed or renamed at the source is deleted here. manifest.json is
# the one thing in audio/ the source does not hold, and /XF keeps the mirror from deleting it.

param([string]$SourcePath = '')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'config.ps1')

$cfg = Read-LocalConfig
$SourcePath = Resolve $SourcePath $cfg 'SourcePath' -Required
$source = Join-Path $SourcePath 'sounds'

if (-not (Test-Path $source)) {
    throw "no sound packs at $source"
}

# /MIR deletes whatever the source no longer holds. /XF names files the mirror neither copies nor
# deletes: desktop.ini is Drive's folder metadata, and manifest.json is the committed record
# that -write rebuilds below.
robocopy $source $AudioDir /MIR /XF desktop.ini manifest.json /NP /NJH /NJS
# robocopy's exit code is a bitmask and anything under 8 is success.
if ($LASTEXITCODE -ge 8) {
    throw "robocopy failed with exit code $LASTEXITCODE"
}

Push-Location $RepoRoot
try {
    go run ./tools/privateassets -write
    if ($LASTEXITCODE -ne 0) { throw 'privateassets -write failed' }
    go run ./tools/privateassets
} finally {
    Pop-Location
}
