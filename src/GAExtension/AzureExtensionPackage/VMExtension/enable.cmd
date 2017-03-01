SET /A errno=0
SET /A ERROR_SERVICE_TUNNEL_EXTENSION_FAILURE=6789
taskkill /IM ServiceTunnelExtension.exe /f
Aquarius\Plugins\ServiceTunnelExtension\Code\ServiceTunnelExtension.exe > ServiceTunnelExtension.log
IF %ERRORLEVEL% NEQ 0 (
   SET /A errno=%ERROR_SERVICE_TUNNEL_EXTENSION_FAILURE%
   goto :EOF
)

:EOF
EXIT /B %errno%

