param(
    [Parameter(Mandatory = $true)]
    [string]$Server,

    [string]$User = "root",

    [string]$IdentityFile = "",

    [string]$RemoteDir = "/tmp/tomato-beinai-nginx"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")
$sslDir = Join-Path $repoRoot "deploy\ssl\tomato.beinai.cc"
$nginxDir = Join-Path $repoRoot "deploy\nginx"

$requiredFiles = @(
    (Join-Path $sslDir "tomato.beinai.cc_bundle.crt"),
    (Join-Path $sslDir "tomato.beinai.cc.key"),
    (Join-Path $nginxDir "tomato.beinai.cc.conf"),
    (Join-Path $nginxDir "install-tomato-beinai-nginx.sh")
)

foreach ($file in $requiredFiles) {
    if (-not (Test-Path $file)) {
        throw "Missing required file: $file"
    }
}

$sshArgs = @()
if ($IdentityFile -ne "") {
    $sshArgs += @("-i", $IdentityFile)
}

$target = "$User@$Server"

ssh @sshArgs $target "rm -rf '$RemoteDir' && mkdir -p '$RemoteDir/deploy/nginx' '$RemoteDir/deploy/ssl/tomato.beinai.cc'"
scp @sshArgs (Join-Path $nginxDir "tomato.beinai.cc.conf") "${target}:$RemoteDir/deploy/nginx/"
scp @sshArgs (Join-Path $nginxDir "install-tomato-beinai-nginx.sh") "${target}:$RemoteDir/deploy/nginx/"
scp @sshArgs (Join-Path $sslDir "tomato.beinai.cc_bundle.crt") "${target}:$RemoteDir/deploy/ssl/tomato.beinai.cc/"
scp @sshArgs (Join-Path $sslDir "tomato.beinai.cc.key") "${target}:$RemoteDir/deploy/ssl/tomato.beinai.cc/"
ssh @sshArgs $target "cd '$RemoteDir' && sudo bash deploy/nginx/install-tomato-beinai-nginx.sh"
