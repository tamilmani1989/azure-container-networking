doskey bin=cd %OUTPUTROOT%\debug-AMD64
doskey aq=cd %SRCROOT%
doskey root=cd %SRCROOT%\..

set MsBuildArgs=/verbosity:minimal

REM TrackFileAccess access isn't working on VC14
set TrackFileAccess=false

call %COREXT_VERSION_FORMAT_PATH%\GenerateVersions.cmd

REM Add default QuickBuild flags for both local and cloud builds
SET QUICKBUILD_ARGS=-Jan2018Flags
