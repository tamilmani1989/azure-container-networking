SET /A errno=0
Aquarius\\Aquarius.exe -install
SET /A errno=%ERRORLEVEL%
EXIT /B %errno%