@if "%_echo%"=="" echo off
GOTO :Start
#
#  Corext Environment Initialization
#
#    Setup the environment to be responsive and contained to the commandline
#    https://microsoft.sharepoint.com/teams/corext/LivingDocs/CorextInitialization.aspx
#
:Start

@REM -----------------------------------------------------------------------------------
@REM -- RepoConfig - minimal tool config [anything of size should be packed/shared]
@REM -----------------------------------------------------------------------------------

@REM -- Product being built (CorextBranch when not in Git)
SET CorextProduct=CoreXT

@REM -- Pull packages and setup the enlistment
CALL :ConfigureCorext
IF ERRORLEVEL 1 GOTO :EOF

CALL :ExpandPackages
IF ERRORLEVEL 1 GOTO :EOF

CALL :BootstrapCorext
IF ERRORLEVEL 1 GOTO :EOF

@REM -- Configure purger with our preferences
SET PURGER=%PURGER% -si -norecycle -i %PackageAddressGenDir:\=\\%.*

@REM -- Corext-Repo-Specific.  You do not want this. Separate file keeps corext-repo info mostly isolated across fork/clone operations
IF EXIST %ROOT%\.config\corext.init.cmd call %ROOT%\.config\corext.init.cmd

@REM -----------------------------------------------------------------------------------
GOTO :EOF





@REM -----------------------------------------------------------------------------------
@REM -----------------------------------------------------------------------------------
:ConfigureCorext
@REM -- Standard Corext Bootstrapper
FOR %%I IN (%~dp0\.corext\..) DO @SET ROOT=%%~fI
SET BaseDir=%ROOT%
SET CoreXTConfig=%ROOT%\.corext
IF NOT DEFINED CoreXTConfigFile SET CoreXTConfigFile=%CoreXTConfig%\corext.config
GOTO :EOF


@REM -----------------------------------------------------------------------------------
@REM -----------------------------------------------------------------------------------
:ExpandPackages
ECHO Generating Expanded CoreXT config

SET ExpandedCoreXTConfigFile=%ROOT%\.gen\corext.expanded.config
SET ResolveCorextConfig=%ROOT%\.config\corext.ResolveCorext.config
@REM To Prevent updating the ResovleCoreXTConfig in the repo, try to use one in the NugetMachineInstallRoot
IF EXIST "%NugetMachineInstallRoot%" (
  IF NOT EXIST %NugetMachineInstallRoot%\corext.ResolveCorext.config COPY %ROOT%\.config\corext.ResolveCorext.config %NugetMachineInstallRoot%\corext.ResolveCorext.config
  SET "ResolveCorextConfig=%NugetMachineInstallRoot%\corext.ResolveCorext.config"
)
SET TEMPCoreXTConfigFile=%CoreXTConfigFile%
SET CoreXTConfigFile=%ResolveCorextConfig%

SET OnlineCommand=
SET Pipe=

IF /I "%IsOfficialBuild%"=="true" SET OnlineCommand=--Online
IF /I "%QBUILD_DISTRIBUTED%"=="1" SET UpdatePackageVersionCommand=--DoNotUpdatePackageVersion

REM Piping to null to keep the output clean, but still show warnings and errors
CALL :BootstrapCorext || (
    SET CoreXTConfigFile=%TEMPCoreXTConfigFile%
    EXIT /B 1
)

SET CoreXTConfigFile=%TEMPCoreXTConfigFile%
CALL ResolveCorextConfig --SourceRoot %ROOT%\src --ExpandedCorextConfig %ExpandedCoreXTConfigFile% %OnlineCommand% --ExpandPackages --AutoUpdateResolveCoreXTConfig %UpdatePackageVersionCommand%
SET CoreXTConfigFile=%ExpandedCoreXTConfigFile%
GOTO :EOF

@REM -----------------------------------------------------------------------------------
@REM -----------------------------------------------------------------------------------
:BootstrapCorext
SET PackageAddressGenDir=%ROOT%\.corext\gen
%CoreXTConfig%\corextBoot.exe init -bootstrap
IF ERRORLEVEL 1 ECHO [Error] CoreXT could not be properly initialized.  This enlistment window will not work. & EXIT /B 1
CALL %PackageAddressGenDir%\init.cmd -recurse
IF DEFINED TEMPCoreXTConfigFile SET CoreXTConfigFile=%TEMPCoreXTConfigFile%
GOTO :EOF
