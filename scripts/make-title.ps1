<#
.SYNOPSIS
    Renders a colored title as a transparent PNG (GitHub Markdown cannot color text).

.EXAMPLE
    .\scripts\make-title.ps1 -Text "Folder panel usage" -Out images\folder-panel-usage.png
#>
param(
    [Parameter(Mandatory)][string]$Text,
    [Parameter(Mandatory)][string]$Out,
    [string]$Color = '#00ff88',
    [int]$FontSize = 56
)

Add-Type -AssemblyName System.Drawing

$font = New-Object System.Drawing.Font('Consolas', $FontSize, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
$sf = [System.Drawing.StringFormat]::GenericTypographic
$tmp = New-Object System.Drawing.Bitmap 10, 10
$g = [System.Drawing.Graphics]::FromImage($tmp)
$size = $g.MeasureString($Text, $font, [int]::MaxValue, $sf)
$g.Dispose(); $tmp.Dispose()

$pad = 12
$w = [int][Math]::Ceiling($size.Width) + 2 * $pad
$h = [int][Math]::Ceiling($size.Height) + 2 * $pad
$bmp = New-Object System.Drawing.Bitmap $w, $h, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.Clear([System.Drawing.Color]::Transparent)
$g.TextRenderingHint = [System.Drawing.Text.TextRenderingHint]::AntiAlias
$brush = New-Object System.Drawing.SolidBrush ([System.Drawing.ColorTranslator]::FromHtml($Color))
$g.DrawString($Text, $font, $brush, $pad, $pad, $sf)
$g.Dispose()

$dir = Split-Path -Parent $Out
if ($dir) { New-Item -ItemType Directory -Force $dir | Out-Null }
$full = Join-Path (Get-Location) $Out
if ([IO.Path]::IsPathRooted($Out)) { $full = $Out }
$bmp.Save($full, [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
Write-Host "Created $Out (${w}x${h})"
