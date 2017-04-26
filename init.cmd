:main
  @if /i "%~1"=="no_cache" set DO_NOT_USE_CACHE=1

  @if /i "%~1"=="clear_cache" if exist "%~dp0.corext\cache" (
    for /D %%i in ("%~dp0.corext\cache\*") do @(
      rd /s /q "%%~dpnxi"
    )
  )

  @if defined BuildTracker set DO_NOT_USE_CACHE=1
  @if defined QBUILD_DISTRIBUTED set DO_NOT_USE_CACHE=1
  
  @:: Set the current folder to be enlistment root
  @:: Needed because WS2012R2 starts admin shortcuts in windows folder
    @cd /d "%~dp0"
  
  @if not defined INIT_CACHE_VARS_TO_SAVE (
    set INIT_CACHE_VARS_TO_SAVE=^
      DO_NOT_USE_CACHE ^
      INIT_CACHE_ENV_DIR ^
      INIT_CACHE_INVALIDATE ^
      INIT_CACHE_FILES_TO_HASH ^
      INIT_CACHE_EXPIRATION_TIMEOUT ^
      NO_RECURSIVE_CACHE
  )

  @if not "%DO_NOT_USE_CACHE%"=="1" if /i not "%DO_NOT_USE_CACHE%"=="true" (
    call "%~dp0init.cache.cmd"
    goto :EOF
  )

  @call "%~dp0init.full.cmd"
@goto :EOF
