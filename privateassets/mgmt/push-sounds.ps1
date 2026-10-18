# Makes the S3 bucket an exact copy of the source folder.
#
#   powershell -File privateassets/mgmt/push-sounds.ps1 [-DryRun] [-Bucket <name>] [-AwsProfile <p>]
#
# The source is the truth. This first mirrors its sounds/ into privateassets/audio/ and rewrites
# the manifest (sync-from-drive.ps1), refuses to go on unless -require passes, then mirrors the
# whole source folder to the bucket root with --delete — so the packs land at
# s3://<bucket>/sounds/, which is where the release fetches them. The bucket is versioned, so a
# deletion here is a delete marker and recoverable. Use -DryRun first to see what would change.
#
# A release requires exactly the files the *committed* manifest names, so commit manifest.json
# after a push that changed it — the script says so when it has.

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

if (-not (Test-Path (Join-Path $SourcePath 'sounds'))) { throw "no sounds folder under $SourcePath" }

# 1. The source into the repo, and the manifest rebuilt from it.
& (Join-Path $PSScriptRoot 'sync-from-drive.ps1') -SourcePath $SourcePath

Push-Location $RepoRoot
try {
    go run ./tools/privateassets -require
    if ($LASTEXITCODE -ne 0) { throw 'privateassets -require failed; fix the manifest before pushing' }
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
    $changed = git status --porcelain -- privateassets/audio/manifest.json
} finally {
    Pop-Location
}
if ($changed) {
    Write-Host ''
    Write-Host 'manifest.json differs from what is committed. Commit it before cutting a release:'
    Write-Host 'a release requires exactly the files the committed manifest names.'
}
