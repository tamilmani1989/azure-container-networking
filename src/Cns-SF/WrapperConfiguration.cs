using System;
using System.Collections.Generic;
using System.Fabric;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace CnsWrapperService
{
    class WrapperConfiguration
    {
        public static TimeSpan ThreadWaitTimeout
        {
            get { return GetTimeSpanConfigurationValue(Constants.ConfigPackage, "WrapperThreadWaitTimeout", Constants.DefaultWrapperThreadWaitTimeout); }
        }

        public static TimeSpan MainLoopTimeout
        {
            get { return GetTimeSpanConfigurationValue(Constants.ConfigPackage, "WrapperMainLoopTimeout", Constants.DefaultWrapperMainLoopTimeout); }
        }

        public static TimeSpan MainThreadShutdownCheck
        {
            get { return GetTimeSpanConfigurationValue(Constants.ConfigPackage, "WrapperMainThreadShutdownCheck", Constants.DefaultWrapperMainThreadShutdownCheck); }
        }

        public static TimeSpan ShutdownThreadWaitTimeout
        {
            get { return GetTimeSpanConfigurationValue(Constants.ConfigPackage, "WrapperShutdownThreadWaitTimeout", Constants.DefaultWrapperShutdownThreadWaitTimeout); }
        }

        public static bool EnablePortOverride
        {
            get { return GetBoolConfigurationValue(Constants.ConfigPackage, "EnablePortOverride", false); }
        }

        public static bool StartCnm
        {
            get { return GetBoolConfigurationValue(Constants.ConfigPackage, "StartCnm", false); }
        }

        public static bool SetupHnsAtStartup
        {
            get { return GetBoolConfigurationValue(Constants.ConfigPackage, "SetupHnsAtStartup", false); }
        }

        public static string DefaultExtNetworkType
        {
            get { return GetStringConfigurationValue(Constants.ConfigPackage, "DefaultExtNetworkType", "l2bridge"); }
        }

        public static bool EnableTelemetry
        {
            get { return GetBoolConfigurationValue(
                Constants.ConfigPackage,
                "EnableTelemetry",
                Constants.EnableTelemetry); }
        }

        public static int HttpConnectionTimeoutInSeconds
        {
            get
            {
                return GetIntConfigurationValue(
                    Constants.ConfigPackage,
                    "HttpConnectionTimeoutInSeconds",
                    Constants.DefaultHttpConnectionTimeoutInSeconds);
            }
        }

        public static int HttpResponseHeaderTimeoutInSeconds
        {
            get
            {
                return GetIntConfigurationValue(
                    Constants.ConfigPackage,
                    "HttpResponseHeaderTimeoutInSeconds",
                    Constants.DefaultHttpResponseHeaderTimeoutInSeconds);
            }
        }

        public static string StoreFilePath
        {
            get { return GetStringConfigurationValue(Constants.ConfigPackage, "StoreFilePath", ""); }
        }

        private static string GetStringConfigurationValue(string section, string parameter, string defaultValue)
        {
            string configValue = GetConfigurationValue(section, parameter);

            if (string.IsNullOrEmpty(configValue))
            {
                return defaultValue;
            }

            return configValue;
        }

        private static bool GetBoolConfigurationValue(string section, string parameter, bool defaultValue)
        {
            bool result;
            string configValue = GetConfigurationValue(section, parameter);

            if (Boolean.TryParse(configValue, out result))
            {
                return result;
            }

            return defaultValue;
        }

        private static TimeSpan GetTimeSpanConfigurationValue(string section, string parameter, TimeSpan defaultTimeSpan)
        {
            var configTimeSpan = GetConfigurationValue(section, parameter);

            if (configTimeSpan == null)
            {
                return defaultTimeSpan;
            }

            TimeSpan returnResult;
            if (!TimeSpan.TryParse(configTimeSpan, out returnResult))
            {
                returnResult = defaultTimeSpan;
            }

            return returnResult;
        }

        private static int GetIntConfigurationValue(string section, string parameter, int defaultIntValue)
        {
            int result;
            string configValue = GetConfigurationValue(section, parameter);

            if(!string.IsNullOrEmpty(configValue) && int.TryParse(configValue, out result))
            {
                return result;
            }

            return defaultIntValue;
        }


        private static string GetConfigurationValue(string section, string parameter)
        {
            var config = FabricRuntime.GetActivationContext().GetConfigurationPackageObject(Constants.ConfigPackage);

            if (config == null ||
                config.Settings == null ||
                config.Settings.Sections[section] == null ||
                config.Settings.Sections[section].Parameters[parameter] == null)
            {
                return null;
            }

            return config.Settings.Sections[section].Parameters[parameter].Value;
        }
    }
}
