
param(
    [string]$RepoRoot = (Get-Location).Path,
    [string]$OutRoot = ''
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = 'Stop'

function Get-GoCommand {
    $preferred = 'K:\go\go1.20.14\bin\go.exe'
    if (Test-Path -LiteralPath $preferred) {
        return $preferred
    }
    $go = Get-Command go -ErrorAction SilentlyContinue
    if ($go) {
        return $go.Source
    }
    throw 'go executable was not found in PATH.'
}

function New-CheckContext {
    param(
        [Parameter(Mandatory=$true)][string]$RepoRoot,
        [string]$OutRoot = ''
    )

    $repo = (Resolve-Path -LiteralPath $RepoRoot).Path
    if ($OutRoot -eq '') {
        $OutRoot = Join-Path $repo '.check-results'
    }

    $timestamp = Get-Date -Format 'yyyyMMdd_HHmmss'
    $resultDir = Join-Path $OutRoot "sttq_stage2_${timestamp}"
    $logsDir = Join-Path $resultDir 'logs'
    $inputsDir = Join-Path $resultDir 'inputs'
    $outputsDir = Join-Path $resultDir 'outputs'
    $metaDir = Join-Path $resultDir 'meta'
    $tmpDir = Join-Path $resultDir 'tmp'

    foreach ($dir in @($resultDir, $logsDir, $inputsDir, $outputsDir, $metaDir, $tmpDir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }

    $ctx = [ordered]@{
        RepoRoot = $repo
        ResultDir = $resultDir
        LogsDir = $logsDir
        InputsDir = $inputsDir
        OutputsDir = $outputsDir
        MetaDir = $metaDir
        TmpDir = $tmpDir
        CommandsPath = Join-Path $resultDir 'commands.jsonl'
        Assessments = New-Object System.Collections.ArrayList
        CommandResults = @{}
        StartedAt = (Get-Date).ToString('o')
        GoCmd = Get-GoCommand
    }
    '' | Set-Content -LiteralPath $ctx.CommandsPath -Encoding UTF8
    return $ctx
}

function Save-Json {
    param(
        [Parameter(Mandatory=$true)][string]$Path,
        [Parameter(Mandatory=$true)]$Value
    )

    $json = $Value | ConvertTo-Json -Depth 40
    Set-Content -LiteralPath $Path -Value $json -Encoding UTF8
}

function Get-DescendantProcessIds {
    param([int]$RootPid)

    $all = New-Object System.Collections.Generic.List[int]
    $queue = New-Object System.Collections.Generic.Queue[int]
    $queue.Enqueue($RootPid)

    while ($queue.Count -gt 0) {
        $current = $queue.Dequeue()
        $children = Get-CimInstance Win32_Process -Filter "ParentProcessId=$current" -ErrorAction SilentlyContinue
        foreach ($child in $children) {
            $childId = [int]$child.ProcessId
            if (-not $all.Contains($childId)) {
                $all.Add($childId)
                $queue.Enqueue($childId)
            }
        }
    }

    return @($all)
}

function Get-TreeWorkingSet {
    param([int]$RootPid)

    $pids = @($RootPid) + @(Get-DescendantProcessIds -RootPid $RootPid)
    $sum = 0
    foreach ($procId in $pids) {
        $proc = Get-Process -Id $procId -ErrorAction SilentlyContinue
        if ($proc) {
            $sum += [int64]$proc.WorkingSet64
        }
    }
    return [int64]$sum
}

function Invoke-HiddenCommand {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Name,
        [Parameter(Mandatory=$true)][string]$FilePath,
        [Parameter(Mandatory=$true)][string[]]$ArgumentList,
        [string]$WorkingDirectory = '',
        [int]$TimeoutSec = 300,
        [int[]]$ExpectedExitCodes = @(0),
        [string]$ObservedExitCodePath = ''
    )

    if ($WorkingDirectory -eq '') {
        $WorkingDirectory = $Ctx.RepoRoot
    }

    $safeName = $Name -replace '[^A-Za-z0-9_.-]', '_'
    $stdoutPath = Join-Path $Ctx.LogsDir "${safeName}.stdout.log"
    $stderrPath = Join-Path $Ctx.LogsDir "${safeName}.stderr.log"
    $combinedPath = Join-Path $Ctx.LogsDir "${safeName}.log"

    $started = Get-Date
    $proc = Start-Process -FilePath $FilePath -ArgumentList $ArgumentList -WorkingDirectory $WorkingDirectory -WindowStyle Hidden -PassThru -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath

    $maxWorkingSet = 0
    $timedOut = $false
    while (-not $proc.HasExited) {
        $elapsed = (Get-Date) - $started
        if ($elapsed.TotalSeconds -ge $TimeoutSec) {
            $timedOut = $true
            break
        }

        $ws = Get-TreeWorkingSet -RootPid $proc.Id
        if ($ws -gt $maxWorkingSet) {
            $maxWorkingSet = $ws
        }
        Start-Sleep -Milliseconds 200
        $proc.Refresh()
    }

    if ($timedOut) {
        & taskkill.exe /PID $proc.Id /T /F | Out-Null
    } else {
        $proc.WaitForExit()
        $proc.Refresh()
    }

    $ended = Get-Date
    $hostProcessExitCode = if ($timedOut) { 124 } else { [int]$proc.ExitCode }
    $exitCode = $hostProcessExitCode
    $exitCodeSource = 'host_process_exit_code'
    if ((-not $timedOut) -and ($ObservedExitCodePath -ne '') -and (Test-Path -LiteralPath $ObservedExitCodePath)) {
        $sidecarRaw = Get-Content -LiteralPath $ObservedExitCodePath -Raw -Encoding UTF8
        $sidecarValue = [string]$sidecarRaw
        if ($sidecarValue -match '^\s*[-]?\d+\s*$') {
            $exitCode = [int]$sidecarValue.Trim()
            $exitCodeSource = 'native_child_sidecar'
        }
    }

    if (Test-Path -LiteralPath $stdoutPath) {
        $stdout = Get-Content -LiteralPath $stdoutPath -Raw -Encoding UTF8
    } else {
        $stdout = ''
    }
    if (Test-Path -LiteralPath $stderrPath) {
        $stderr = Get-Content -LiteralPath $stderrPath -Raw -Encoding UTF8
    } else {
        $stderr = ''
    }

    @(
        "name: $Name"
        "working_directory: $WorkingDirectory"
        "file_path: $FilePath"
        "arguments: $($ArgumentList -join ' ')"
        "expected_exit_codes: $($ExpectedExitCodes -join ',')"
        "host_process_exit_code: $hostProcessExitCode"
        "exit_code: $exitCode"
        "exit_code_source: $exitCodeSource"
        "observed_exit_code_path: $ObservedExitCodePath"
        "timed_out: $timedOut"
        "max_tree_working_set_bytes: $maxWorkingSet"
        "started_at: $($started.ToString('o'))"
        "ended_at: $($ended.ToString('o'))"
        ''
        'stdout:'
        $stdout
        ''
        'stderr:'
        $stderr
    ) | Set-Content -LiteralPath $combinedPath -Encoding UTF8

    $record = [ordered]@{
        name = $Name
        file_path = $FilePath
        arguments = $ArgumentList
        working_directory = $WorkingDirectory
        expected_exit_codes = $ExpectedExitCodes
        host_process_exit_code = $hostProcessExitCode
        exit_code = $exitCode
        exit_code_source = $exitCodeSource
        timed_out = $timedOut
        max_tree_working_set_bytes = $maxWorkingSet
        started_at = $started.ToString('o')
        ended_at = $ended.ToString('o')
        duration_ms = [int](($ended - $started).TotalMilliseconds)
        log = "logs/${safeName}.log"
    }

    ($record | ConvertTo-Json -Compress) | Add-Content -LiteralPath $Ctx.CommandsPath -Encoding UTF8
    $Ctx.CommandResults[$Name] = $record

    $expected = $false
    foreach ($code in $ExpectedExitCodes) {
        if ($exitCode -eq $code) {
            $expected = $true
            break
        }
    }

    return [ordered]@{
        ExitCode = $exitCode
        HostProcessExitCode = $hostProcessExitCode
        ExitCodeSource = $exitCodeSource
        Expected = $expected
        TimedOut = $timedOut
        MaxWorkingSet = $maxWorkingSet
        Stdout = $stdout
        Stderr = $stderr
        Log = "logs/${safeName}.log"
        DurationMs = [int](($ended - $started).TotalMilliseconds)
    }
}

function Invoke-HiddenPowerShell {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Name,
        [Parameter(Mandatory=$true)][string]$Command,
        [string]$WorkingDirectory = '',
        [int]$TimeoutSec = 300,
        [int[]]$ExpectedExitCodes = @(0)
    )

    $sidecarPath = Join-Path $Ctx.TmpDir ("native_exit.{0}.exitcode" -f [guid]::NewGuid().ToString('N'))
    $wrapped = @"
`$ErrorActionPreference = 'Stop'
`$__nativeExit = 0
try {
$Command
    if (`$null -ne `$LASTEXITCODE) {
        `$__nativeExit = [int]`$LASTEXITCODE
    } elseif (`$?) {
        `$__nativeExit = 0
    } else {
        `$__nativeExit = 1
    }
} catch {
    `$__nativeExit = 1
}
[System.IO.File]::WriteAllText('$sidecarPath', [string]`$__nativeExit, [Text.UTF8Encoding]::new(`$false))
Write-Output ('NATIVE_CHILD_EXIT_CODE=' + `$__nativeExit)
exit 0
"@
    return Invoke-HiddenCommand -Ctx $Ctx -Name $Name -FilePath 'powershell.exe' -ArgumentList @('-NoProfile','-ExecutionPolicy','Bypass','-Command', $wrapped) -WorkingDirectory $WorkingDirectory -TimeoutSec $TimeoutSec -ExpectedExitCodes $ExpectedExitCodes -ObservedExitCodePath $sidecarPath
}

function Add-Assessment {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [Parameter(Mandatory=$true)][string]$Id,
        [Parameter(Mandatory=$true)][ValidateSet('minimum','good','excellent','engineering')][string]$Level,
        [Parameter(Mandatory=$true)][bool]$Ok,
        [string]$Requirement = '',
        [string[]]$Evidence = @(),
        [string]$Details = ''
    )

    $item = [ordered]@{
        id = $Id
        level = $Level
        category = 'runtime'
        requirement = if ($Requirement -ne '') { $Requirement } else { $Id }
        implementation = if ($Ok) { 'full' } else { 'partial' }
        conformance = if ($Ok) { 'conformant' } else { 'nonconformant' }
        evidence = @($Evidence)
        details = $Details
    }
    $Ctx.Assessments.Add($item) | Out-Null
}

function Get-JsonlObjects {
    param([Parameter(Mandatory=$true)][string]$Path)

    $objects = @()
    $lineNumber = 0
    foreach ($line in Get-Content -LiteralPath $Path -Encoding UTF8) {
        $lineNumber++
        if ([string]::IsNullOrWhiteSpace($line)) {
            continue
        }
        try {
            $objects += ($line | ConvertFrom-Json)
        } catch {
            throw ("failed to parse jsonl {0} line {1}: {2}" -f $Path, $lineNumber, $_.Exception.Message)
        }
    }
    return @($objects)
}

function Get-FileSha256 {
    param([Parameter(Mandatory=$true)][string]$Path)
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $stream = [System.IO.File]::OpenRead($Path)
        try {
            $hashBytes = $sha.ComputeHash($stream)
        } finally {
            $stream.Close()
        }
    } finally {
        $sha.Dispose()
    }
    return ([BitConverter]::ToString($hashBytes) -replace '-', '').ToLowerInvariant()
}

function Get-StringSha256Hex {
    param([Parameter(Mandatory=$true)][string]$Value)
    $bytes = [Text.Encoding]::UTF8.GetBytes($Value)
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $hashBytes = $sha.ComputeHash($bytes)
    } finally {
        $sha.Dispose()
    }
    return ([BitConverter]::ToString($hashBytes) -replace '-', '').ToLowerInvariant()
}

function Get-StableRecordId {
    param(
        [Parameter(Mandatory=$true)][string]$Domain,
        [Parameter(Mandatory=$true)][string]$SlashPath
    )
    $hex = Get-StringSha256Hex -Value ($Domain + ':' + $SlashPath)
    return ($Domain + '-' + $hex.Substring(0, 12))
}

function Write-FakeHelperSource {
    param([Parameter(Mandatory=$true)][string]$Path)

    $helper = @'
package main

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type stream struct {
	SampleRate string `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

type ffprobeOut struct {
	Streams []stream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "fixture" {
		if err := runFixture(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	name := strings.ToLower(filepath.Base(os.Args[0]))
	switch {
	case strings.Contains(name, "ffprobe"):
		if err := runFFProbe(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	case strings.Contains(name, "ffmpeg"):
		if err := runFFMpeg(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(9)
		}
	case strings.Contains(name, "whisper"):
		if err := runWhisper(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(7)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown mode")
		os.Exit(3)
	}
}

func appendLog(line string) {
	dir := os.Getenv("STTQ_FAKE_LOG")
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, "fake-tools.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line + "\n")
}

func runFFProbe() error {
	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[len(os.Args)-1]
	}
	lower := strings.ToLower(arg)
	dur := "1.000"
	if strings.Contains(lower, "c2") {
		dur = "1.200"
	}
	if strings.Contains(lower, "c3") {
		dur = "12.400"
	}
	if strings.Contains(lower, "c4") {
		dur = "0.900"
	}
	if strings.Contains(lower, "f1") {
		dur = "2.500"
	}
	if strings.Contains(lower, "f2") {
		dur = "3.000"
	}
	rate := "16000"
	if strings.Contains(lower, "prepared8") || strings.Contains(lower, "wav8") || strings.Contains(lower, "-8k") {
		rate = "8000"
	}
	out := ffprobeOut{Streams: []stream{{SampleRate: rate, Channels: 1}}}
	out.Format.Duration = dur
	payload, _ := json.Marshal(out)
	appendLog("ffprobe|" + strings.Join(os.Args[1:], " "))
	fmt.Println(string(payload))
	return nil
}

func runFFMpeg() error {
	args := os.Args[1:]
	appendLog("ffmpeg|" + strings.Join(args, " "))
	joined := strings.ToLower(strings.Join(args, " "))
	if !(strings.Contains(joined, "-ac 1") && strings.Contains(joined, "-c:a pcm_s16le")) {
		return fmt.Errorf("ffmpeg args missing mandatory values")
	}
	if !(strings.Contains(joined, "-ar 16000") || strings.Contains(joined, "-ar 8000")) {
		return fmt.Errorf("ffmpeg args missing sample rate")
	}

	if strings.Contains(joined, "fail") {
		return fmt.Errorf("forced ffmpeg failure")
	}
	if strings.Contains(joined, "slow") {
		time.Sleep(3 * time.Second)
	}

	if len(args) == 0 {
		return fmt.Errorf("empty ffmpeg args")
	}
	outPath := args[len(args)-1]
	rate := "16000"
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-ar" {
			rate = args[i+1]
		}
	}
	if _, err := strconv.Atoi(rate); err != nil {
		return fmt.Errorf("invalid sample rate %q", rate)
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte("wav:"+rate+"|"+strings.Join(args, "|")), 0o644)
}

func runWhisper() error {
	args := os.Args[1:]
	appendLog(fmt.Sprintf("whisper|pid=%d|%s", os.Getpid(), strings.Join(args, " ")))
	joined := strings.ToLower(strings.Join(args, " "))
	if strings.Contains(joined, "slow") {
		time.Sleep(3 * time.Second)
	}
	if strings.Contains(joined, "fail") {
		return fmt.Errorf("forced whisper failure")
	}

	audio := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--audio" {
			audio = args[i+1]
		}
	}
	base := strings.TrimSuffix(filepath.Base(audio), filepath.Ext(audio))
	text := "hyp-" + base
	switch strings.ToLower(base) {
	case "c1", "crowd_c1", "id-c1":
		text = "привет мир"
	case "c2", "crowd_c2", "id-c2":
		text = "ежик где ты"
	case "c3", "crowd_c3", "id-c3":
		text = "длинная запись"
	case "c4", "crowd_c4", "id-c4":
		text = "съешь еще этих мягких французских булок"
	case "f1", "farfield_f1", "id-f1":
		text = "мама мыла ламу"
	case "f2", "farfield_f2", "id-f2":
		text = "еж и енот"
	}
	fmt.Printf("%s\n", text)
	return nil
}

func runFixture() error {
	out := "."
	for i := 2; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--out" {
			out = os.Args[i+1]
		}
	}

	base := filepath.Join(out, "golos_dir")
	crowdAudio := filepath.Join(base, "crowd", "audio")
	farAudio := filepath.Join(base, "farfield", "audio")
	if err := os.MkdirAll(crowdAudio, 0o755); err != nil { return err }
	if err := os.MkdirAll(farAudio, 0o755); err != nil { return err }

	if err := os.WriteFile(filepath.Join(crowdAudio, "c1.wav"), []byte("crowd1"), 0o644); err != nil { return err }
	if err := os.WriteFile(filepath.Join(crowdAudio, "c2.wav"), []byte("crowd2"), 0o644); err != nil { return err }
	if err := os.WriteFile(filepath.Join(crowdAudio, "c3.wav"), []byte("crowd3"), 0o644); err != nil { return err }
	if err := os.WriteFile(filepath.Join(crowdAudio, "c4.wav"), []byte("crowd4"), 0o644); err != nil { return err }
	if err := os.WriteFile(filepath.Join(farAudio, "f1.wav"), []byte("far1"), 0o644); err != nil { return err }
	if err := os.WriteFile(filepath.Join(farAudio, "f2.wav"), []byte("far2"), 0o644); err != nil { return err }

	crowdManifest := filepath.Join(base, "crowd", "manifest.jsonl")
	farManifest := filepath.Join(base, "farfield", "manifest.jsonl")

	crowdLines := []string{
		`{"audio_filepath":"audio/c1.wav","text":"Привет, Мир!","duration":1.0}`,
		`{"audio_filepath":"audio/c2.wav","text":"Ёжик, где\tты?","duration":1.2}`,
		`{"audio_filepath":"audio/c3.wav","text":"Длинная запись!!!","duration":12.4}`,
		`{"audio_filepath":"audio/c4.wav","text":"Съешь ещё этих мягких французских булок.","duration":0.9}`,
	}
	farLines := []string{
		`{"audio_filepath":"audio/f1.wav","text":"мама мыла раму","duration":2.5}`,
		`{"audio_filepath":"audio/f2.wav","text":"ёж и енот","duration":3.0}`,
	}
	if err := os.WriteFile(crowdManifest, []byte(strings.Join(crowdLines, "\n")+"\n"), 0o644); err != nil { return err }
	if err := os.WriteFile(farManifest, []byte(strings.Join(farLines, "\n")+"\n"), 0o644); err != nil { return err }

	unicodeBase := filepath.Join(out, "корпус с пробелами")
	if err := os.RemoveAll(unicodeBase); err != nil { return err }
	if err := copyDir(base, unicodeBase); err != nil { return err }

	tarPath := filepath.Join(out, "test.tar")
	tarFile, err := os.Create(tarPath)
	if err != nil { return err }
	defer tarFile.Close()
	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	files := []string{
		filepath.Join(base, "crowd", "audio", "c3.wav"),
		filepath.Join(base, "crowd", "audio", "c2.wav"),
		filepath.Join(base, "farfield", "audio", "f2.wav"),
		filepath.Join(base, "crowd", "manifest.jsonl"),
		filepath.Join(base, "farfield", "audio", "f1.wav"),
		filepath.Join(base, "crowd", "audio", "c4.wav"),
		filepath.Join(base, "crowd", "audio", "c1.wav"),
		filepath.Join(base, "farfield", "manifest.jsonl"),
	}

	for _, source := range files {
		data, err := os.ReadFile(source)
		if err != nil { return err }
		rel, err := filepath.Rel(base, source)
		if err != nil { return err }
		header := &tar.Header{Name: filepath.ToSlash(filepath.Join("rootprefix", rel)), Mode: 0o644, Size: int64(len(data))}
		if err := tw.WriteHeader(header); err != nil { return err }
		if _, err := tw.Write(data); err != nil { return err }
	}

	return nil
}

func copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, payload, 0o644)
	})
}
'@

    Set-Content -LiteralPath $Path -Value $helper -Encoding UTF8
}

function Get-PassedTestsFromJson {
    param([Parameter(Mandatory=$true)][string]$Path)
    $passed = @{}
    if (-not (Test-Path -LiteralPath $Path)) {
        return $passed
    }
    foreach ($line in Get-Content -LiteralPath $Path -Encoding UTF8) {
        if ([string]::IsNullOrWhiteSpace($line)) {
            continue
        }
        try {
            $event = $line | ConvertFrom-Json
        } catch {
            continue
        }
        $action = $null
        if ($event.PSObject.Properties.Match('Action').Count -gt 0) {
            $action = [string]$event.Action
        }
        $testName = $null
        if ($event.PSObject.Properties.Match('Test').Count -gt 0) {
            $testName = [string]$event.Test
        }
        if ($action -eq 'pass' -and -not [string]::IsNullOrWhiteSpace($testName)) {
            $passed[$testName] = $true
        }
    }
    return $passed
}

function Get-BenchmarkPresence {
    param([Parameter(Mandatory=$true)][string]$BenchLogText)
    $required = @('BenchmarkNormalization', 'BenchmarkWordAlignment', 'BenchmarkCharacterAlignment', 'BenchmarkJSONL', 'BenchmarkReport')
    $result = @{}
    foreach ($name in $required) {
        $pattern = [Regex]::Escape($name) + '\S*\s+\d+\s+[\d\.]+\s+ns/op'
        $result[$name] = [bool]([regex]::Match($BenchLogText, $pattern).Success) -and ($BenchLogText -match "$name.*B/op") -and ($BenchLogText -match "$name.*allocs/op")
    }
    return $result
}

function Get-RelativePathSafe {
    param(
        [Parameter(Mandatory=$true)][string]$BasePath,
        [Parameter(Mandatory=$true)][string]$TargetPath
    )
    $baseResolved = (Resolve-Path -LiteralPath $BasePath).Path
    $targetResolved = (Resolve-Path -LiteralPath $TargetPath).Path
    $baseUri = New-Object System.Uri(($baseResolved.TrimEnd('\') + '\'))
    $targetUri = New-Object System.Uri($targetResolved)
    return [System.Uri]::UnescapeDataString($baseUri.MakeRelativeUri($targetUri).ToString()).Replace('/', '\')
}

function Complete-Check {
    param(
        [Parameter(Mandatory=$true)]$Ctx,
        [hashtable]$Extra = @{}
    )

    $summary = [ordered]@{}
    foreach ($level in @('minimum','good','excellent','engineering')) {
        $items = @($Ctx.Assessments | Where-Object { $_.level -eq $level })
        $summary[$level] = [ordered]@{
            total = $items.Count
            full = @($items | Where-Object { $_.implementation -eq 'full' }).Count
            partial = @($items | Where-Object { $_.implementation -eq 'partial' }).Count
            not_implemented = @($items | Where-Object { $_.implementation -eq 'not_implemented' }).Count
            conformant = @($items | Where-Object { $_.conformance -eq 'conformant' }).Count
            nonconformant = @($items | Where-Object { $_.conformance -eq 'nonconformant' }).Count
            not_tested = @($items | Where-Object { $_.conformance -eq 'not_tested' }).Count
        }
    }

    Save-Json -Path (Join-Path $Ctx.ResultDir 'assessment.json') -Value ([ordered]@{
        schema_version = 2
        statuses = [ordered]@{
            implementation = @('not_implemented','partial','full')
            conformance = @('not_tested','nonconformant','conformant')
        }
        summary = $summary
        features = @($Ctx.Assessments)
    })

    Save-Json -Path (Join-Path $Ctx.ResultDir 'manifest.json') -Value ([ordered]@{
        student = 'anastasia_gromova_stub_stage2'
        repo_root = $Ctx.RepoRoot
        started_at = $Ctx.StartedAt
        completed_at = (Get-Date).ToString('o')
        go = (& $Ctx.GoCmd version)
        notes = $Extra
    })

    Invoke-HiddenPowerShell -Ctx $Ctx -Name 'meta_git_head' -Command "git rev-parse HEAD | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_head.txt' -Encoding UTF8" | Out-Null
    Invoke-HiddenPowerShell -Ctx $Ctx -Name 'meta_git_status' -Command "git status --short | Set-Content -LiteralPath '$($Ctx.MetaDir)\git_status_short.txt' -Encoding UTF8" | Out-Null
    Invoke-HiddenPowerShell -Ctx $Ctx -Name 'meta_go_version' -Command "& '$($Ctx.GoCmd)' version | Set-Content -LiteralPath '$($Ctx.MetaDir)\go_version.txt' -Encoding UTF8" | Out-Null

    $zipPath = "$($Ctx.ResultDir).zip"
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    Compress-Archive -Path (Join-Path $Ctx.ResultDir '*') -DestinationPath $zipPath -Force

    Write-Host "CHECK_RESULT_DIR=$($Ctx.ResultDir)"
    Write-Host "CHECK_RESULT_ZIP=$zipPath"
    return $zipPath
}

$ctx = New-CheckContext -RepoRoot $RepoRoot -OutRoot $OutRoot
$root = $ctx.RepoRoot
$checks = @{}

$ids = @(
    @{ id='minimum.go120_build'; level='minimum' },
    @{ id='minimum.cross_platform_paths'; level='minimum' },
    @{ id='minimum.golos_import'; level='minimum' },
    @{ id='minimum.transcripts_preserved'; level='minimum' },
    @{ id='minimum.stable_ids'; level='minimum' },
    @{ id='minimum.quotas'; level='minimum' },
    @{ id='minimum.deterministic_selection'; level='minimum' },
    @{ id='minimum.manifest_validation'; level='minimum' },
    @{ id='minimum.audio_integrity_probe'; level='minimum' },
    @{ id='minimum.audio_profiles'; level='minimum' },
    @{ id='minimum.normalization_profiles'; level='minimum' },
    @{ id='minimum.wer_cer'; level='minimum' },
    @{ id='minimum.alignment'; level='minimum' },
    @{ id='minimum.aggregate_wer'; level='minimum' },

    @{ id='good.coverage_rtf'; level='good' },
    @{ id='good.grouped_metrics'; level='good' },
    @{ id='good.deterministic_json'; level='good' },
    @{ id='good.html_details'; level='good' },
    @{ id='good.compare_exit_codes'; level='good' },
    @{ id='good.whisper_adapter'; level='good' },
    @{ id='good.runner_controls'; level='good' },

    @{ id='excellent.unit_tests'; level='excellent' },
    @{ id='excellent.golden_benchmarks'; level='excellent' },
    @{ id='excellent.contextual_errors'; level='excellent' },
    @{ id='excellent.atomic_files'; level='excellent' },
    @{ id='excellent.offline_demo'; level='excellent' },
    @{ id='excellent.readme_complete'; level='excellent' },

    @{ id='engineering.unit_tests_present'; level='engineering' },
    @{ id='engineering.benchmarks_present'; level='engineering' },
    @{ id='engineering.go_test_passes'; level='engineering' },
    @{ id='engineering.make_test_runs'; level='engineering' },
    @{ id='engineering.make_bench_runs'; level='engineering' },
    @{ id='engineering.make_demo_runs'; level='engineering' },
    @{ id='engineering.readme'; level='engineering' },
    @{ id='engineering.make_test'; level='engineering' },
    @{ id='engineering.make_bench'; level='engineering' },
    @{ id='engineering.make_demo'; level='engineering' },
    @{ id='engineering.control_data'; level='engineering' },
    @{ id='engineering.solution_doc'; level='engineering' }
)

foreach ($item in $ids) {
    $checks[$item.id] = $false
}

$sttqExe = Join-Path $ctx.OutputsDir 'sttq.exe'
$fakeSource = Join-Path $ctx.TmpDir 'fake_helper.go'
$fakeExe = Join-Path $ctx.TmpDir 'fake_tool.exe'
$fakeFFProbe = Join-Path $ctx.TmpDir 'ffprobe.exe'
$fakeFFMpeg = Join-Path $ctx.TmpDir 'ffmpeg.exe'
$fakeWhisper = Join-Path $ctx.TmpDir 'whisper-cli.exe'
$fakeLogDir = Join-Path $ctx.OutputsDir 'fake_logs'
New-Item -ItemType Directory -Force -Path $fakeLogDir | Out-Null

# Core checks.
$goVersion = & $ctx.GoCmd version
[System.IO.File]::WriteAllText((Join-Path $ctx.MetaDir 'go_version.txt'), ($goVersion + [Environment]::NewLine), [Text.UTF8Encoding]::new($false))
$checks['minimum.go120_build'] = ($goVersion -match 'go1\.20\.[0-9]+') -and ($goVersion -match 'windows')

$buildRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'build_cli' -Command "& '$($ctx.GoCmd)' build -mod=vendor -o '$sttqExe' ./cmd/sttq" -WorkingDirectory $root -TimeoutSec 180
$linuxExe = Join-Path $ctx.TmpDir 'sttq-linux'
$buildLinuxRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'build_cli_linux' -Command "`$env:GOOS='linux'; `$env:GOARCH='amd64'; `$env:CGO_ENABLED='0'; & '$($ctx.GoCmd)' build -mod=vendor -o '$linuxExe' ./cmd/sttq" -WorkingDirectory $root -TimeoutSec 240
$checks['minimum.go120_build'] = $checks['minimum.go120_build'] -and $buildRes.Expected -and $buildLinuxRes.Expected

Write-FakeHelperSource -Path $fakeSource
$buildHelperRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'build_fake_helper' -Command "& '$($ctx.GoCmd)' build -o '$fakeExe' '$fakeSource'" -WorkingDirectory $root -TimeoutSec 120
if ($buildHelperRes.Expected) {
    Copy-Item -LiteralPath $fakeExe -Destination $fakeFFProbe -Force
    Copy-Item -LiteralPath $fakeExe -Destination $fakeFFMpeg -Force
    Copy-Item -LiteralPath $fakeExe -Destination $fakeWhisper -Force
}

$fixtureRoot = Join-Path $ctx.InputsDir 'fixtures'
$fixtureRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'generate_fixtures' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$fakeExe' fixture --out '$fixtureRoot'" -WorkingDirectory $root -TimeoutSec 120

$golosDir = Join-Path $fixtureRoot 'golos_dir'
$golosTar = Join-Path $fixtureRoot 'test.tar'
$golosUnicode = Join-Path $fixtureRoot 'корпус с пробелами'

$importDirOut = Join-Path $ctx.OutputsDir 'import_dir'
$importTarOut = Join-Path $ctx.OutputsDir 'import_tar'
$importUnicodeOut = Join-Path $ctx.OutputsDir 'import_unicode'
$importDiffSeedOut = Join-Path $ctx.OutputsDir 'import_seed_alt'

$importDirRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'import_dir' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' corpus import-golos --source '$golosDir' --out '$importDirOut' --seed stage2-seed --quota crowd=2 --quota farfield=1 --ffprobe '$fakeFFProbe'" -WorkingDirectory $root -TimeoutSec 180
$importTarRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'import_tar' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' corpus import-golos --source '$golosTar' --out '$importTarOut' --seed stage2-seed --quota crowd=2 --quota farfield=1 --ffprobe '$fakeFFProbe'" -WorkingDirectory $root -TimeoutSec 180
$importUnicodeRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'import_unicode_path' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' corpus import-golos --source '$golosUnicode' --out '$importUnicodeOut' --seed stage2-seed --quota crowd=2 --quota farfield=1 --ffprobe '$fakeFFProbe'" -WorkingDirectory $root -TimeoutSec 180

$checks['minimum.golos_import'] = $importDirRes.Expected -and $importTarRes.Expected -and $importUnicodeRes.Expected

$dirManifest = Join-Path $importDirOut 'manifest.jsonl'
$tarManifest = Join-Path $importTarOut 'manifest.jsonl'
$unicodeManifest = Join-Path $importUnicodeOut 'manifest.jsonl'

if ((Test-Path -LiteralPath $dirManifest) -and (Test-Path -LiteralPath $tarManifest) -and (Test-Path -LiteralPath $unicodeManifest)) {
    $dirItems = Get-JsonlObjects -Path $dirManifest
    $tarItems = Get-JsonlObjects -Path $tarManifest
    $unicodeItems = Get-JsonlObjects -Path $unicodeManifest
    $checks['minimum.quotas'] = (@($dirItems | Where-Object { $_.tags[0] -eq 'crowd' }).Count -eq 2) -and (@($dirItems | Where-Object { $_.tags[0] -eq 'farfield' }).Count -eq 1)
    $checks['minimum.stable_ids'] = (($dirItems.id -join ',') -eq ($tarItems.id -join ',')) -and (($dirItems.id -join ',') -eq ($unicodeItems.id -join ','))
    $checks['minimum.cross_platform_paths'] = (@($dirItems | Where-Object { $_.audio -match '\\' }).Count -eq 0) -and (@($dirItems | Where-Object { $_.audio -match '^[a-zA-Z]:' }).Count -eq 0)

    $sourceByID = @{}
    foreach ($domain in @('crowd', 'farfield')) {
        $manifestPath = Join-Path (Join-Path $golosDir $domain) 'manifest.jsonl'
        foreach ($line in Get-Content -LiteralPath $manifestPath -Encoding UTF8) {
            if ([string]::IsNullOrWhiteSpace($line)) { continue }
            $obj = $line | ConvertFrom-Json
            $slashPath = ($domain + '/' + $obj.audio_filepath).Replace('\', '/')
            $id = Get-StableRecordId -Domain $domain -SlashPath $slashPath
            $sourceByID[$id] = [string]$obj.text
        }
    }
    $transcriptPreserved = $true
    foreach ($item in $dirItems) {
        if (-not $sourceByID.ContainsKey($item.id)) {
            $transcriptPreserved = $false
            break
        }
        if ([string]$item.text -ne [string]$sourceByID[$item.id]) {
            $transcriptPreserved = $false
            break
        }
    }
    $checks['minimum.transcripts_preserved'] = $transcriptPreserved
    $checks['minimum.stable_ids'] = $checks['minimum.stable_ids'] -and $transcriptPreserved
}

$importDirOut2 = Join-Path $ctx.OutputsDir 'import_dir_repeat'
$importRepeatRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'import_dir_repeat' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' corpus import-golos --source '$golosDir' --out '$importDirOut2' --seed stage2-seed --quota crowd=2 --quota farfield=1 --ffprobe '$fakeFFProbe'" -WorkingDirectory $root -TimeoutSec 180
$importDiffSeedRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'import_dir_different_seed' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' corpus import-golos --source '$golosDir' --out '$importDiffSeedOut' --seed stage2-seed-other --quota crowd=2 --quota farfield=1 --ffprobe '$fakeFFProbe'" -WorkingDirectory $root -TimeoutSec 180
if ($importRepeatRes.Expected) {
    $sameSelection = (Get-FileSha256 -Path (Join-Path $importDirOut 'source\selection.json')) -eq (Get-FileSha256 -Path (Join-Path $importDirOut2 'source\selection.json'))
    $differentSelection = $false
    if ($importDiffSeedRes.Expected) {
        $differentSelection = (Get-FileSha256 -Path (Join-Path $importDirOut 'source\selection.json')) -ne (Get-FileSha256 -Path (Join-Path $importDiffSeedOut 'source\selection.json'))
    }
    $checks['minimum.deterministic_selection'] = $sameSelection -and $differentSelection
}

$validateGoodRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'validate_good' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' corpus validate --manifest '$dirManifest' --ffprobe '$fakeFFProbe'" -WorkingDirectory $root -TimeoutSec 180
$statsRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'stats_good' -Command "& '$sttqExe' corpus stats --manifest '$dirManifest'" -WorkingDirectory $root -TimeoutSec 120

$invalidManifest = Join-Path $ctx.InputsDir 'invalid_manifest.jsonl'
$invalidAudioDir = Join-Path $ctx.InputsDir 'audio'
New-Item -ItemType Directory -Force -Path $invalidAudioDir | Out-Null
$presentAudio = Join-Path $invalidAudioDir 'present.wav'
[System.IO.File]::WriteAllBytes($presentAudio, [byte[]](1,2,3,4,5,6))
$presentAudio2 = Join-Path $invalidAudioDir 'present2.wav'
[System.IO.File]::WriteAllBytes($presentAudio2, [byte[]](9,8,7,6,5,4))
$invalidBuilder = New-Object System.Text.StringBuilder
@(
    '{"id":"z","audio":"../outside.wav","text":"x","language":"ru","duration_ms":0,"sample_rate":0,"channels":0,"tags":["b","a"],"sha256":"00"}',
    '{"id":"a","audio":"audio/present.wav","text":"x","language":"ru","duration_ms":1,"sample_rate":1,"channels":1,"tags":["ok","ok"],"sha256":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}',
    '{"id":"a","audio":"audio/present.wav","text":"x","language":"ru","duration_ms":1,"sample_rate":1,"channels":1,"tags":["ok"],"sha256":"ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"}',
    '{"id":"n","audio":"audio/present2.wav","text":"x","language":"ru","duration_ms":0,"sample_rate":0,"channels":0,"tags":["z","y"],"sha256":"11"}',
    '{"id":"m","audio":"audio/missing.wav","text":"x","language":"ru","duration_ms":1,"sample_rate":1,"channels":1,"tags":["ok"],"sha256":"11"}',
    'not-json'
) | ForEach-Object { $null = $script:invalidBuilder.AppendLine($_) }
[System.IO.File]::WriteAllText($invalidManifest, $script:invalidBuilder.ToString(), [Text.UTF8Encoding]::new($false))

$invalidManifestRel = Get-RelativePathSafe -BasePath $root -TargetPath $invalidManifest
$validateBadCommand = "& '$sttqExe' corpus validate --manifest '$invalidManifestRel' --ffprobe '$fakeFFProbe'"
$validateBadRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'validate_bad' -Command $validateBadCommand -WorkingDirectory $root -TimeoutSec 120 -ExpectedExitCodes @(2)
$validateBadRepeatRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'validate_bad_repeat' -Command $validateBadCommand -WorkingDirectory $root -TimeoutSec 120 -ExpectedExitCodes @(2)
$validateBadObserved = ($validateBadRes.ExitCode -eq 2) -and ($validateBadRes.Stderr -match 'invalid json') -and ($validateBadRes.Stderr -match 'duplicate id') -and ($validateBadRes.Stderr -match 'reuses audio') -and ($validateBadRes.Stderr -match 'unsafe audio path') -and ($validateBadRes.Stderr -match 'audio .* is missing') -and ($validateBadRes.Stderr -match 'checksum mismatch') -and ($validateBadRes.Stderr -match 'duration_ms must be positive') -and ($validateBadRes.Stderr -match 'sample_rate must be positive') -and ($validateBadRes.Stderr -match 'channels must be positive') -and ($validateBadRes.Stderr -match 'tags must be sorted and unique') -and ($validateBadRes.Stderr -match 'manifest is not sorted by id')
$validateBadRepeatObserved = ($validateBadRepeatRes.ExitCode -eq 2) -and ($validateBadRepeatRes.Stderr -match 'invalid json') -and ($validateBadRepeatRes.Stderr -match 'duplicate id') -and ($validateBadRepeatRes.Stderr -match 'reuses audio') -and ($validateBadRepeatRes.Stderr -match 'unsafe audio path') -and ($validateBadRepeatRes.Stderr -match 'audio .* is missing') -and ($validateBadRepeatRes.Stderr -match 'checksum mismatch') -and ($validateBadRepeatRes.Stderr -match 'duration_ms must be positive') -and ($validateBadRepeatRes.Stderr -match 'sample_rate must be positive') -and ($validateBadRepeatRes.Stderr -match 'channels must be positive') -and ($validateBadRepeatRes.Stderr -match 'tags must be sorted and unique') -and ($validateBadRepeatRes.Stderr -match 'manifest is not sorted by id')
$validateBadLinesA = @($validateBadRes.Stderr -split "`r?`n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
$validateBadLinesB = @($validateBadRepeatRes.Stderr -split "`r?`n" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
$validateBadStable = ($validateBadRes.ExitCode -eq 2) -and ($validateBadRepeatRes.ExitCode -eq 2) -and ((Get-StringSha256Hex -Value ($validateBadLinesA -join "`n")) -eq (Get-StringSha256Hex -Value ($validateBadLinesB -join "`n"))) -and (($validateBadLinesA -join "`n") -eq ($validateBadLinesB -join "`n"))
$checks['minimum.manifest_validation'] = $validateGoodRes.Expected -and $validateBadObserved -and $validateBadRepeatObserved

$prepared16Out = Join-Path $ctx.OutputsDir 'prepared16'
$prepared8Out = Join-Path $ctx.OutputsDir 'prepared8'
$prepare16Res = Invoke-HiddenPowerShell -Ctx $ctx -Name 'audio_prepare_wav16' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' audio prepare --manifest '$dirManifest' --out '$prepared16Out' --profile wav-16k --workers 2 --timeout 30s --ffmpeg '$fakeFFMpeg' --ffprobe '$fakeFFProbe'" -WorkingDirectory $root -TimeoutSec 180
$prepare8Res = Invoke-HiddenPowerShell -Ctx $ctx -Name 'audio_prepare_wav8' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' audio prepare --manifest '$dirManifest' --out '$prepared8Out' --profile wav-8k --workers 2 --timeout 30s --ffmpeg '$fakeFFMpeg' --ffprobe '$fakeFFProbe'" -WorkingDirectory $root -TimeoutSec 180
$checks['minimum.audio_integrity_probe'] = $prepare16Res.Expected -and $prepare8Res.Expected

$fakeToolsLog = Join-Path $fakeLogDir 'fake-tools.log'
if (Test-Path -LiteralPath $fakeToolsLog) {
    $logText = Get-Content -LiteralPath $fakeToolsLog -Raw -Encoding UTF8
    $checks['minimum.audio_profiles'] = ($logText -match '-ar 16000') -and ($logText -match '-ar 8000') -and ($logText -match '-ac 1') -and ($logText -match '-c:a pcm_s16le')
}

$preparedManifest = Join-Path $prepared16Out 'manifest.jsonl'
$runOut = Join-Path $ctx.OutputsDir 'run_whisper.jsonl'
$runRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'run_whisper' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' run whispercpp --manifest '$preparedManifest' --binary '$fakeWhisper' --model '$($ctx.InputsDir)\fake.bin' --language ru --workers 2 --timeout 20s --out '$runOut'" -WorkingDirectory $root -TimeoutSec 180
$resumeRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'run_whisper_resume' -Command "`$env:STTQ_FAKE_LOG='$fakeLogDir'; & '$sttqExe' run whispercpp --manifest '$preparedManifest' --binary '$fakeWhisper' --model '$($ctx.InputsDir)\fake.bin' --language ru --workers 2 --timeout 20s --resume --out '$runOut'" -WorkingDirectory $root -TimeoutSec 180
$checks['good.whisper_adapter'] = $runRes.Expected -and ($runRes.ExitCode -eq 0)
$checks['good.runner_controls'] = $resumeRes.Expected -and ($resumeRes.ExitCode -eq 0)
if (Test-Path -LiteralPath $fakeToolsLog) {
    $whisperLogText = Get-Content -LiteralPath $fakeToolsLog -Raw -Encoding UTF8
    $checks['good.whisper_adapter'] = $checks['good.whisper_adapter'] -and ($whisperLogText -match 'whisper\|') -and ($whisperLogText -match '--model') -and ($whisperLogText -match '--language ru') -and ($whisperLogText -match '--audio')
}

$eval1 = Join-Path $ctx.OutputsDir 'eval1.json'
$eval2 = Join-Path $ctx.OutputsDir 'eval2.json'
$evalRes1 = Invoke-HiddenPowerShell -Ctx $ctx -Name 'evaluate_1' -Command "& '$sttqExe' evaluate --manifest '$preparedManifest' --hypotheses '$runOut' --normalization ru-default --out '$eval1'" -WorkingDirectory $root -TimeoutSec 180
$evalRes2 = Invoke-HiddenPowerShell -Ctx $ctx -Name 'evaluate_2' -Command "& '$sttqExe' evaluate --manifest '$preparedManifest' --hypotheses '$runOut' --normalization ru-default --out '$eval2'" -WorkingDirectory $root -TimeoutSec 180

if ($evalRes1.Expected -and (Test-Path -LiteralPath $eval1)) {
    $evalObj = Get-Content -LiteralPath $eval1 -Raw -Encoding UTF8 | ConvertFrom-Json
    $checks['minimum.normalization_profiles'] = ([string]$evalObj.normalization -eq 'ru-default') -and (@($evalObj.records | Where-Object { ($null -ne $_.normalized_reference) -and ($null -ne $_.normalized_hypothesis) }).Count -gt 0)
    $checks['minimum.wer_cer'] = ($null -ne $evalObj.summary.wer) -and ($null -ne $evalObj.summary.cer)
    $checks['minimum.alignment'] = (@($evalObj.records | Where-Object { $_.alignment.Count -gt 0 }).Count -gt 0)
    $checks['minimum.aggregate_wer'] = ($evalObj.summary.counts.reference_n -ge 0)
    $checks['good.coverage_rtf'] = ($null -ne $evalObj.summary.coverage) -and ($null -ne $evalObj.summary.rtf)
    $checks['good.grouped_metrics'] = (@($evalObj.groups).Count -gt 0)
}

if ((Test-Path -LiteralPath $eval1) -and (Test-Path -LiteralPath $eval2)) {
    $checks['good.deterministic_json'] = (Get-FileSha256 -Path $eval1) -eq (Get-FileSha256 -Path $eval2)
}

$evalHtmlHyp = Join-Path $ctx.OutputsDir 'run_html.jsonl'
if (Test-Path -LiteralPath $runOut) {
    $runLines = Get-JsonlObjects -Path $runOut
    if ($runLines.Count -gt 0) {
        $runLines[0].text = '<script>alert(1)</script>'
        $buf = New-Object System.Text.StringBuilder
        foreach ($it in $runLines) {
            $null = $buf.AppendLine(($it | ConvertTo-Json -Compress))
        }
        [System.IO.File]::WriteAllText($evalHtmlHyp, $buf.ToString(), [Text.UTF8Encoding]::new($false))
    }
}

$evalHtmlJson = Join-Path $ctx.OutputsDir 'eval_html.json'
$evalHtmlRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'evaluate_html' -Command "& '$sttqExe' evaluate --manifest '$preparedManifest' --hypotheses '$evalHtmlHyp' --normalization strict --out '$evalHtmlJson'" -WorkingDirectory $root -TimeoutSec 180
$htmlOut = Join-Path $ctx.OutputsDir 'report.html'
$reportRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'report_html' -Command "& '$sttqExe' report --input '$evalHtmlJson' --format html --out '$htmlOut'" -WorkingDirectory $root -TimeoutSec 120

if ($reportRes.Expected -and (Test-Path -LiteralPath $htmlOut)) {
    $html = Get-Content -LiteralPath $htmlOut -Raw -Encoding UTF8
    $checks['good.html_details'] = ($html -match 'STTQ Report') -and ($html -match 'Groups') -and ($html -match 'Records') -and ($html -match '&lt;script&gt;') -and ($html -match 'equal|substitute|delete|insert') -and ($html -match 'Error')
}

$compareErrPath = Join-Path $ctx.OutputsDir 'missing.json'
$comparePassCommand = "& '$sttqExe' compare --baseline '$eval1' --current '$eval2' --max-wer-delta 0.02 --max-cer-delta 0.02"
$compareRegressionCommand = "& '$sttqExe' compare --baseline '$eval1' --current '$eval2' --max-wer-delta -0.0001 --max-cer-delta 0.02"
$compareErrorCommand = "& '$sttqExe' compare --baseline '$eval1' --current '$compareErrPath' --max-wer-delta 0.02 --max-cer-delta 0.02"
$comparePassRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'compare_pass' -Command $comparePassCommand -WorkingDirectory $root -TimeoutSec 120 -ExpectedExitCodes @(0)
$compareRegRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'compare_regression' -Command $compareRegressionCommand -WorkingDirectory $root -TimeoutSec 120 -ExpectedExitCodes @(1)
$compareErrRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'compare_error' -Command $compareErrorCommand -WorkingDirectory $root -TimeoutSec 120 -ExpectedExitCodes @(2)
$checks['good.compare_exit_codes'] = ($comparePassRes.ExitCode -eq 0) -and ($comparePassRes.Stdout -match 'Status: PASS') -and ($compareRegRes.ExitCode -eq 1) -and ($compareRegRes.Stdout -match 'Status: REGRESSION') -and ($compareErrRes.ExitCode -eq 2) -and ($compareErrRes.Stderr -match 'cannot find|cannot find the file|no such|open|read')

$requiredTests = @(
    'TestNormalizeStrict',
    'TestNormalizeRUDefaultNFC',
    'TestWordCharacterMetricsAndAlignment',
    'TestAggregateMetrics',
    'TestImportGolosDirectoryAndTar',
    'TestDeterministicSelectionBySeed',
    'TestValidateManifestAggregatesErrors',
    'TestAudioPrepareProfilesAndTimeout',
    'TestWhisperRunnerConcurrencyTimeoutResume',
    'TestReportGolden',
    'TestCoverageRTFAndGroupedMetrics',
    'TestCompareExitCodes',
    'TestAtomicFile'
)
$testPattern = ($requiredTests -join '|')
$testNamedJsonPath = Join-Path $ctx.OutputsDir 'go_test_named.jsonl'
$testNamedRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'go_test_named_json' -Command "& '$($ctx.GoCmd)' test -mod=vendor -count=1 -json -run '$testPattern' ./... | Set-Content -LiteralPath '$testNamedJsonPath' -Encoding UTF8" -WorkingDirectory $root -TimeoutSec 300
$testListPath = Join-Path $ctx.OutputsDir 'go_test_list.txt'
$testListRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'go_test_list' -Command "& '$($ctx.GoCmd)' test -mod=vendor -list 'Test|Benchmark' ./... | Set-Content -LiteralPath '$testListPath' -Encoding UTF8" -WorkingDirectory $root -TimeoutSec 180

$goFmtRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'go_fmt_all' -Command "& '$($ctx.GoCmd)' fmt ./..." -WorkingDirectory $root -TimeoutSec 300
$goTestRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'go_test_all' -Command "& '$($ctx.GoCmd)' test -mod=vendor ./..." -WorkingDirectory $root -TimeoutSec 300
$goVetRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'go_vet_all' -Command "& '$($ctx.GoCmd)' vet -mod=vendor ./..." -WorkingDirectory $root -TimeoutSec 300
$makeTestRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'make_test' -Command "make test" -WorkingDirectory $root -TimeoutSec 300
$makeBenchRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'make_bench' -Command "make bench" -WorkingDirectory $root -TimeoutSec 420
$makeDemoRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'make_demo' -Command "`$env:GOPROXY='off'; `$env:GOSUMDB='off'; `$env:HTTPS_PROXY='http://127.0.0.1:1'; `$env:HTTP_PROXY='http://127.0.0.1:1'; make demo" -WorkingDirectory $root -TimeoutSec 300

$passedTests = Get-PassedTestsFromJson -Path $testNamedJsonPath
$testPassed = @{}
foreach ($name in $requiredTests) {
    $testPassed[$name] = $passedTests.ContainsKey($name)
}
$allRequiredTestsPassed = $true
foreach ($name in $requiredTests) {
    if (-not $testPassed[$name]) {
        $allRequiredTestsPassed = $false
        break
    }
}
$checks['excellent.unit_tests'] = $testNamedRes.Expected -and $allRequiredTestsPassed
$benchLogPath = Join-Path $ctx.LogsDir 'make_bench.log'
$benchLogText = if (Test-Path -LiteralPath $benchLogPath) { Get-Content -LiteralPath $benchLogPath -Raw -Encoding UTF8 } else { '' }
$benchPresence = Get-BenchmarkPresence -BenchLogText $benchLogText
$allBenchPresent = $true
foreach ($item in $benchPresence.GetEnumerator()) {
    if (-not [bool]$item.Value) {
        $allBenchPresent = $false
        break
    }
}
$checks['excellent.golden_benchmarks'] = $makeBenchRes.Expected -and $testPassed['TestReportGolden'] -and (Test-Path -LiteralPath (Join-Path $root 'testdata\golden\evaluation.json')) -and $allBenchPresent
$validatePathLineContext = ($validateBadRes.Stderr -match ([regex]::Escape($invalidManifestRel) + ':[0-9]+:'))
$validateIdContext = ($validateBadRes.Stderr -match 'record "[^"]+"') -or ($validateBadRes.Stderr -match 'duplicate id "[^"]+"')
$compareErrorContext = ($compareErrRes.Stderr -match 'cannot find|cannot find the file|no such|open|read')
$checks['excellent.contextual_errors'] = $validateBadObserved -and $validateBadRepeatObserved -and $validateBadStable -and $validatePathLineContext -and $validateIdContext -and ($compareErrRes.ExitCode -eq 2) -and $compareErrorContext
$runtimeTempLeftovers = @()
foreach ($probePath in @($prepared16Out, $prepared8Out, $ctx.OutputsDir)) {
    if (-not (Test-Path -LiteralPath $probePath)) {
        continue
    }
    $runtimeTempLeftovers += @(Get-ChildItem -LiteralPath $probePath -Recurse -File -ErrorAction SilentlyContinue | Where-Object { $_.Name -match '\.tmp$' })
}
$checks['excellent.atomic_files'] = $testPassed['TestAtomicFile'] -and $testPassed['TestAudioPrepareProfilesAndTimeout'] -and ($runtimeTempLeftovers.Count -eq 0)
$checks['excellent.offline_demo'] = $makeDemoRes.Expected

$readmePath = Join-Path $root 'README.md'
$solutionPath = Join-Path $root 'docs\solution.md'
$readmeText = if (Test-Path -LiteralPath $readmePath) { (Get-Content -LiteralPath $readmePath -Raw -Encoding UTF8).ToLowerInvariant() } else { '' }
$solutionText = if (Test-Path -LiteralPath $solutionPath) { (Get-Content -LiteralPath $solutionPath -Raw -Encoding UTF8).ToLowerInvariant() } else { '' }

$readmeTopicPatterns = [ordered]@{
    purpose = 'purpose|stage2-only|toolkit'
    build = 'build|go\s+1\.20|make test|make bench'
    golos = 'golos'
    import = 'import-golos'
    manifest = 'manifest'
    quota = 'quota'
    audio = 'audio prepare|ffmpeg|ffprobe'
    hypotheses = 'hypotheses'
    wer_cer = 'wer|cer'
    normalization = 'normalization|strict|ru-default'
    cli = 'cli|sttq|cmd/sttq'
    report = 'report|html'
    baseline = 'baseline|compare'
    exit_codes = 'exit|0/1/2|pass|regression|error'
    demo = 'demo'
    limitations = 'limitations|does not train|not train'
    service_stub_status = 'stub'
}
$readmeMissingTopics = New-Object System.Collections.ArrayList
foreach ($topic in $readmeTopicPatterns.Keys) {
    if (-not ($readmeText -match $readmeTopicPatterns[$topic])) {
        $readmeMissingTopics.Add($topic) | Out-Null
    }
}
$checks['excellent.readme_complete'] = (Test-Path -LiteralPath $readmePath) -and ($readmeMissingTopics.Count -eq 0)

$testListText = if (Test-Path -LiteralPath $testListPath) { Get-Content -LiteralPath $testListPath -Raw -Encoding UTF8 } else { '' }
$checks['engineering.unit_tests_present'] = $testListRes.Expected -and $allRequiredTestsPassed
$checks['engineering.benchmarks_present'] = $allBenchPresent -and ($testListText -match 'BenchmarkNormalization') -and ($testListText -match 'BenchmarkWordAlignment') -and ($testListText -match 'BenchmarkCharacterAlignment') -and ($testListText -match 'BenchmarkJSONL') -and ($testListText -match 'BenchmarkReport')
$checks['engineering.go_test_passes'] = $goFmtRes.Expected -and $goTestRes.Expected -and $goVetRes.Expected
$checks['engineering.make_test_runs'] = $makeTestRes.Expected
$checks['engineering.make_bench_runs'] = $makeBenchRes.Expected
$checks['engineering.make_demo_runs'] = $makeDemoRes.Expected
$checks['engineering.readme'] = $checks['excellent.readme_complete']

$makefileText = if (Test-Path -LiteralPath (Join-Path $root 'Makefile')) { Get-Content -LiteralPath (Join-Path $root 'Makefile') -Raw -Encoding UTF8 } else { '' }
$checks['engineering.make_test'] = $makeTestRes.Expected
$checks['engineering.make_bench'] = $makeBenchRes.Expected
$checks['engineering.make_demo'] = $makeDemoRes.Expected
$demoCurrentPath = Join-Path $root 'demo\current.json'
$demoHtmlPath = Join-Path $root 'demo\report.html'
$demoControlOk = $false
if ($makeDemoRes.Expected -and (Test-Path -LiteralPath $demoCurrentPath) -and (Test-Path -LiteralPath $demoHtmlPath)) {
    try {
        $demoObj = Get-Content -LiteralPath $demoCurrentPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $demoHtmlText = Get-Content -LiteralPath $demoHtmlPath -Raw -Encoding UTF8
        $demoControlOk = ($null -ne $demoObj.summary) -and ($demoHtmlText -match '<html')
    } catch {
        $demoControlOk = $false
    }
}
$makeDemoLogPath = Join-Path $ctx.LogsDir 'make_demo.log'
$makeDemoLogText = if (Test-Path -LiteralPath $makeDemoLogPath) { Get-Content -LiteralPath $makeDemoLogPath -Raw -Encoding UTF8 } else { '' }
$checks['engineering.control_data'] = $demoControlOk -and (Test-Path -LiteralPath (Join-Path $root 'testdata\control\manifest.jsonl')) -and (Test-Path -LiteralPath (Join-Path $root 'testdata\control\hypotheses.jsonl')) -and ($makeDemoLogText -match 'testdata/control/manifest\.jsonl') -and ($makeDemoLogText -match 'testdata/control/hypotheses\.jsonl')
$solutionTopicPatterns = [ordered]@{
    architecture = 'architecture|internal/app|cmd/sttq'
    golos = 'golos'
    stable_id = 'stable|sha256|id'
    selection = 'selection|seed'
    model = 'model|whisper'
    levenshtein = 'levenshtein|tie-order|wer|cer'
    normalization = 'normaliz|nfc|ru-default|strict'
    process = 'process|os/exec|ffmpeg|ffprobe'
    determinism = 'determin|seed|stable'
    atomic = 'atomic|atomicfile'
    benchmarks = 'bench|benchmark'
}
$solutionMissingTopics = New-Object System.Collections.ArrayList
foreach ($topic in $solutionTopicPatterns.Keys) {
    if (-not ($solutionText -match $solutionTopicPatterns[$topic])) {
        $solutionMissingTopics.Add($topic) | Out-Null
    }
}
$checks['engineering.solution_doc'] = (Test-Path -LiteralPath $solutionPath) -and ($solutionMissingTopics.Count -eq 0)

# Additional checks required by target specification.
$checks['minimum.audio_integrity_probe'] = $checks['minimum.audio_integrity_probe'] -and ($goVetRes.Expected)

# Performance 10k run and cleanup marker.
$perfManifest = Join-Path $ctx.TmpDir 'perf_manifest.jsonl'
$perfHyp = Join-Path $ctx.TmpDir 'perf_hyp.jsonl'
$swm = New-Object System.IO.StreamWriter($perfManifest, $false, [Text.UTF8Encoding]::new($false))
$swh = New-Object System.IO.StreamWriter($perfHyp, $false, [Text.UTF8Encoding]::new($false))
try {
    for ($i = 0; $i -lt 10000; $i++) {
        $id = ('id-{0:d5}' -f $i)
        $swm.WriteLine( ('{{"id":"{0}","audio":"audio/a.wav","text":"mama myla ramu","language":"ru","duration_ms":1000,"sample_rate":16000,"channels":1,"tags":["crowd"],"sha256":"11"}}' -f $id) )
        $swh.WriteLine( ('{{"id":"{0}","text":"mama myla lamu","elapsed_ms":100}}' -f $id) )
    }
} finally {
    $swm.Flush(); $swm.Close(); $swh.Flush(); $swh.Close()
}

$perfOut = Join-Path $ctx.TmpDir 'perf_eval.json'
$perfRes = Invoke-HiddenPowerShell -Ctx $ctx -Name 'perf_evaluate_10k' -Command "& '$sttqExe' evaluate --manifest '$perfManifest' --hypotheses '$perfHyp' --normalization ru-default --out '$perfOut'" -WorkingDirectory $root -TimeoutSec 120
$perfOk = $perfRes.Expected -and ($perfRes.DurationMs -le 10000) -and ($perfRes.MaxWorkingSet -le 268435456)
$perfHash = if (Test-Path -LiteralPath $perfOut) { Get-FileSha256 -Path $perfOut } else { '' }
Save-Json -Path (Join-Path $ctx.OutputsDir 'performance_metrics.json') -Value ([ordered]@{
    evaluate_10k_elapsed_ms = $perfRes.DurationMs
    peak_tree_working_set_bytes = $perfRes.MaxWorkingSet
    under_10s = ($perfRes.DurationMs -le 10000)
    under_256mib = ($perfRes.MaxWorkingSet -le 268435456)
    output_sha256 = $perfHash
})

# Cleanup temporary binaries, generated perf data and large synthetic artifacts.
$cleanupTargets = @(
    $sttqExe,
    $linuxExe,
    $fakeExe,
    $fakeFFProbe,
    $fakeFFMpeg,
    $fakeWhisper,
    $fakeSource,
    $perfManifest,
    $perfHyp,
    $perfOut,
    $invalidAudioDir,
    $fixtureRoot,
    (Join-Path $fixtureRoot 'test.tar'),
    $prepared16Out,
    $prepared8Out,
    (Join-Path $ctx.OutputsDir 'import_dir\audio'),
    (Join-Path $ctx.OutputsDir 'import_tar\audio'),
    (Join-Path $ctx.OutputsDir 'import_unicode\audio'),
    (Join-Path $ctx.OutputsDir 'import_dir_repeat\audio'),
    (Join-Path $ctx.OutputsDir 'import_seed_alt\audio'),
    (Join-Path $root 'demo\current.json'),
    (Join-Path $root 'demo\report.html')
)
foreach ($target in $cleanupTargets) {
    if (Test-Path -LiteralPath $target) {
        Remove-Item -LiteralPath $target -Recurse -Force -ErrorAction SilentlyContinue
    }
}
Get-ChildItem -LiteralPath $ctx.TmpDir -Filter '*.exitcode' -File -ErrorAction SilentlyContinue | ForEach-Object {
    Remove-Item -LiteralPath $_.FullName -Force -ErrorAction SilentlyContinue
}

$cleanupViolations = New-Object System.Collections.ArrayList
$forbiddenPattern = '\.exe$|\.tar$|\.wav$|\.tmp$|perf_'
foreach ($entry in Get-ChildItem -LiteralPath $ctx.ResultDir -Recurse -File -ErrorAction SilentlyContinue) {
    $rel = Get-RelativePathSafe -BasePath $ctx.ResultDir -TargetPath $entry.FullName
    if (($rel -notmatch '^logs\\') -and ($rel -match $forbiddenPattern)) {
        $cleanupViolations.Add([ordered]@{ path = $rel; reason = 'forbidden_extension_or_perf_artifact' }) | Out-Null
    }
    if ($entry.Length -gt 5MB) {
        $cleanupViolations.Add([ordered]@{ path = $rel; reason = 'file_exceeds_5mb'; size_bytes = $entry.Length }) | Out-Null
    }
}
foreach ($entry in Get-ChildItem -LiteralPath $root -Recurse -File -ErrorAction SilentlyContinue) {
    $rel = Get-RelativePathSafe -BasePath $root -TargetPath $entry.FullName
    if ($rel -match '^\.check-results\\') {
        continue
    }
    if ($rel -match $forbiddenPattern) {
        $cleanupViolations.Add([ordered]@{ path = "repo:$rel"; reason = 'forbidden_in_repo_after_cleanup' }) | Out-Null
    }
    if ($entry.Length -gt 5MB) {
        $cleanupViolations.Add([ordered]@{ path = "repo:$rel"; reason = 'repo_file_exceeds_5mb'; size_bytes = $entry.Length }) | Out-Null
    }
}
$cleanupOk = ($cleanupViolations.Count -eq 0)
Save-Json -Path (Join-Path $ctx.OutputsDir 'cleanup.json') -Value ([ordered]@{
    cleanup_ok = $cleanupOk
    violations = @($cleanupViolations)
})

# Map remaining checks to combine runtime and targeted tests.
$checks['minimum.normalization_profiles'] = $checks['minimum.normalization_profiles'] -and $testPassed['TestNormalizeStrict'] -and $testPassed['TestNormalizeRUDefaultNFC']
$checks['minimum.wer_cer'] = $checks['minimum.wer_cer'] -and $testPassed['TestWordCharacterMetricsAndAlignment']
$checks['minimum.alignment'] = $checks['minimum.alignment'] -and $testPassed['TestWordCharacterMetricsAndAlignment']
$checks['minimum.aggregate_wer'] = $checks['minimum.aggregate_wer'] -and $testPassed['TestAggregateMetrics']
$checks['minimum.golos_import'] = $checks['minimum.golos_import'] -and $testPassed['TestImportGolosDirectoryAndTar']
$checks['minimum.deterministic_selection'] = $checks['minimum.deterministic_selection'] -and $testPassed['TestDeterministicSelectionBySeed']
$checks['minimum.manifest_validation'] = $checks['minimum.manifest_validation'] -and $testPassed['TestValidateManifestAggregatesErrors']
$checks['minimum.audio_profiles'] = $checks['minimum.audio_profiles'] -and $testPassed['TestAudioPrepareProfilesAndTimeout']
$checks['minimum.audio_integrity_probe'] = $checks['minimum.audio_integrity_probe'] -and $testPassed['TestAudioPrepareProfilesAndTimeout']
$checks['good.coverage_rtf'] = $checks['good.coverage_rtf'] -and $testPassed['TestCoverageRTFAndGroupedMetrics']
$checks['good.grouped_metrics'] = $checks['good.grouped_metrics'] -and $testPassed['TestCoverageRTFAndGroupedMetrics']
$checks['good.compare_exit_codes'] = $checks['good.compare_exit_codes'] -and $testPassed['TestCompareExitCodes']
$checks['good.runner_controls'] = $checks['good.runner_controls'] -and $testPassed['TestWhisperRunnerConcurrencyTimeoutResume']

# Ensure stage1 logsan is absent.
$logsanFiles = @(Get-ChildItem -LiteralPath (Join-Path $root 'cmd\logsan') -Recurse -File -ErrorAction SilentlyContinue)
if ($logsanFiles.Count -gt 0) {
    foreach ($key in $checks.Keys) {
        if ($key -like 'minimum.*' -or $key -like 'good.*') {
            $checks[$key] = $false
        }
    }
}

$repoSnapshotRoot = Join-Path $ctx.ResultDir 'repo_snapshot'
$repoSnapshotDocs = Join-Path $repoSnapshotRoot 'docs'
$repoSnapshotControl = Join-Path (Join-Path $repoSnapshotRoot 'testdata') 'control'
$repoSnapshotGolden = Join-Path (Join-Path $repoSnapshotRoot 'testdata') 'golden'
foreach ($dir in @($repoSnapshotRoot, $repoSnapshotDocs, $repoSnapshotControl, $repoSnapshotGolden)) {
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
}
Copy-Item -LiteralPath (Join-Path $root 'README.md') -Destination (Join-Path $repoSnapshotRoot 'README.md') -Force
Copy-Item -LiteralPath (Join-Path $root 'Makefile') -Destination (Join-Path $repoSnapshotRoot 'Makefile') -Force
Copy-Item -LiteralPath (Join-Path $root 'go.mod') -Destination (Join-Path $repoSnapshotRoot 'go.mod') -Force
Copy-Item -LiteralPath (Join-Path $root 'docs') -Destination $repoSnapshotRoot -Recurse -Force
Copy-Item -LiteralPath (Join-Path $root 'testdata\control') -Destination (Join-Path $repoSnapshotRoot 'testdata') -Recurse -Force
Copy-Item -LiteralPath (Join-Path $root 'testdata\golden') -Destination (Join-Path $repoSnapshotRoot 'testdata') -Recurse -Force

$requirementMap = @{
    'minimum.go120_build' = 'Go 1.20.x is used and both Windows/Linux builds succeed.'
    'minimum.cross_platform_paths' = 'Manifest contains only relative slash paths without drive letter.'
    'minimum.golos_import' = 'Golos import works from directory, tar and unicode path.'
    'minimum.transcripts_preserved' = 'Imported manifest texts exactly match selected source records.'
    'minimum.stable_ids' = 'Stable IDs match across dir/tar/unicode imports.'
    'minimum.quotas' = 'Domain quotas crowd=2 and farfield=1 are enforced.'
    'minimum.deterministic_selection' = 'Same seed yields same selection, different seed changes selection.'
    'minimum.manifest_validation' = 'validate reports full diagnostics and returns exit code 2.'
    'minimum.audio_integrity_probe' = 'Audio integrity is checked via ffprobe/ffmpeg pipeline.'
    'minimum.audio_profiles' = 'wav-16k/wav-8k profiles use exact ffmpeg args and output rates.'
    'minimum.normalization_profiles' = 'strict and ru-default normalization profiles are verified.'
    'minimum.wer_cer' = 'WER/CER calculations are validated on controlled cases.'
    'minimum.alignment' = 'Alignment operations equal/insert/substitute/delete are present and valid.'
    'minimum.aggregate_wer' = 'Aggregate WER is derived from total counts.'
    'good.coverage_rtf' = 'Coverage and RTF are verified, including 10k evaluate run.'
    'good.grouped_metrics' = 'Grouped metrics by tag and duration are verified.'
    'good.deterministic_json' = 'Repeated evaluate output is byte-deterministic JSON.'
    'good.html_details' = 'HTML report contains summary/groups/errors/alignment and escaped hostile input.'
    'good.compare_exit_codes' = 'compare returns exact exit 0/1/2 for pass/regression/error.'
    'good.whisper_adapter' = 'Whisper adapter forwards argv model/language/audio and writes hypotheses.'
    'good.runner_controls' = 'Runner resume/workers/timeout behavior is validated by black-box and tests.'
    'excellent.unit_tests' = 'All required targeted unit tests pass by go test -json events.'
    'excellent.golden_benchmarks' = 'TestReportGolden passes and all five benchmarks expose benchmem metrics.'
    'excellent.contextual_errors' = 'Repeated validate diagnostics are stable and contextual.'
    'excellent.atomic_files' = 'Atomic file behavior is verified by targeted tests and runtime cleanup.'
    'excellent.offline_demo' = 'Offline make demo succeeds and generates valid artifacts.'
    'excellent.readme_complete' = 'README covers all required assignment topics including stub marker.'
    'engineering.unit_tests_present' = 'Unit test presence is verified from go test -list output.'
    'engineering.benchmarks_present' = 'Benchmark presence is verified from list output and bench logs.'
    'engineering.go_test_passes' = 'Full go test ./... run succeeds.'
    'engineering.make_test_runs' = 'make test command succeeds in hidden run.'
    'engineering.make_bench_runs' = 'make bench command succeeds in hidden run.'
    'engineering.make_demo_runs' = 'make demo command succeeds in hidden run.'
    'engineering.readme' = 'README exists and satisfies semantic topic checks.'
    'engineering.make_test' = 'Makefile test target is executable and validated by run.'
    'engineering.make_bench' = 'Makefile bench target is executable and validated by run.'
    'engineering.make_demo' = 'Makefile demo target is executable and validated by run.'
    'engineering.control_data' = 'make demo consumes testdata/control inputs and emits demo outputs.'
    'engineering.solution_doc' = 'solution.md covers required architecture and algorithm topics.'
}

$evidenceMap = @{
    'minimum.go120_build' = @('logs/build_cli.log', 'logs/build_cli_linux.log', 'meta/go_version.txt')
    'minimum.cross_platform_paths' = @('logs/import_dir.log', 'outputs/import_dir/manifest.jsonl')
    'minimum.golos_import' = @('logs/import_dir.log', 'logs/import_tar.log', 'logs/import_unicode_path.log')
    'minimum.transcripts_preserved' = @('outputs/import_dir/manifest.jsonl', 'outputs/import_tar/manifest.jsonl', 'outputs/import_unicode/manifest.jsonl')
    'minimum.stable_ids' = @('outputs/import_dir/manifest.jsonl', 'outputs/import_tar/manifest.jsonl', 'outputs/import_unicode/manifest.jsonl')
    'minimum.quotas' = @('outputs/import_dir/manifest.jsonl', 'logs/import_dir.log')
    'minimum.deterministic_selection' = @('outputs/import_dir/source/selection.json', 'outputs/import_dir_repeat/source/selection.json', 'outputs/import_seed_alt/source/selection.json')
    'minimum.manifest_validation' = @('logs/validate_good.log', 'logs/validate_bad.log', 'logs/validate_bad_repeat.log')
    'minimum.audio_integrity_probe' = @('logs/audio_prepare_wav16.log', 'logs/audio_prepare_wav8.log', 'outputs/fake_logs/fake-tools.log')
    'minimum.audio_profiles' = @('logs/audio_prepare_wav16.log', 'logs/audio_prepare_wav8.log', 'outputs/fake_logs/fake-tools.log')
    'minimum.normalization_profiles' = @('logs/evaluate_1.log', 'outputs/eval1.json', 'outputs/go_test_named.jsonl')
    'minimum.wer_cer' = @('outputs/eval1.json', 'logs/evaluate_1.log')
    'minimum.alignment' = @('outputs/eval1.json', 'outputs/eval_html.json')
    'minimum.aggregate_wer' = @('outputs/eval1.json')
    'good.coverage_rtf' = @('outputs/eval1.json', 'outputs/performance_metrics.json', 'logs/perf_evaluate_10k.log')
    'good.grouped_metrics' = @('outputs/eval1.json')
    'good.deterministic_json' = @('outputs/eval1.json', 'outputs/eval2.json')
    'good.html_details' = @('outputs/report.html', 'logs/report_html.log')
    'good.compare_exit_codes' = @('logs/compare_pass.log', 'logs/compare_regression.log', 'logs/compare_error.log')
    'good.whisper_adapter' = @('logs/run_whisper.log', 'outputs/run_whisper.jsonl', 'outputs/fake_logs/fake-tools.log')
    'good.runner_controls' = @('logs/run_whisper_resume.log', 'outputs/run_whisper.jsonl', 'outputs/go_test_named.jsonl')
    'excellent.unit_tests' = @('logs/go_test_named_json.log', 'outputs/go_test_named.jsonl')
    'excellent.golden_benchmarks' = @('logs/make_bench.log', 'repo_snapshot/testdata/golden/evaluation.json')
    'excellent.contextual_errors' = @('logs/validate_bad.log', 'logs/validate_bad_repeat.log', 'logs/compare_error.log')
    'excellent.atomic_files' = @('logs/audio_prepare_wav16.log', 'logs/run_whisper.log', 'outputs/cleanup.json', 'outputs/go_test_named.jsonl')
    'excellent.offline_demo' = @('logs/make_demo.log', 'repo_snapshot/testdata/control/manifest.jsonl', 'repo_snapshot/testdata/control/hypotheses.jsonl')
    'excellent.readme_complete' = @('repo_snapshot/README.md', 'repo_snapshot/docs/solution.md')
    'engineering.unit_tests_present' = @('logs/go_test_list.log', 'outputs/go_test_list.txt')
    'engineering.benchmarks_present' = @('logs/go_test_list.log', 'logs/make_bench.log')
    'engineering.go_test_passes' = @('logs/go_fmt_all.log', 'logs/go_test_all.log', 'logs/go_vet_all.log')
    'engineering.make_test_runs' = @('logs/make_test.log')
    'engineering.make_bench_runs' = @('logs/make_bench.log')
    'engineering.make_demo_runs' = @('logs/make_demo.log')
    'engineering.readme' = @('repo_snapshot/README.md')
    'engineering.make_test' = @('repo_snapshot/Makefile', 'logs/make_test.log')
    'engineering.make_bench' = @('repo_snapshot/Makefile', 'logs/make_bench.log')
    'engineering.make_demo' = @('repo_snapshot/Makefile', 'logs/make_demo.log')
    'engineering.control_data' = @('repo_snapshot/testdata/control/manifest.jsonl', 'repo_snapshot/testdata/control/hypotheses.jsonl', 'logs/make_demo.log')
    'engineering.solution_doc' = @('repo_snapshot/docs/solution.md')
}

$detailsMap = @{
    'minimum.go120_build' = "go_version=$goVersion build_exit_windows=$($buildRes.ExitCode) build_exit_linux=$($buildLinuxRes.ExitCode)"
    'minimum.deterministic_selection' = "same_seed_hash_equal=$($checks['minimum.deterministic_selection'])"
    'minimum.manifest_validation' = "validate_bad_exit=$($validateBadRes.ExitCode) source=$($validateBadRes.ExitCodeSource) validate_bad_repeat_exit=$($validateBadRepeatRes.ExitCode) source_repeat=$($validateBadRepeatRes.ExitCodeSource)"
    'good.compare_exit_codes' = "compare_pass_exit=$($comparePassRes.ExitCode) source=$($comparePassRes.ExitCodeSource) compare_regression_exit=$($compareRegRes.ExitCode) source_regression=$($compareRegRes.ExitCodeSource) compare_error_exit=$($compareErrRes.ExitCode) source_error=$($compareErrRes.ExitCodeSource)"
    'excellent.unit_tests' = "required_tests_passed=$allRequiredTestsPassed"
    'excellent.golden_benchmarks' = "test_report_golden_passed=$($testPassed['TestReportGolden']) benchmarks_present=$allBenchPresent"
    'excellent.contextual_errors' = "validate_repeat_equal=$validateBadStable manifest_rel=$invalidManifestRel"
    'excellent.atomic_files' = "TestAtomicFile=$($testPassed['TestAtomicFile']) TestAudioPrepareProfilesAndTimeout=$($testPassed['TestAudioPrepareProfilesAndTimeout']) runtime_tmp_leftovers=$($runtimeTempLeftovers.Count)"
    'excellent.readme_complete' = "missing_topics=$([string]::Join(',', @($readmeMissingTopics)))"
    'engineering.readme' = "missing_topics=$([string]::Join(',', @($readmeMissingTopics)))"
    'engineering.solution_doc' = "missing_topics=$([string]::Join(',', @($solutionMissingTopics)))"
    'engineering.go_test_passes' = "go_fmt_exit=$($goFmtRes.ExitCode) go_test_exit=$($goTestRes.ExitCode) go_vet_exit=$($goVetRes.ExitCode)"
}

foreach ($item in $ids) {
    $id = [string]$item.id
    $details = if ($detailsMap.ContainsKey($id)) { [string]$detailsMap[$id] } else { "check=$($checks[$id])" }
    $evidence = if ($evidenceMap.ContainsKey($id)) { @($evidenceMap[$id]) } else { @() }
    Add-Assessment -Ctx $ctx -Id $id -Level $item.level -Ok ([bool]$checks[$id]) -Requirement $requirementMap[$id] -Evidence $evidence -Details $details
}

$missingEvidence = New-Object System.Collections.ArrayList
for ($i = 0; $i -lt $ctx.Assessments.Count; $i++) {
    $assessment = $ctx.Assessments[$i]
    $localMissing = New-Object System.Collections.ArrayList
    foreach ($ev in @($assessment.evidence)) {
        $evPath = Join-Path $ctx.ResultDir $ev
        if (-not (Test-Path -LiteralPath $evPath)) {
            $localMissing.Add($ev) | Out-Null
        }
    }
    if ($localMissing.Count -gt 0) {
        $assessment.implementation = 'partial'
        $assessment.conformance = 'nonconformant'
        $oldDetails = [string]$assessment.details
        $appendDetails = "missing_evidence=$([string]::Join(',', @($localMissing)))"
        if ([string]::IsNullOrWhiteSpace($oldDetails)) {
            $assessment.details = $appendDetails
        } else {
            $assessment.details = "$oldDetails | $appendDetails"
        }
        $missingEvidence.Add([ordered]@{ id = $assessment.id; missing = @($localMissing) }) | Out-Null
    }
}
Save-Json -Path (Join-Path $ctx.OutputsDir 'evidence_validation.json') -Value ([ordered]@{ missing = @($missingEvidence) })

# Enforce exact count/distribution as explicit guard.
$assessmentIds = @($ctx.Assessments | ForEach-Object { $_.id })
$uniqueCount = @($assessmentIds | Select-Object -Unique).Count
if ($ctx.Assessments.Count -ne 39 -or $uniqueCount -ne 39) {
    throw "assessment ids mismatch: count=$($ctx.Assessments.Count) unique=$uniqueCount"
}

$distribution = @{
    minimum = @($ctx.Assessments | Where-Object { $_.level -eq 'minimum' }).Count
    good = @($ctx.Assessments | Where-Object { $_.level -eq 'good' }).Count
    excellent = @($ctx.Assessments | Where-Object { $_.level -eq 'excellent' }).Count
    engineering = @($ctx.Assessments | Where-Object { $_.level -eq 'engineering' }).Count
}
if ($distribution.minimum -ne 14 -or $distribution.good -ne 7 -or $distribution.excellent -ne 6 -or $distribution.engineering -ne 12) {
    throw "distribution mismatch: $($distribution | ConvertTo-Json -Compress)"
}

Complete-Check -Ctx $ctx -Extra @{
    stage = 'stage2-only sttq'
    expected_ids = 39
    performance_ok = $perfOk
    cleanup_ok = $cleanupOk
    peak_tree_working_set_limit_bytes = 268435456
}
