# Builds <SourcePath>\game-ready\ from the library, as data/music.json and data/sounds.json say.
#
#   powershell -File privateassets/mgmt/build-game-ready.ps1                       # status of every sound
#   powershell -File privateassets/mgmt/build-game-ready.ps1 -Track combat-ambience-1 [-Track ...]
#   powershell -File privateassets/mgmt/build-game-ready.ps1 -Missing              # build what is not built yet
#   powershell -File privateassets/mgmt/build-game-ready.ps1 -All                  # rebuild everything
#   powershell -File privateassets/mgmt/build-game-ready.ps1 -Measure <pack>/<file>.wav
#
# The library, <SourcePath>\sounds\<pack>\, is never written: each record's Source is read from it
# and its game-ready File written beside it, into game-ready\music\ for a track and game-ready\sounds\
# for an effect, each flat. game-ready\ is derived, so a build overwrites what is there; it is the
# folder sync-from-drive.ps1 mirrors into privateassets/audio/ and push-sounds.ps1 sends to the bucket.
# -Track takes either kind by name: a name is unique across both catalogs.
#
# For each sound built:
#   - a MIDI file, or a source already in its game-ready format at no gain, is copied.
#   - anything else is encoded with ffmpeg's libvorbis at the record's own Quality, with the record's
#     own GainDB applied, at the game's 44100 Hz, carrying the text tags over (-map_metadata 0). Embedded cover art and the
#     WAV-only chunks (bext, iXML, editor blocks) have no Ogg equivalent and are left in the library.
#   - the result is refused unless its duration matches the source's to the millisecond, since a
#     loop that comes out longer or shorter has a gap or a cut at the seam, and unless it peaks
#     below -0.1 dBFS, since one at full scale has clipped: lower the record's GainDB.
#
# -Measure reads one library file and says its sample and true peaks, and the gain that would put
# the true peak at -1.0 dBFS, which is the figure a new record's GainDB starts from. It changes
# nothing.
#
# Afterwards: sync-from-drive.ps1, which mirrors game-ready\ into audio/ and records each file's
# hash in its catalog.

param(
    [string[]]$Track = @(),
    [switch]$Missing,
    [switch]$All,
    [string]$Measure = '',
    [string]$SourcePath = ''
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'config.ps1')

$cfg = Read-LocalConfig
$SourcePath = Resolve $SourcePath $cfg 'SourcePath' -Required
$library = Join-Path $SourcePath 'sounds'
$ready = Join-Path $SourcePath 'game-ready'
if (-not (Test-Path $library)) { throw "no library at $library" }

# The game's audio context runs at internal/music's sampleRate, so an encode lands there and the
# decoder resamples nothing at startup.
$GameRate = 44100

# Every record of both catalogs, each tagged with the folder it is filed in.
$tracks = @()
foreach ($kind in @(@('music', 'music.json'), @('sounds', 'sounds.json'))) {
    $records = Get-Content -Raw -Encoding UTF8 (Join-Path $RepoRoot ('data\' + $kind[1])) | ConvertFrom-Json
    foreach ($r in @($records)) {
        $r | Add-Member -NotePropertyName Dir -NotePropertyValue $kind[0]
        $tracks += $r
    }
}
function ReadyPath($t) { Join-Path (Join-Path $ready $t.Dir) $t.File }

function TrackName($t) { [IO.Path]::GetFileNameWithoutExtension($t.File) }
function IsEncoded($t) {
    ([IO.Path]::GetExtension($t.File) -ne [IO.Path]::GetExtension($t.Source)) -or
        ($null -ne $t.GainDB -and [double]$t.GainDB -ne 0)
}

# --- the tools -----------------------------------------------------------------------------------

function NeedTools {
    # winget puts ffmpeg on the user PATH, which a terminal opened before the install has not read.
    $env:Path = [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' + [Environment]::GetEnvironmentVariable('Path', 'User')
    foreach ($tool in 'ffmpeg.exe', 'ffprobe.exe') {
        if (-not (Get-Command $tool -CommandType Application -ErrorAction SilentlyContinue)) {
            throw "$tool is not on PATH. Install it with: winget install -e --id Gyan.FFmpeg"
        }
    }
}

# Native tools are judged by their exit code. Under 'Stop', Windows PowerShell 5.1 turns a line on
# a redirected stderr into a terminating error, so the preference is relaxed around each call.
function Probe([string]$file) {
    $ErrorActionPreference = 'Continue'
    $json = & ffprobe.exe -v error -print_format json -show_format -show_streams $file
    if ($LASTEXITCODE -ne 0) { throw "ffprobe could not read $file" }
    return ($json | Out-String | ConvertFrom-Json)
}

# The file's sample peak in dBFS, off ffmpeg's volumedetect (which reports on stderr).
function PeakDb([string]$file) {
    $ErrorActionPreference = 'Continue'
    $log = & ffmpeg.exe -hide_banner -nostdin -i $file -map 0:a:0 -af volumedetect -f null NUL 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) { throw "ffmpeg could not measure $file" }
    if ($log -match 'max_volume:\s*(-?[\d.]+) dB') { return [double]$Matches[1] }
    throw "no peak reported for $file"
}

# The file's true peak in dBFS — the waveform between the samples, off ffmpeg's ebur128. A master
# whose sample peak is under full scale can still go over between samples, and a resample or a
# lossy encode is what brings that out, so this is the figure a gain is chosen against.
function TruePeakDb([string]$file) {
    $ErrorActionPreference = 'Continue'
    $log = & ffmpeg.exe -hide_banner -nostdin -i $file -map 0:a:0 -af 'ebur128=peak=true' -f null NUL 2>&1 | Out-String
    if ($LASTEXITCODE -ne 0) { throw "ffmpeg could not measure $file" }
    if ($log -match 'True peak:\s*\r?\n\s*Peak:\s*(-?[\d.]+) dBFS') { return [double]$Matches[1] }
    throw "no true peak reported for $file"
}

function Encode([string]$from, [string]$to, $t) {
    $ErrorActionPreference = 'Continue'
    # **Bit-exact, so the same record builds the same bytes.** Without it ffmpeg gives every Ogg a
    # random stream serial, and a rebuild of an unchanged record would change its hash — a fresh push
    # and a fresh commit for nothing.
    $a = @('-hide_banner', '-loglevel', 'error', '-nostdin', '-y', '-i', $from, '-map', '0:a:0', '-map_metadata', '0',
        '-fflags', '+bitexact', '-flags:a', '+bitexact')
    $gain = [double]$t.GainDB
    if ($gain -ne 0) { $a += @('-af', ('volume={0}dB' -f $gain.ToString([Globalization.CultureInfo]::InvariantCulture))) }
    $a += @('-ar', [string]$GameRate, '-c:a', 'libvorbis', '-q:a', [string]$t.Quality, $to)
    & ffmpeg.exe @a
    if ($LASTEXITCODE -ne 0) { throw "ffmpeg failed on $from" }
}

# --- -Measure ------------------------------------------------------------------------------------

if ($Measure) {
    NeedTools
    $file = Join-Path $library ($Measure -replace '/', '\')
    if (-not (Test-Path $file -PathType Leaf)) { throw "no file at $file" }
    $peak = PeakDb $file
    $truePeak = TruePeakDb $file
    $a = (Probe $file).streams | Where-Object codec_type -eq 'audio' | Select-Object -First 1
    Write-Host ('{0}: sample peak {1:N1} dBFS, true peak {2:N1} dBFS, {3} ch, {4} Hz, {5}' -f `
        $Measure, $peak, $truePeak, $a.channels, $a.sample_rate, $a.codec_name)
    Write-Host ('  GainDB {0:N1} puts its true peak at -1.0 dBFS' -f (-1.0 - $truePeak))
    return
}

# --- with nothing to build, the status -----------------------------------------------------------

if ($Track.Count -eq 0 -and -not $Missing -and -not $All) {
    $byFile = @{}
    foreach ($t in $tracks) {
        $byFile[$t.Dir + '\' + $t.File] = $true
        $dst = ReadyPath $t
        $state = if (-not (Test-Path $dst)) { 'not built' }
        elseif (-not $t.SHA256) { 'built, not recorded' }
        elseif ((Get-FileHash $dst -Algorithm SHA256).Hash.ToLower() -ne $t.SHA256) { 'built, differs from its record' }
        else { 'built, matches its record' }
        Write-Host ('  {0,-28} {1}' -f (TrackName $t), $state)
    }
    if (Test-Path $ready) {
        foreach ($f in Get-ChildItem $ready -File -Recurse | Where-Object Name -ne 'desktop.ini') {
            $rel = $f.FullName.Substring($ready.Length + 1)
            if (-not $byFile[$rel]) { Write-Host ('  {0,-28} in game-ready and named by no record' -f $rel) }
        }
    }
    return
}

# --- the builds ----------------------------------------------------------------------------------

$chosen = @()
if ($All) { $chosen = $tracks }
elseif ($Missing) { $chosen = @($tracks | Where-Object { -not (Test-Path (ReadyPath $_)) }) }
else {
    foreach ($name in $Track) {
        $t = $tracks | Where-Object { (TrackName $_) -eq $name } | Select-Object -First 1
        if (-not $t) { throw "no sound named $name in data/music.json or data/sounds.json" }
        $chosen += $t
    }
}
if ($chosen.Count -eq 0) {
    Write-Host 'nothing to build'
    return
}

NeedTools
New-Item -ItemType Directory -Force $ready | Out-Null

foreach ($t in $chosen) {
    $name = TrackName $t
    $src = Join-Path $library ($t.Source -replace '/', '\')
    $dst = ReadyPath $t
    New-Item -ItemType Directory -Force (Split-Path -Parent $dst) | Out-Null
    if (-not (Test-Path $src -PathType Leaf)) { throw "${name}: no source at $src" }

    if (-not (IsEncoded $t)) {
        Copy-Item $src $dst -Force
        Write-Host "copied $name"
        continue
    }

    # Encoded outside Drive first, so the sync client never uploads a half-written file.
    $tmp = Join-Path ([IO.Path]::GetTempPath()) ("duello-" + [Guid]::NewGuid().ToString('N') + [IO.Path]::GetExtension($t.File))
    try {
        Write-Host ('encoding {0} (q{1}, {2:+0.0;-0.0;0} dB)' -f $name, $t.Quality, [double]$t.GainDB)
        Encode $src $tmp $t

        $in = Probe $src
        $out = Probe $tmp
        $a = $in.streams | Where-Object codec_type -eq 'audio' | Select-Object -First 1
        $b = $out.streams | Where-Object codec_type -eq 'audio' | Select-Object -First 1
        $dIn = [double]$in.format.duration
        $dOut = [double]$out.format.duration
        if ([Math]::Abs($dIn - $dOut) -gt 0.001) {
            throw ("{0}: the Ogg runs {1:N4}s against the source's {2:N4}s; not filing it" -f $name, $dOut, $dIn)
        }
        if ([int]$b.sample_rate -ne $GameRate -or $a.channels -ne $b.channels) {
            throw "${name}: the Ogg is $($b.channels) ch at $($b.sample_rate) Hz; wanted $($a.channels) ch at $GameRate Hz"
        }
        $peakIn = PeakDb $src
        $peakOut = PeakDb $tmp
        if ($peakOut -gt -0.1) {
            throw ("{0}: the Ogg peaks at {1:N1} dBFS and has clipped; lower its GainDB" -f $name, $peakOut)
        }

        Move-Item $tmp $dst -Force

        $tags = @()
        if ($b.tags) { $tags = @($b.tags.PSObject.Properties | ForEach-Object { $_.Name }) }
        Write-Host ('  {0:N1} MB -> {1:N1} MB, {2} ch, {3} Hz, {4:N3}s, peak {5:N1} -> {6:N1} dBFS' -f `
            ((Get-Item $src).Length / 1MB), ((Get-Item $dst).Length / 1MB), $b.channels, $b.sample_rate, $dOut, $peakIn, $peakOut)
        Write-Host "  tags carried: $(if ($tags) { $tags -join ', ' } else { 'none' })"
    } finally {
        if (Test-Path $tmp) { Remove-Item $tmp -Force }
    }
}

Write-Host ''
Write-Host 'Next: sync-from-drive.ps1, to mirror game-ready into audio/ and record the hashes in the catalogs.'
