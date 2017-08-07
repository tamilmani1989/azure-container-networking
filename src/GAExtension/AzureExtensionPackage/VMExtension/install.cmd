SET /A errno=0
start /wait .\vcredist\vc_redist.x64.exe /install /passive /norestart /log log.txt
IF %ERRORLEVEL% NEQ 0 (
   SET /A errno=%ERRORLEVEL%
   goto :EOF
)
REM Aquarius\\Aquarius.exe -install
:EOF
EXIT /B %errno%