# Makes the S3 bucket an exact copy of the source folder.
#
#   powershell -File privateassets/mgmt/push-sounds.ps1 [-DryRun] [-Bucket <name>] [-AwsProfile <p>]
#
# The source is the truth. This first mirrors its game-ready/ into privateassets/audio/ and records
# the hashes in data/music.json (sync-from-drive.ps1), refuses to go on unless -require passes,
# then mirrors the whole source folder to the bucket root with --delete. So the game-ready tracks
# land at s3://<bucket>/game-ready/, which is all a release fetches, and the library lands at
# s3://<bucket>/sounds/ as a backup the release role cannot read. The bucket is versioned, so a
# deletion here is a delete marker and recoverable. Use -DryRun first to see what would change.
#
# A release requires exactly the files the *committed* music.json names, so commit it after a push
# that changed it — the script says so when it has.

param(
    [string]$Bucket = '',
    [string]$AwsProfile = '',
    [string]$SourcePath = '',
    [switch]$DryRun
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'config.ps1')

$cfg = Read-LocalConfig
$Bucket = Resolve $Bucket $cfg 'Bucket' -Required
$AwsProfile = Resolve $AwsProfile $cfg 'AwsProfile'
$SourcePath = Resolve $SourcePath $cfg 'SourcePath' -Required

if (-not (Test-Path (Join-Path $SourcePath 'game-ready'))) { throw "no game-ready folder under $SourcePath" }

# 1. game-ready into the repo, and its hashes recorded.
& (Join-Path $PSScriptRoot 'sync-from-drive.ps1') -SourcePath $SourcePath

Push-Location $RepoRoot
try {
    go run ./tools/privateassets -require
    if ($LASTEXITCODE -ne 0) { throw 'privateassets -require failed; fix data/music.json or game-ready before pushing' }
} finally {
    Pop-Location
}

# 2. The source to the bucket. desktop.ini is Drive's folder metadata and never goes up.
$target = "s3://$Bucket/"
$a = @('s3', 'sync', $SourcePath, $target, '--delete', '--exclude', '*desktop.ini')
if ($DryRun) { $a += '--dryrun' }
if ($AwsProfile) { $a += @('--profile', $AwsProfile) }

Write-Host ''
Write-Host "mirroring $SourcePath -> $target$(if ($DryRun) { ' (dry run)' })"
& aws @a
if ($LASTEXITCODE -ne 0) { throw "aws s3 sync failed with exit code $LASTEXITCODE" }

Push-Location $RepoRoot
try {
    $changed = git status --porcelain -- data/music.json
} finally {
    Pop-Location
}
if ($changed) {
    Write-Host ''
    Write-Host 'data/music.json differs from what is committed. Commit it before cutting a release:'
    Write-Host 'a release requires exactly the files the committed music.json names.'
}
