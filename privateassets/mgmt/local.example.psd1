# Copy to local.psd1 (gitignored) and fill in. Read by config.ps1.
@{
    # The folder that is the source of truth: its sounds/ is the library, its game-ready/ is built
    # from it and mirrored into privateassets/audio/, and the whole folder is mirrored to the bucket.
    SourcePath = 'C:\path\to\ascend_assets'

    # The private S3 bucket the release fetches game-ready/ from.
    Bucket     = 'your-bucket-name'
    Region     = 'us-east-1'

    # An AWS CLI profile name, or '' for the default credentials.
    AwsProfile = ''
}
