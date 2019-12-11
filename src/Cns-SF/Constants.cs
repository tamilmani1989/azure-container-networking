using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace CnsWrapperService
{
    /// <summary>
    /// Constant values used in CNS project
    /// </summary>
    internal class Constants
    {
        #region Global constants

        internal const string ServiceType = "CnsWrapperServiceType";

        internal const string ConfigPackage = "Config";

        internal const string WrapperSetupNameCmdLineArgument = "-WrapperSetupType";

        internal const string WrapperLoggerNameCmdLineArgument = "-WrapperLoggerType";

        #endregion

        #region Default configuration values

        internal static readonly string LogPath = "wrapper.log";

        internal static readonly TimeSpan DefaultWrapperMainThreadShutdownCheck = TimeSpan.FromSeconds(10);

        internal static readonly TimeSpan DefaultWrapperThreadWaitTimeout = TimeSpan.FromSeconds(10);

        internal static readonly TimeSpan DefaultWrapperShutdownThreadWaitTimeout = DefaultWrapperThreadWaitTimeout;

        internal static readonly TimeSpan DefaultWrapperMainLoopTimeout = TimeSpan.FromMilliseconds(DefaultWrapperThreadWaitTimeout.TotalMilliseconds / 2);

        internal static readonly int DefaultHttpConnectionTimeoutInSeconds = 5;

        internal static readonly int DefaultHttpResponseHeaderTimeoutInSeconds = 120;

        internal static readonly bool EnableTelemetry = true;
        
        #endregion
    }
}
