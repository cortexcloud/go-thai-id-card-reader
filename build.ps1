# PowerShell build script for Thai ID Card Reader

# Get version from VERSION file
$VERSION = (Get-Content VERSION).Trim()

# Set output directory
$OUTPUT_DIR = "dist"

# Create output directory if it doesn't exist
if (!(Test-Path $OUTPUT_DIR)) {
    New-Item -ItemType Directory -Path $OUTPUT_DIR | Out-Null
}

Write-Host "Building Thai ID Card Reader v$VERSION"
Write-Host "================================================"

# Build for Windows (amd64)
Write-Host "Building for Windows (amd64)..."
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-s -w -X main.Version=$VERSION" `
    -o "$OUTPUT_DIR\thai-id-card-reader-$VERSION.exe" `
    .\cmd\card-service

# Reset environment variables
$env:GOOS = ""
$env:GOARCH = ""

Write-Host ""
Write-Host "Build complete! Files are in the $OUTPUT_DIR directory:"
Get-ChildItem "$OUTPUT_DIR\thai-id-card-reader-*" | Select-Object Name, Length