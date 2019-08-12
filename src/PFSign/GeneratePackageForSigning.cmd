set OUTPUTROOT=%1

echo %OUTPUTROOT%
mkdir "%OUTPUTROOT%\HostNetAgent"
move "%OUTPUTROOT%\*" "%OUTPUTROOT%\HostNetAgent\"
