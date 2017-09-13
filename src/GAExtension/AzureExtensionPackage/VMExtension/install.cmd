SET /A errno=0
start /wait .\vcredist\vc_redist.x64.exe /install /passive /norestart /log log.txt
IF %ERRORLEVEL% NEQ 0  (
IF %ERRORLEVEL% NEQ 1638 ( 
   SET /A errno=%ERRORLEVEL%
   goto :EOF
))
Aquarius\\Aquarius.exe -install
SET /A errno=%ERRORLEVEL%
:EOF
EXIT /B %errno%