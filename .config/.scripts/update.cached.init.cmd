:main
  @call :set_defaults

  @call :verify_update || (
    echo WRN: Update was not done. Repo was not modified.
    exit /b 1
  )

  @echo INF: Init upgrade started.

  @for %%i in (prepare_repo copy_files stage_files) do @(
    call :%%~i || (
      echo FATAL ERROR: Target "%%~i" failed! 1>&2

      call :rollback_on_fail

      echo.
      echo Please fix issues and try to upgrade again or contact wabldsup@microsoft.com for assistance. 1>&3

      exit /b 1
    )
  )

  @echo INF: Init upgrade successfully completed.
  @echo INF: Please verify staged files, commit and push when you are ready.
@goto :EOF

:stage_files
  @for %%i in (%full_filelist_to_add%) do @(
    git add -f "%%~i" || (
      echo ERR: Failed to add file "%%~i" 1>&2
    )
  )
@goto :EOF

:prepare_repo
  @call git reset -- "%root_file_source%" 1>nul 2>&1 || (
    echo ERR: Failed to reset "%root_file_source%". 1>&2
    exit /b 1
  )

  @call git checkout -- "%root_file_source%" || (
    echo ERR: Failed to checkout "%root_file_source%". 1>&2
    exit /b 1
  )
@goto :EOF

:copy_files
  @if not exist "%root_file_source%" (
    echo ERR: There is no "%root_file_source%". Are you in the repo root? 1>&2
    exit /b 1
  )

  @set _copy_files_failed=0

  @if not exist "%root_file_target%" (
    move /y "%root_file_source%" "%root_file_target%" 1>nul 2>&1 || (
      echo ERR: Unable to move "%root_file_source%" to "%root_file_target%". 1>&2
      exit /b 1
    )
  )

  @for %%i in (%root_file_list_to_copy% %ps_file_list_to_copy%) do @(
    if not exist "%%~dpi" md "%%~dpi" 1>nul 2>&1

    copy /y /z "%~dp0..\..\%%~i" "%%~dpnxi" 1>nul 2>&1 || (
      echo ERR: Failed to copy "%~dp0..\..\%%~i" to "%%~dpnxi". 1>&2
      set _copy_files_failed=1
    )
  )
@exit /b %_copy_files_failed%

:verify_update
  @if not exist ".git" (
    echo ERR: Failed to find ".git". Are you in git repo root? 1>&2
    exit /b 1
  )
@goto :EOF

:rollback_on_fail
  @set _rollback_on_fail_failed=0

  @for %%i in (%full_filelist_to_add%) do @(
    if exist "%%~i" (
      git reset -- "%%~i" 1>nul 2>&1 || (
        echo ERR: Failed to reset file "%%~i" 1>&2
        set _rollback_on_fail_failed=1
      )

      del /f /q "%%~i"
    )
  )

  @if exist "%root_file_target%" (
    move "%root_file_target%" "%root_file_source%" || (
      echo WRN: Unable to move "%root_file_target%" to "%root_file_source%".
      echo INF: Trying to checkout file

      call git checkout -- "%root_file_source%" || (
        echo ERR: Failed to checkout "%root_file_source%". 1>&2
        set _rollback_on_fail_failed=1
      )
    )
  ) else (
    call git checkout -- "%root_file_source%" || (
      echo ERR: Failed to checkout "%root_file_source%". 1>&2
      set _rollback_on_fail_failed=1
    )
  )

  @if "%_rollback_on_fail_failed%"=="1" (
    echo ERR: Faled to recover repo after upgrade failure. 1>&2
    echo INF: Please fix it manually.
  )
@exit /b %_rollback_on_fail_failed%

:set_defaults
  @set root_file_source=init.cmd
  @set root_file_target=init.full.cmd

  @set root_file_list_to_copy= ^
    %root_file_source% ^
    init.cache.cmd

  @set ps_file_list_to_copy= ^
    .config\.scripts\CleanExpiredCache.ps1 ^
    .config\.scripts\GenCacheHash.ps1 ^
    .config\.scripts\GenInitDedupedEnv.ps1 ^
    .config\.scripts\GenInitDiffCmd.ps1

  @set full_filelist_to_add= ^
    %root_file_list_to_copy% ^
    %ps_file_list_to_copy% ^
    %root_file_target%

@goto :EOF
