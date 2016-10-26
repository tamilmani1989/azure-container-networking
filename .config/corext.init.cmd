doskey bin=cd %OUTPUTROOT%\debug-AMD64
doskey aq=cd %SRCROOT%
doskey root=cd %SRCROOT%\..

set MsBuildArgs=/verbosity:minimal

call %COREXT_VERSION_FORMAT_PATH%\GenerateVersions.cmd