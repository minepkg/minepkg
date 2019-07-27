$output = "$Env:LOCALAPPDATA\minepkg\minepkg.exe"

Write-Output "Downloading ..."
Remove-Item -ErrorAction Ignore "$env:TEMP\minepkg.exe"
$url = "https://storage.googleapis.com/minepkg-client/latest/minepkg-windows-amd64.exe"
(New-Object System.Net.WebClient).DownloadFile($url, "$env:TEMP\minepkg.exe")
Write-Output "Download finished"

New-Item -ItemType directory -Force -Path $env:LOCALAPPDATA\minepkg\ | Out-Null
Copy-Item -Force "$env:TEMP\minepkg.exe" $output

Write-Output "Adding binary to PATH (this might take a few seconds)"

$hasEnv = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User).Split(";").Contains("$env:LOCALAPPDATA\minepkg\")

if (-Not $hasEnv) {
  [Environment]::SetEnvironmentVariable(
      "Path",
      [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::User) + ";$env:LOCALAPPDATA\minepkg\",
      [EnvironmentVariableTarget]::User
  )
}


$ENV:PATH="$ENV:PATH;$env:LOCALAPPDATA\minepkg\"

Write-Output "Installation complete"
