//-----------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation.  All rights reserved.
//-----------------------------------------------------------------------------

using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Runtime.CompilerServices;
using System.Text;
using System.Threading.Tasks;

namespace CnsWrapperService
{
    /// <summary>
    /// TelemetrySettingsService reads settings from XMl files and converts them to a config json file for cns
    /// </summary>
    class TelemetrySettingsService
    {
        /// <summary>
        /// Config file name to write to
        /// If later you need to write given a path, this can be the path
        /// </summary>
        private static string _cnsConfigFileLocation = "cns_config.json";

        /// <summary>
        /// Settings object to hold cns config settings
        /// </summary>
        private static TelemetrySettingsModel _settings = new TelemetrySettingsModel();

        /// <summary>
        /// Creates cns_config.json file
        /// </summary>
        public static void CreateCNSConfigFile()
        {
            GetServiceSettings();
            CheckSettingsValues();
            WriteCnsConfigFile();
        }

        /// <summary>
        /// Reads cns settings from XML files and saves them in TelemetrySettingsModel object.
        /// Tested by trying default values, zero values and arbitrary values in XML file.
        /// </summary>
        private static void GetServiceSettings()
        {
            Program.WrapperSetup.GetLogger().LogInfo("Reading cns_config values from overriden values of ApplicationManifest.xml.");
            _settings.DisableAll = WrapperConfiguration.DisableAll;
            _settings.DisableTrace = WrapperConfiguration.DisableTrace;
            _settings.DisableMetric = WrapperConfiguration.DisableMetric;
            _settings.DisableEvent = WrapperConfiguration.DisableEvent;
            _settings.TelemetryBatchSizeBytes = WrapperConfiguration.TelemetryBatchSizeBytes;
            _settings.TelemetryBatchIntervalInSecs = WrapperConfiguration.TelemetryBatchIntervalInSecs;
            _settings.HeartBeatIntervalInMins = WrapperConfiguration.HeartBeatIntervalInMins;
            _settings.DisableMetadataRefreshThread = WrapperConfiguration.DisableMetadataRefreshThread;
            _settings.RefreshIntervalInSecs = WrapperConfiguration.RefreshIntervalInSecs;
            _settings.DebugMode = WrapperConfiguration.DebugMode;
            _settings.SnapshotIntervalInMins = WrapperConfiguration.SnapshotIntervalInMins;
        }

        /// <summary>
        /// Checks if any settings are 0 and if so, changes them to default setting values
        /// Tested by trying default values, zero values and arbitrary values in XML file.
        /// </summary>
        private static void CheckSettingsValues()
        {
            Program.WrapperSetup.GetLogger().LogInfo("Checking for zero values of cns_config settings. If zero, making them default values.");
            if (_settings.TelemetryBatchSizeBytes == 0)
            {
                _settings.TelemetryBatchSizeBytes = Constants.DefaultTelemetryBatchSizeBytes;
            }

            if (_settings.TelemetryBatchIntervalInSecs == 0)
            {
                _settings.TelemetryBatchIntervalInSecs = Constants.DefaultTelemetryBatchIntervalInSecs;
            }

            if (_settings.HeartBeatIntervalInMins == 0)
            {
                _settings.HeartBeatIntervalInMins = Constants.DefaultHeartBeatIntervalInMins;
            }

            if (_settings.RefreshIntervalInSecs == 0)
            {
                _settings.RefreshIntervalInSecs = Constants.DefaultRefreshIntervalInSecs;
            }

            if (_settings.SnapshotIntervalInMins == 0)
            {
                _settings.SnapshotIntervalInMins = Constants.DefaultSnapshotIntervalInMins;
            }
        }

        /// <summary>
        /// Serializes TelemetrySettingsModel to JSON and writes to file name specified by _cnsConfigFileLocation
        /// </summary>
        private static void WriteCnsConfigFile()
        {
            // Write the cns_config.json with settings read from xml files.
            try
            {
                Program.WrapperSetup.GetLogger().LogInfo("Writing the cns_config.json with settings read from xml files.");
                // The root variable is so that "TelemetrySettings" appears in JSON file
                var root = new
                {
                    TelemetrySettings = _settings
                };
                string json = JsonConvert.SerializeObject(root, Formatting.Indented);
                using (FileStream fs = File.Create(_cnsConfigFileLocation))
                {
                    byte[] info = new UTF8Encoding(true).GetBytes(json);
                    fs.Write(info, 0, info.Length);
                }
            } catch (Exception ex)
            {
                Program.WrapperSetup.GetLogger().LogInfo("Error when writing the cns_config.json with settings read from xml files. Exception: " + ex.Message);
                throw ex;
            }
        }
    }
}
