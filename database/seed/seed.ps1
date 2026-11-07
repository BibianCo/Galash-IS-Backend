#requires -Version 7.0
# Stream SQL bytes unchanged: a PowerShell text pipeline can replace accents with '?'.
# -Real / -RealReset load / delete the real GALASH data (SGI members, group projects, seedbeds); those files are not in git.
param([switch]$RepairUtf8, [switch]$DryRun, [switch]$Real, [switch]$RealReset)
$ErrorActionPreference = 'Stop'
if ($DryRun -and -not $RepairUtf8) { throw '-DryRun requires -RepairUtf8.' }
$fileNames = if ($Real) { @('sgi_uptc_integrantes.sql', 'proyectos_grupo.sql', 'semilleros_galash.sql') } elseif ($RealReset) { @('semilleros_galash_reset.sql', 'proyectos_grupo_reset.sql', 'sgi_uptc_integrantes_reset.sql') } elseif ($RepairUtf8) { @('repair_utf8.sql') } else { @('synthetic.sql') }
$databaseUser = if ($env:POSTGRES_USER) { $env:POSTGRES_USER } else { 'galash' }
$databaseName = if ($env:POSTGRES_DB) { $env:POSTGRES_DB } else { 'galash' }
foreach ($fileName in $fileNames) {
    $seedFile = Join-Path $PSScriptRoot $fileName
    $startInfo = [System.Diagnostics.ProcessStartInfo]::new('docker')
    $startInfo.WorkingDirectory = (Resolve-Path (Join-Path $PSScriptRoot '../..')).Path
    $startInfo.UseShellExecute = $false
    $startInfo.RedirectStandardInput = $true
    foreach ($argument in @('compose', 'exec', '-T', '-e', 'PGCLIENTENCODING=UTF8', 'postgres', 'psql', '-X', '-v', 'ON_ERROR_STOP=1', '-q', '-U', $databaseUser, '-d', $databaseName)) {
        $startInfo.ArgumentList.Add($argument)
    }
    $process = [System.Diagnostics.Process]::new()
    $process.StartInfo = $startInfo
    $inputFile = $null
    try {
        if (-not $process.Start()) { throw 'Could not start Docker.' }
        if ($DryRun) {
            $sql = [System.IO.File]::ReadAllText($seedFile, [System.Text.Encoding]::UTF8).Replace('COMMIT;', 'ROLLBACK;')
            $bytes = [System.Text.Encoding]::UTF8.GetBytes($sql)
            $process.StandardInput.BaseStream.Write($bytes, 0, $bytes.Length)
        } else {
            $inputFile = [System.IO.File]::OpenRead($seedFile)
            $inputFile.CopyTo($process.StandardInput.BaseStream)
        }
        $process.StandardInput.Close()
        $process.WaitForExit()
        if ($process.ExitCode -ne 0) { throw "psql exited with code $($process.ExitCode)." }
        Write-Host "OK: $fileName (UTF-8; dry run: $DryRun)"
    } finally {
        if ($inputFile) { $inputFile.Dispose() }
        $process.Dispose()
    }
}
