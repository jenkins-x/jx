@echo off
setlocal EnableDelayedExpansion

REM This script is intended to be used to match a pass attempt in a loop. Currently used in command_test.go

set _file=%1

if exist !_file! (
    set /p TRY=<!_file!
    if "!TRY!" == "" (
        set TRY=1
    )
) else (
    set TRY=1
)

set PASS_ATTEMPT=%2

if "!TRY!" == "!PASS_ATTEMPT!" (
    echo PASS
    set /a TRY=!TRY!+1
    echo !TRY!>!_file!
    exit 0
) else (
    echo FAILURE^^!
    set /a TRY=!TRY!+1
    echo !TRY!>!_file!
    exit 1
)