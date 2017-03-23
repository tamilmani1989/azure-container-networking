:main
  @:: Reset environment to be able to start init in the same window multiple times.
  @::
  @call :reset_env || (
    if defined CorextExeVersion (
      echo INF: Already in init session.
      exit /b 0
    )
  )

  @call :set_defaults || (
    echo WRN: Failed to set defaults. Cache will not be created.

    call :clean_env

    call "%~dp0init.full.cmd" || exit /b 1

    exit /b 0
  )

  @call :validate_cache || set INIT_CACHE_INVALIDATE=1

  @if "%INIT_CACHE_INVALIDATE%"=="1" (
    echo INF: Init cache invalidated.
    echo INF: Do full init.

    echo INF: Save cache to "%INIT_CACHE_ENV_DIR%"

    set 1>"%INIT_CACHE_PRE_ENV_PATH%"

    call "%~dp0init.full.cmd" || exit /b 1

    set 1>"%INIT_CACHE_POST_ENV_PATH%"

    call :backup_gen || (
      echo WRN: Failed to gen environment diff. Cache will not be created.

      call :clean_cache
      call :clean_env

      exit /b 0
    )
  ) else (
    echo INF: Cached Init.
    echo INF: Restore cache from "%INIT_CACHE_ENV_DIR%"

    call :restore_gen || (
      echo WRN: Failed to restore from cache. Calling full init.

      call :clean_cache
      call :clean_env

      call "%~dp0init.full.cmd" || exit /b 1

      exit /b 0
    )
    echo INF: Init restored from cache.
    echo INF: For full initialization, run init with 'no_cache' argument. [no quotes]
    echo INF: To reset the cache, run init with 'clear_cache' arugment. [no quotes]
  )

  @call :dedupe_env
@goto :EOF

:validate_cache
  @call powershell -ExecutionPolicy Bypass -NoLogo -NoProfile -File "%~dp0\.config\.scripts\CleanExpiredCache.ps1" || (
    del /f /q "%RESTORE_CACHED_ENV_PATH%" 1>nul 2>&1
  )

  @if not exist "%RESTORE_CACHED_ENV_PATH%" exit /b 1
@exit /b 0

:gen_cahe_hash
  @if exist "%ENV_CACHE_HASH_SRC_PATH%" del /f /q "%ENV_CACHE_HASH_SRC_PATH%" || exit /b 1

  @for %%i in (%INIT_CACHE_FILES_TO_HASH%) do @(
    if exist "%%~dpnxi" type "%%~dpnxi"
  )>>"%ENV_CACHE_HASH_SRC_PATH%"

  @if not "%NO_RECURSIVE_CACHE%"=="1" if /i not "%NO_RECURSIVE_CACHE%"=="true" (
    for /F "tokens=* delims=" %%i in ('git ls-files **\packages.config') do @(
      type "%%~dpnxi"
    )>>"%ENV_CACHE_HASH_SRC_PATH%"
  )

  @call powershell -ExecutionPolicy Bypass -NoLogo -NoProfile -File "%~dp0\.config\.scripts\GenCacheHash.ps1" || exit /b 1
@goto :EOF

:backup_gen
  @call powershell -ExecutionPolicy Bypass -NoLogo -NoProfile -File "%~dp0\.config\.scripts\GenInitDiffCmd.ps1" || exit /b 1

  @if exist "%INIT_GEN_DIR%" (
    if exist "%INIT_CACHE_GEN_DIR%" rd /s /q "%INIT_CACHE_GEN_DIR%"
    xcopy "%INIT_GEN_DIR%" "%INIT_CACHE_GEN_DIR%\" /e /i /y 1>nul || exit /b 1
  )

  @if exist "%RESTORE_CACHED_ALIASES_PATH%" del /f /q "%RESTORE_CACHED_ALIASES_PATH%"

  @for /f "skip=1 tokens=1,* delims== " %%i in ('alias') do @(
    echo %%i    %%j
  )>>"%RESTORE_CACHED_ALIASES_PATH%"
@exit /b 0

:restore_gen
  @:: Change directory to repo root as init did without cache
  @cd /d "%ROOT%" || exit /b 1

  @echo INF: [%time%] Starting environment restoration
  @call "%RESTORE_CACHED_ENV_PATH%" || exit /b 1
  @echo INF: [%time%] Finished restoration of environment

  @if exist "%INIT_CACHE_GEN_DIR%" (
    if exist "%INIT_GEN_DIR%" rd /s /q "%INIT_GEN_DIR%"
    xcopy "%INIT_CACHE_GEN_DIR%" "%INIT_GEN_DIR%\" /e /i /y 1>nul || exit /b 1
  )
  @echo INF: [%time%] Finished restoration of directories
  @call alias -f "%RESTORE_CACHED_ALIASES_PATH%"
  @echo INF: [%time%] Finished restoration of alias
@exit /b 0

:set_defaults
  @set ROOT=%~dp0
  @set ROOT=%ROOT:~0,-1%

  @set ENV_CACHE_HASH_SRC_FILE=cache.src
  @set ENV_CACHE_HASH_DEST_FILE=cache.hash

  @if not defined INIT_CACHE_FILES_TO_HASH set INIT_CACHE_FILES_TO_HASH=^
    "%ROOT%\init.full.cmd" ^
    "%ROOT%\init.cache.cmd" ^
    "%ROOT%\init.cmd" ^
    "%ROOT%\.corext\corext.config" ^
    "%ROOT%\.config\.scripts\*.ps1"

  @set INIT_GEN_DIR=%ROOT%\.corext\gen
  
  @:: May Use LocalAPPData to not depend on repo and git clean
  @:: We may want to change it to interactive mode to provide cache
  @:: Or use predefined path such as NugetMachineInstallRoot for example
    @if not defined INIT_CACHE_ENV_DIR set INIT_CACHE_ENV_DIR=%ROOT%\.corext\cache

  @if not exist "%INIT_CACHE_ENV_DIR%" md "%INIT_CACHE_ENV_DIR%" 1>nul 2>&1

  @set ENV_CACHE_HASH_SRC_PATH=%INIT_CACHE_ENV_DIR%\%ENV_CACHE_HASH_SRC_FILE%
  @set ENV_CACHE_HASH_DEST_PATH=%INIT_CACHE_ENV_DIR%\%ENV_CACHE_HASH_DEST_FILE%

  @call :gen_cahe_hash || (
    set INIT_CACHE_INVALIDATE=1

    call :clean_cache
    exit /b 1
  )

  @set /p ENV_CACHE_HASH=<"%ENV_CACHE_HASH_DEST_PATH%"

  @set INIT_CACHE_ENV_DIR=%INIT_CACHE_ENV_DIR%\%ENV_CACHE_HASH%
  @set INIT_CACHE_GEN_DIR=%INIT_CACHE_ENV_DIR%\gen

  @set INIT_CACHE_PRE_ENV_PATH=%INIT_CACHE_ENV_DIR%\preinit.env
  @set INIT_CACHE_POST_ENV_PATH=%INIT_CACHE_ENV_DIR%\postinit.env

  @set RESTORE_CACHED_ENV_PATH=%INIT_CACHE_ENV_DIR%\restore.env.cmd
  @set RESTORE_CACHED_ALIASES_PATH=%INIT_CACHE_ENV_DIR%\restore.aliases.pub

  @if not defined INIT_CACHE_EXPIRATION_TIMEOUT set INIT_CACHE_EXPIRATION_TIMEOUT=24

  @if defined INIT_CACHE_INVALIDATE if not "%INIT_CACHE_INVALIDATE%" equ "0" if /i not "%INIT_CACHE_INVALIDATE%"=="false" (
    call :clean_cache
  )

  @if not exist "%INIT_CACHE_ENV_DIR%" md "%INIT_CACHE_ENV_DIR%" 1>nul 2>&1

@goto :EOF

:clean_cache
  @if defined INIT_CACHE_ENV_DIR if exist "%INIT_CACHE_ENV_DIR%" rd /s /q "%INIT_CACHE_ENV_DIR%"
@goto :EOF

:clean_env
  @if exist "%INIT_CACHE_PRE_ENV_PATH%"  del /f /q "%INIT_CACHE_PRE_ENV_PATH%"
  @if exist "%INIT_CACHE_POST_ENV_PATH%" del /f /q "%INIT_CACHE_POST_ENV_PATH%"

  @if exist "%ENV_CACHE_HASH_SRC_PATH%"  del /f /q "%ENV_CACHE_HASH_SRC_PATH%"
  @if exist "%ENV_CACHE_HASH_DEST_PATH%"  del /f /q "%ENV_CACHE_HASH_DEST_PATH%"

  @for /F "tokens=1,* delims==" %%i in ('set ENV_CACHE_')      do @(set %%~i=)
  @for /F "tokens=1,* delims==" %%i in ('set INIT_CACHE_')     do @(set %%~i=)
  @for /F "tokens=1,* delims==" %%i in ('set RESTORE_CACHED_') do @(set %%~i=)

  @(set INIT_GEN_DIR=)
@exit /b 0

:reset_env
  @if not exist "%~dp0.corext\cache" md "%~dp0.corext\cache" 1>nul 2>&1

  @start /wait /i cmd.exe /x /c "set 1>"%~dp0.corext\cache\clean.env"" || exit /b 1

  @if defined INIT_CACHE_VARS_TO_SAVE for %%i in (%INIT_CACHE_VARS_TO_SAVE%) do @(
    if defined %%~i set %%~i
  ) 1>>"%~dp0.corext\cache\clean.env"

@:: Reset incorrectly set platform environment variable. It is set on some new SKU for developers.
@:: As a result new hires and devs who got a recent machine update get a broken build with error:
@::     The Platform for project 'XYZ' is invalid.  Platform='BWS'. You may be seeing this 
@::     message because you are trying to build a project without a solution file, and have 
@::     specified a non-default Platform that doesn't exist for this project.  
  @call findstr /v /b /i /c:"Platform=" "%~dp0.corext\cache\clean.env" 1>"%~dp0.corext\cache\clean_filtered.env"
  @for /F "usebackq  tokens=1,* delims==" %%i in (`set`) do @(set %%~i=)
  @for /F "usebackq tokens=1 delims=" %%i in ("%~dp0.corext\cache\clean_filtered.env") do @(set %%~i)
@exit /b 0

:dedupe_env
  @call :clean_env

  @if not exist "%~dp0.corext\cache" md "%~dp0.corext\cache" 1>nul 2>&1

  @call powershell -ExecutionPolicy Bypass -NoLogo -NoProfile -File "%~dp0\.config\.scripts\GenInitDedupedEnv.ps1" -Path "%~dp0.corext\cache\postinit.env.cmd" && (
    @call "%~dp0.corext\cache\postinit.env.cmd"
  )

  @if exist "%~dp0.corext\cache\postinit.env.cmd" del /f /q "%~dp0.corext\cache\postinit.env.cmd"
@exit /b 0
