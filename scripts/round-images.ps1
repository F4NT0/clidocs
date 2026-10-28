<#
.SYNOPSIS
    Rounds the corners of README images (PNG/JPG) and prints centered HTML tags.

.DESCRIPTION
    Every .png/.jpg/.jpeg in the images folder is redrawn with rounded corners and a
    transparent background, and saved as PNG (a .jpg produces a .png next to it).
    Re-running on an already rounded PNG is safe.

.PARAMETER Dir      Images folder (default: ..\images relative to this script).
.PARAMETER Radius   Corner radius in pixels (default: 20).
.PARAMETER Width    Width used in the generated HTML tags (default: 800).
.PARAMETER Snippet  Print a centered <p align="center"><img ...></p> tag for each image.

.EXAMPLE
    .\scripts\round-images.ps1 -Radius 24 -Snippet
#>
param(
    [string]$Dir = (Join-Path $PSScriptRoot '..\images'),
    [int]$Radius = 20,
    [int]$Width = 800,
    [switch]$Snippet
)

Add-Type -AssemblyName System.Drawing

if (-not (Test-Path $Dir)) { Write-Error "Folder not found: $Dir"; exit 1 }
$Dir = (Resolve-Path $Dir).Path

$files = Get-ChildItem $Dir -File | Where-Object { $_.Extension -match '^\.(png|jpe?g)$' }
if (-not $files) { Write-Host "No images found in $Dir"; exit 0 }

foreach ($f in $files) {
    $src = [System.Drawing.Image]::FromFile($f.FullName)
    try {
        $w = $src.Width; $h = $src.Height
        $r = [Math]::Min($Radius, [Math]::Min($w, $h) / 2)
        $d = $r * 2

        $bmp = New-Object System.Drawing.Bitmap $w, $h, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
        $g = [System.Drawing.Graphics]::FromImage($bmp)
        $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
        $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        $g.Clear([System.Drawing.Color]::Transparent)

        $path = New-Object System.Drawing.Drawing2D.GraphicsPath
        $path.AddArc(0, 0, $d, $d, 180, 90)
        $path.AddArc($w - $d, 0, $d, $d, 270, 90)
        $path.AddArc($w - $d, $h - $d, $d, $d, 0, 90)
        $path.AddArc(0, $h - $d, $d, $d, 90, 90)
        $path.CloseFigure()

        $g.SetClip($path)
        $g.DrawImage($src, 0, 0, $w, $h)
        $g.Dispose(); $path.Dispose()

        $out = Join-Path $Dir ([IO.Path]::GetFileNameWithoutExtension($f.Name) + '.png')
        $tmp = "$out.tmp"
        $bmp.Save($tmp, [System.Drawing.Imaging.ImageFormat]::Png)
        $bmp.Dispose()
    }
    finally { $src.Dispose() }

    Move-Item -LiteralPath $tmp -Destination $out -Force
    Write-Host "Rounded: $(Split-Path $out -Leaf) (${w}x${h}, radius $r)"

    if ($Snippet) {
        $name = Split-Path $out -Leaf
        Write-Host "<p align=`"center`">`n  <img src=`"images/$name`" alt=`"$([IO.Path]::GetFileNameWithoutExtension($name))`" width=`"$Width`">`n</p>`n"
    }
}
