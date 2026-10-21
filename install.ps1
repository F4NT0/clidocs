#Requires -Version 5.1
<#
.SYNOPSIS
    Adiciona o diretório dist\ do clidocs ao PATH do usuario (sem copiar arquivos).
.DESCRIPTION
    Resolve o caminho absoluto de dist\ relativo ao script e o adiciona
    ao PATH permanente do usuario atual. Nao requer privilegios de Administrador.
.NOTES
    Para remover, edite manualmente as variaveis de ambiente do usuario.
#>

$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
# Configuração — resolve o diretório dist\ relativo ao script
# ---------------------------------------------------------------------------
$ExeName   = "clidocs.exe"
# $PSScriptRoot pode ser vazio se rodado via '.\install.ps1' sem path completo.
# Fallback: usa o diretório atual do terminal.
$ScriptDir = if ($PSScriptRoot -ne "") { $PSScriptRoot } else { $PWD.Path }
$DistDir   = Join-Path $ScriptDir "dist"
$ExePath   = Join-Path $DistDir $ExeName

# ---------------------------------------------------------------------------
# Verificar se o exe existe no dist\
# ---------------------------------------------------------------------------
if (-not (Test-Path $ExePath)) {
    Write-Host "ERRO: $ExePath nao encontrado." -ForegroundColor Red
    Write-Host "Execute primeiro: go build -o dist\clidocs.exe ." -ForegroundColor Yellow
    exit 1
}

Write-Host "clidocs.exe encontrado em: $DistDir" -ForegroundColor Cyan

# ---------------------------------------------------------------------------
# Adicionar dist\ ao PATH do usuario (nao requer Admin)
# ---------------------------------------------------------------------------
$currentPath = [System.Environment]::GetEnvironmentVariable("Path", "User")

if ($currentPath -like "*$DistDir*") {
    Write-Host "OK: $DistDir ja esta no PATH do usuario." -ForegroundColor Green
} else {
    $newPath = $currentPath.TrimEnd(";") + ";" + $DistDir
    [System.Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "OK: $DistDir adicionado ao PATH do usuario." -ForegroundColor Green
}

# Atualizar PATH da sessao atual para testar imediatamente
$env:Path = [System.Environment]::GetEnvironmentVariable("Path", "Machine") + ";" +
            [System.Environment]::GetEnvironmentVariable("Path", "User")

# ---------------------------------------------------------------------------
# Verificar instalação
# ---------------------------------------------------------------------------
Write-Host ""
$installed = Get-Command clidocs -ErrorAction SilentlyContinue
if ($installed) {
    Write-Host "Sucesso! 'clidocs' disponivel em: $($installed.Source)" -ForegroundColor Green
    Write-Host "Execute 'clidocs' em qualquer terminal para iniciar." -ForegroundColor Green
} else {
    Write-Host "PATH atualizado. Abra um novo terminal e execute: clidocs" -ForegroundColor Yellow
}
