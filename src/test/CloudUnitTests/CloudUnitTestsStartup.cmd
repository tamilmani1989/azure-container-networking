

REM unset RdRoleId to decouple from the runtime of the worker that's running us
set RdRoleId=

REM Enable the NetTcpPortSharing service 
sc config nettcpportsharing start= auto 
sc start nettcpportsharing

REM disable signing checks to allow developer builds
CloudUnitTests\sn -Vr *,31bf3856ad364e35

REM detect the OS and optionally get rid of webservices.dll as it exists in 2008r2/7 and breaks tests in that case
systeminfo | find "OS Name" > %TEMP%\osname.txt
FOR /F "usebackq delims=: tokens=2" %%i IN (%TEMP%\osname.txt) DO set vers=%%i

echo %vers% | find "Windows 7" > nul
if %ERRORLEVEL% == 0 goto ver_7

echo %vers% | find "Windows Server 2008 R2" > nul
if %ERRORLEVEL% == 0 goto ver_2008_r2

echo %vers% | find "Windows Server 2008" > nul
if %ERRORLEVEL% == 0 goto ver_2008

echo %vers% | find "Windows Serverr 2008" > nul
if %ERRORLEVEL% == 0 goto ver_2008

echo %vers% | find "Windows Vista" > nul
if %ERRORLEVEL% == 0 goto ver_vista

goto exit

:ver_vista
:ver_2008
:Run Windows Server 2008 specific commands here.
	echo Windows Server 2008
goto exit

:ver_2008_r2
:ver_7
:Run Windows Server 2008 R2 or Win 7 specific commands here.
	echo Windows Server 2008 R2 or Win7
    sc config aelookupsvc start= auto
    net start aelookupsvc
REM See CL 253751.
REM ren webservices.dll webservices.dll.preR2
goto exit

:exit

echo %~0 exit %errorlevel%.
REM Reset the error level from the last command
exit /B 0
