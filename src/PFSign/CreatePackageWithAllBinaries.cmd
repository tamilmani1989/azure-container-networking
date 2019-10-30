set SRC=%1
set DES=%2
set CopyStartBatAndServiceConf=%3

echo %SRC%
echo %DES%
mkdir %DES%
xcopy /s /y /h %SRC% %DES%

IF %CopyStartBatAndServiceConf% == true (
xcopy /s /y /h "start.bat" %DES%
xcopy /s /y /h "ServiceConfig.ini" %DES%
xcopy /s /y /h "..\HostPluginPackage\NetAgentHostPlugin.PluginSpecific.xml" %DES%
@powershell -Command "get-content %DES%/NetAgentHostPlugin.PluginSpecific.xml | %%{$_ -replace \"NetAgentGcsRegion\",\"Region\"} > %DES%/PluginSpecificTemplate.xml"
)