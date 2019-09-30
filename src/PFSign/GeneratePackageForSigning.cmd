set PARENT=%1

echo %PARENT%
mkdir "%PARENT%\HostNetAgentCopy"
move "%PARENT%\HostNetAgent" "%PARENT%\HostNetAgentCopy"
rename "%PARENT%\HostNetAgentCopy" "HostNetAgent"
