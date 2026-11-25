$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $true
$work = Join-Path $env:RUNNER_TEMP 'windows smoke'
$repo = Join-Path $work '示例 repository'
$install = Join-Path $work '应用程序'
$evidence = Join-Path $env:RUNNER_TEMP 'native-windows-evidence'
New-Item -ItemType Directory -Path $repo, $evidence | Out-Null
git -C $repo init --quiet
Set-Content -Encoding utf8NoBOM -Path "$repo/README.md" -Value '# Windows smoke'
git -C $repo add .
git -C $repo -c user.name=Fixture -c user.email=test@example.org commit --quiet -m fixture

$installer = Get-ChildItem "$env:RUNNER_TEMP/desktop-build/windows-amd64/*-setup-*.exe"
if (@($installer).Count -ne 1) { throw 'Expected one candidate installer' }
$setup = Start-Process $installer.FullName -ArgumentList "/S /D=$install" -PassThru -Wait
if ($setup.ExitCode -ne 0) { throw "Installer exited with $($setup.ExitCode)" }

$app = Start-Process "$install/Repolens.exe" -ArgumentList "-lang en `"$repo`"" -PassThru
try {
    for ($i = 0; $i -lt 100; $i++) {
        $app.Refresh()
        if ($app.HasExited) { throw "Installed app exited with $($app.ExitCode)" }
        if ($app.MainWindowHandle -ne 0 -and $app.MainWindowTitle -eq 'Repolens') { break }
        Start-Sleep -Milliseconds 300
    }
    if ($app.MainWindowTitle -ne 'Repolens') { throw 'Installed app did not open the native window' }
} finally {
    if (!$app.HasExited) { Stop-Process $app.Id -Force; $app.WaitForExit() }
}

# The fixture window uses an instrumented binary, separate from the shipped installer.
go build -tags e2e -o "$work/probe.exe" ./cmd/repolens-desktop
go build -o "$work/automation.exe" github.com/egoist/mygo/plugins/automation/cmd/mygo-automation
$env:MYGO_AUTOMATION = Join-Path $env:RUNNER_TEMP 'repolens.sock'
$env:MYGO_AUTOMATION_SESSION = 'repolens-win'
$app = Start-Process "$work/probe.exe" -ArgumentList "-lang en `"$repo`"" -PassThru
try {
    for ($i = 0; $i -lt 100; $i++) {
        if ($app.HasExited) { throw "Fixture app exited with $($app.ExitCode)" }
        if (Test-Path $env:MYGO_AUTOMATION) { break }
        Start-Sleep -Milliseconds 200
    }
    $automation = Join-Path $work 'automation.exe'
    & $automation open --socket $env:MYGO_AUTOMATION
    & $automation wait --text 'Paths and Rules'
    & $automation screenshot "$evidence/site-settings.png"
    & $automation press 'key=build-preview' --settle
    & $automation press 'key=start-build' --settle
    & $automation wait --text 'Build succeeded'
    & $automation snapshot | Set-Content -Encoding utf8 "$evidence/build.txt"
    & $automation screenshot "$evidence/build.png"
} finally {
    if (!$app.HasExited) { Stop-Process $app.Id -Force; $app.WaitForExit() }
    $uninstall = Start-Process "$install/Uninstall.exe" -ArgumentList '/S' -PassThru -Wait
    if ($uninstall.ExitCode -ne 0) { throw "Uninstaller exited with $($uninstall.ExitCode)" }
    for ($i = 0; $i -lt 50 -and (Test-Path "$install/Repolens.exe"); $i++) {
        Start-Sleep -Milliseconds 200
    }
    if (Test-Path "$install/Repolens.exe") { throw 'Uninstaller left the installed executable' }
}
