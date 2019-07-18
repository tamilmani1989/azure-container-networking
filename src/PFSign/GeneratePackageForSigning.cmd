set OUTPUTROOT=%1

mkdir "%OUTPUTROOT%\HostNetAgentCopy"
mv "%OUTPUTROOT%\*" "%OUTPUTROOT%\HostNetAgentCopy\"
rename "%OUTPUTROOT%\HostNetAgentCopy" "HostNetAgent"
