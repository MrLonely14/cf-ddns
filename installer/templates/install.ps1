# Cloudflare DDNS - Windows Task Scheduler Installation Script

param(
    [string]$ExecPath = "{{.ExecPath}}",
    [string]$ConfigPath = "{{.ConfigPath}}"
)

$TaskName = "CloudflareDDNS"
$TaskDescription = "Cloudflare Dynamic DNS Updater"

Write-Host "Installing Cloudflare DDNS as Windows scheduled task..."

# Check if running as administrator
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Error "This script must be run as Administrator"
    exit 1
}

# Create the scheduled task action
$Action = New-ScheduledTaskAction -Execute $ExecPath -Argument "run -config `"$ConfigPath`""

# Create the trigger (at startup)
$Trigger = New-ScheduledTaskTrigger -AtStartup

# Create the principal (run with highest privileges)
$Principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType ServiceAccount -RunLevel Highest

# Create the settings
$Settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -RestartCount 3 `
    -RestartInterval (New-TimeSpan -Minutes 1)

# Register the scheduled task
try {
    Register-ScheduledTask `
        -TaskName $TaskName `
        -Description $TaskDescription `
        -Action $Action `
        -Trigger $Trigger `
        -Principal $Principal `
        -Settings $Settings `
        -Force | Out-Null

    Write-Host "Successfully created scheduled task: $TaskName"

    # Start the task immediately
    Start-ScheduledTask -TaskName $TaskName
    Write-Host "Task started successfully"

    Write-Host "`nThe service will now run at system startup."
    Write-Host "To manage the task, use Task Scheduler (taskschd.msc)"
}
catch {
    Write-Error "Failed to create scheduled task: $_"
    exit 1
}
