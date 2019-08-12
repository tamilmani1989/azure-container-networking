@perl -x "%~f0" %*
@exit /b %errorlevel%
#!perl
#line 5

#
# CreateBuildShare
#
#   CreateBuildShare from "$(BinariesBuildTypeArchDirectory)\HostNetAgent" to $packageDestDir
#   which will then be signed for PfClient.
#

use strict;
use warnings FATAL => 'uninitialized';
use File::Path;
# use Logmsg;

my $arrSize = @ARGV;

my $packageDestDir = $ARGV[0];

#logmsg("Start CreateBuildShare.cmd");
#logmsg("dest: $packageDestDir");

Run_cmd("copy /Y .\\start.bat $packageDestDir",1,1);
Run_cmd("copy /Y .\\ServiceConfig.ini $packageDestDir",1,1);

sub Run_cmd
{
    my $Command = shift;
    my $ignoreErrorLevel = shift;
    my $Quiet = shift;
    # $Command = ExpandEnvVariables($Command);
    if (!$Quiet)
    {
        timemsg("Running $Command");
    }
    my $StartTime = time();
    my $result = `$Command`;
    my $EndTime = time();
    my $ExitCode = ($? >> 8);

    my $ElapsedTime = ($EndTime - $StartTime);
    if (!$Quiet)
    {
        if ($ElapsedTime > 30)
        {
            timemsg("$Command exiting with ExitCode:$ExitCode; elapsed time " . substr(($ElapsedTime / 60.0), 0, 5) . " minutes");
        }
        else
        {
            timemsg("$Command exiting with ExitCode:$ExitCode; elapsed time " . $ElapsedTime . " seconds");
        }
    }

    if (($ExitCode != 0) && (!$ignoreErrorLevel))
    {
       errormsg("error running '$Command' : $ExitCode");
       errormsg("  Result: $result");
       Carp::confess();      
    }

    if (!$Quiet)
    {
        #logmsg($result);
    }

    # return success
    my @myReturn = ($ExitCode, $result);
    return @myReturn ;
}

sub ExpandEnvVariables
{
   my $path = shift;
   $path = Win32::ExpandEnvironmentStrings($path);
   return $path;
}


1;