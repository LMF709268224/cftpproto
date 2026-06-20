# gen.ps1 — Generate Go protobuf code for cftp services
# Usage:
#   .\gen.ps1          (Generates all services)
#   .\gen.ps1 <name>   (Generates specific service, e.g., .\gen.ps1 prog)

param (
    [string]$Target
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

# --- Configure protoc include path ---
$protocInclude = if ($env:PROTOC_INCLUDE) {
    $env:PROTOC_INCLUDE
} elseif (Test-Path "C:\protoc\include") {
    "C:\protoc\include"
} elseif (Test-Path "C:\Program Files\protoc\include") {
    "C:\Program Files\protoc\include"
} else {
    $protocPath = (Get-Command protoc -ErrorAction SilentlyContinue).Source
    if ($protocPath) {
        Join-Path (Split-Path (Split-Path $protocPath -Parent) -Parent) "include"
    } else {
        Write-Error "Cannot find protoc include directory. Set PROTOC_INCLUDE env var."
        exit 1
    }
}
Write-Host "Using protoc include path: $protocInclude"

$outDir = Resolve-Path ".."
$goModule = (Get-Content ../go.mod | Select-String '^module\s+(\S+)').Matches.Groups[1].Value
$goOut = if ($goModule -like "*/cftp") { "cftp" } else { $goModule }


# --- Proto definitions ---
$protos = @(
    @{File="cc.proto";      Dir="gcc"},
    @{File="config.proto";  Dir="cfgserver"},
    @{File="creds.proto";   Dir="gcreds"},
    @{File="exam.proto";    Dir="gexam"},
    @{File="lms.proto";     Dir="glms"},
    @{File="mail.proto";    Dir="gmail"},
    @{File="mall.proto";    Dir="gmall"},
    @{File="msg.proto";     Dir="gmsg"},
    @{File="pay.proto";   Dir="gpay"},
    @{File="prog.proto";    Dir="gprog"},
    @{File="mid.proto";     Dir="gmid"},
    @{File="pdfgen.proto";  Dir="gpdf"},
    @{File="mbr.proto";     Dir="gmbr"}
)

# --- Filter logic ---
if (-not [string]::IsNullOrWhiteSpace($Target)) {
    $filteredProtos = $protos | Where-Object { $_.File -like "*$Target*" -or $_.Dir -like "*$Target*" }
    
    if ($filteredProtos.Count -eq 0) {
        Write-Host "Warning: No proto file or directory matching '$Target' found." -ForegroundColor Yellow
        exit 1
    }
    $protos = $filteredProtos
}

# --- Execution ---
foreach ($p in $protos) {
    $protoFile = $p.File
    $svcDir    = Join-Path $outDir $p.Dir

    New-Item -ItemType Directory -Force -Path $svcDir | Out-Null

    Write-Host ""
    Write-Host "=== Generating $protoFile -> $svcDir ==="

    # Generate .pb.go
    protoc `
        --proto_path=. `
        --proto_path="$protocInclude" `
        --go_out="$outDir" `
        --go_opt=module=$goOut `
        $protoFile

    # Generate _grpc.pb.go
    protoc `
        --proto_path=. `
        --proto_path="$protocInclude" `
        --go-grpc_out="$outDir" `
        --go-grpc_opt=module=$goOut `
        $protoFile

    Write-Host "  Success"
}

Write-Host ""
Write-Host "=== All generation tasks completed ==="
