# One-time setup for the private sound bucket.
#
#   powershell -File privateassets/mgmt/setup-aws.ps1 [-Bucket <name>] [-Region <r>] [-AwsProfile <p>]
#
# Bucket, Region and AwsProfile come from local.psd1 unless given here.
#
# Installs the AWS CLI if it is missing, then creates, or brings into line if they already exist:
#   - the S3 bucket: private, public access blocked, versioned, TLS only
#   - GitHub's OIDC identity provider in IAM
#   - a role the release jobs assume, readable only from main and v* tags of this repository,
#     granting s3:ListBucket and s3:GetObject on game-ready/ and nothing else — not the library
#
# Safe to re-run. It never sets the GitHub repository variables; it prints the commands, because
# setting SOUNDS_BUCKET arms the release's -require guard and that is a decision, not a side effect.

param(
    [string]$Bucket = '',
    [string]$Region = '',
    [string]$AwsProfile = '',
    [string]$Repo = 'CuriousJC/ascend-duel',
    [string]$RoleName = 'duello-release-sounds'
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'config.ps1')

$cfg = Read-LocalConfig
$Bucket = Resolve $Bucket $cfg 'Bucket' -Required
$Region = Resolve $Region $cfg 'Region' -Default 'us-east-1'
$AwsProfile = Resolve $AwsProfile $cfg 'AwsProfile'

# PowerShell names are case-insensitive, so inside a function called Aws a bare `aws` is the
# function itself. Every call to the CLI names `aws.exe`.
#
# Both wrappers judge the CLI by its exit code alone. Under 'Stop', Windows PowerShell 5.1 turns
# any line a native command writes to stderr into a terminating error, redirected or not, so the
# preference is relaxed inside them.
function Aws {
    $ErrorActionPreference = 'Continue'
    $a = @($args)
    if ($AwsProfile) { $a += @('--profile', $AwsProfile) }
    $out = & aws.exe @a
    if ($LASTEXITCODE -ne 0) { throw "aws $($args -join ' ') failed with exit code $LASTEXITCODE" }
    return $out
}

# Like Aws, but a failure is an answer rather than an error: "does this exist yet".
function AwsQuiet {
    $ErrorActionPreference = 'Continue'
    $a = @($args)
    if ($AwsProfile) { $a += @('--profile', $AwsProfile) }
    $out = & aws.exe @a 2>$null
    return @{ Ok = ($LASTEXITCODE -eq 0); Out = $out }
}

function WriteJson($obj) {
    $path = [System.IO.Path]::GetTempFileName()
    $obj | ConvertTo-Json -Depth 10 | Set-Content -Path $path -Encoding ascii
    return "file://$path"
}

# --- the CLI -------------------------------------------------------------------------------------

if (-not (Get-Command aws.exe -CommandType Application -ErrorAction SilentlyContinue)) {
    Write-Host 'installing the AWS CLI'
    winget install -e --id Amazon.AWSCLI --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -ne 0) { throw 'winget could not install the AWS CLI' }
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not (Get-Command aws.exe -CommandType Application -ErrorAction SilentlyContinue)) {
        throw 'the AWS CLI installed but is not on PATH yet; open a new terminal and re-run'
    }
}
Write-Host (& aws.exe --version)

$who = AwsQuiet sts get-caller-identity --output json
if (-not $who.Ok) {
    throw "no working AWS credentials. Run 'aws configure sso' (or 'aws configure') first, then re-run with -AwsProfile if you named one."
}
$account = ($who.Out | ConvertFrom-Json).Account
Write-Host "account $account, region $Region"

# --- the bucket ----------------------------------------------------------------------------------

if ((AwsQuiet s3api head-bucket --bucket $Bucket).Ok) {
    Write-Host "bucket $Bucket exists"
} else {
    Write-Host "creating bucket $Bucket"
    if ($Region -eq 'us-east-1') {
        # us-east-1 refuses an explicit LocationConstraint.
        Aws s3api create-bucket --bucket $Bucket --region $Region | Out-Null
    } else {
        Aws s3api create-bucket --bucket $Bucket --region $Region --create-bucket-configuration "LocationConstraint=$Region" | Out-Null
    }
}

Aws s3api put-public-access-block --bucket $Bucket --public-access-block-configuration `
    'BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true' | Out-Null

# Versioned because the refresh mirrors with --delete: a file dropped from Drive by mistake is a
# delete marker here, and recoverable, rather than gone.
Aws s3api put-bucket-versioning --bucket $Bucket --versioning-configuration Status=Enabled | Out-Null

$bucketPolicy = @{
    Version   = '2012-10-17'
    Statement = @(@{
            Sid       = 'TlsOnly'
            Effect    = 'Deny'
            Principal = '*'
            Action    = 's3:*'
            Resource  = @("arn:aws:s3:::$Bucket", "arn:aws:s3:::$Bucket/*")
            Condition = @{ Bool = @{ 'aws:SecureTransport' = 'false' } }
        })
}
Aws s3api put-bucket-policy --bucket $Bucket --policy (WriteJson $bucketPolicy) | Out-Null
Write-Host 'bucket: public access blocked, versioning on, TLS only'

# --- GitHub's OIDC provider ----------------------------------------------------------------------

$providerArn = "arn:aws:iam::${account}:oidc-provider/token.actions.githubusercontent.com"
if ((AwsQuiet iam get-open-id-connect-provider --open-id-connect-provider-arn $providerArn).Ok) {
    Write-Host 'GitHub OIDC provider exists'
} else {
    Write-Host 'creating the GitHub OIDC provider'
    # IAM validates GitHub's certificate against its own trusted CAs; the thumbprint is required by
    # the API and no longer what decides trust.
    Aws iam create-open-id-connect-provider --url 'https://token.actions.githubusercontent.com' `
        --client-id-list 'sts.amazonaws.com' --thumbprint-list '6938fd4d98bab03faadb97b34396831e3780aea1' | Out-Null
}

# --- the release role ----------------------------------------------------------------------------

# Only a workflow run of this repository on main (the manual release) or on a v* tag (a pushed
# tag) can assume it. No environment is used by the release jobs, so the sub claim is the ref.
$trust = @{
    Version   = '2012-10-17'
    Statement = @(@{
            Effect    = 'Allow'
            Principal = @{ Federated = $providerArn }
            Action    = 'sts:AssumeRoleWithWebIdentity'
            Condition = @{
                StringEquals = @{ 'token.actions.githubusercontent.com:aud' = 'sts.amazonaws.com' }
                StringLike   = @{ 'token.actions.githubusercontent.com:sub' = @(
                        "repo:${Repo}:ref:refs/heads/main",
                        "repo:${Repo}:ref:refs/tags/v*"
                    )
                }
            }
        })
}

if ((AwsQuiet iam get-role --role-name $RoleName).Ok) {
    Write-Host "role $RoleName exists; refreshing its trust policy"
    Aws iam update-assume-role-policy --role-name $RoleName --policy-document (WriteJson $trust) | Out-Null
} else {
    Write-Host "creating role $RoleName"
    Aws iam create-role --role-name $RoleName --assume-role-policy-document (WriteJson $trust) `
        --description 'Duello release jobs: read the game-ready sounds' | Out-Null
}

# Read-only, and only under game-ready/: list the prefix, get the objects. The library under
# sounds/ is a backup and a release has no business reading it.
$read = @{
    Version   = '2012-10-17'
    Statement = @(
        @{
            Effect    = 'Allow'
            Action    = 's3:ListBucket'
            Resource  = "arn:aws:s3:::$Bucket"
            Condition = @{ StringLike = @{ 's3:prefix' = @('game-ready/*') } }
        },
        @{
            Effect   = 'Allow'
            Action   = 's3:GetObject'
            Resource = "arn:aws:s3:::$Bucket/game-ready/*"
        }
    )
}
Aws iam put-role-policy --role-name $RoleName --policy-name 'read-sounds' --policy-document (WriteJson $read) | Out-Null

$roleArn = "arn:aws:iam::${account}:role/$RoleName"
Write-Host ''
Write-Host "done. Role: $roleArn"
Write-Host ''
Write-Host 'Next: upload the sounds with privateassets/mgmt/push-sounds.ps1. Then, when you want releases to'
Write-Host 'carry sound and to refuse a build without it, set the repository variables:'
Write-Host ''
Write-Host "  gh variable set SOUNDS_BUCKET   --repo $Repo --body $Bucket"
Write-Host "  gh variable set SOUNDS_ROLE_ARN --repo $Repo --body $roleArn"
Write-Host "  gh variable set SOUNDS_REGION   --repo $Repo --body $Region"
