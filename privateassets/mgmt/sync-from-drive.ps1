# Mirrors <SourcePath>\game-ready into privateassets/audio/, then records each file's hash in its
# catalog, data/music.json or data/sounds.json.
#
#   powershell -File privateassets/mgmt/sync-from-drive.ps1 [-SourcePath <ascend_assets folder>]
#
# game-ready\ is the source and audio/ is a mirror of it: music\ holding exactly the tracks
# data/music.json names and sounds\ exactly the effects data/sounds.json names, each flat, built by
# build-game-ready.ps1. A file removed or renamed there is deleted
# here. The library, <SourcePath>\sounds\, never reaches the repository.
#
# README.md is the one file in audio/ that game-ready does not hold. It is committed, because
# `//go:embed audio` fails to compile on a directory that is not there, and /XF keeps the mirror
# from deleting it.

param([string]$SourcePath = '')

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'config.ps1')

$cfg = Read-LocalConfig
$SourcePath = Resolve $SourcePath $cfg 'SourcePath' -Required
$source = Join-Path $SourcePath 'game-ready'

if (-not (Test-Path $source)) {
    throw "no game-ready folder at $source; build it with build-game-ready.ps1"
}

# /MIR deletes whatever the source no longer holds. /XF names files the mirror neither copies nor
# deletes: desktop.ini is Drive's folder metadata, README.md is the committed placeholder.
robocopy $source $AudioDir /MIR /XF desktop.ini README.md /NP /NJH /NJS
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

# The status of every track, so a record with nothing built yet is not forgotten.
Write-Host ''
& (Join-Path $PSScriptRoot 'build-game-ready.ps1') -SourcePath $SourcePath
