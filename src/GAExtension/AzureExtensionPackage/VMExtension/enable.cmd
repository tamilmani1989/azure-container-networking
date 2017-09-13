SET /A errno=0
SET /A ERROR_AQUARIUS_FAILURE=6789
start /wait net stop AquariusSvc
start /wait net start AquariusSvc
timeout /t 200 /nobreak > NUL
start /wait Aquarius\Aqua.RestClient.exe
IF %ERRORLEVEL% NEQ 0 (
   SET /A errno=%ERROR_AQUARIUS_FAILURE=6789%
   goto :EOF
)
:EOF
echo %errno%
EXIT /B %errno%

