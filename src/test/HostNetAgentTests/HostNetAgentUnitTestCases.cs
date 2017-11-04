//-----------------------------------------------------------------------
// <copyright file="HostNetAgentUnitTestCases.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// 
// </summary>
// 
//-----------------------------------------------------------------------
namespace Microsoft.WindowsAzure.Networking.Aquarius.HostNetAgentTests
{
    using System;
    using System.Collections.Generic;
    using System.Diagnostics;
    using System.IO;
    using System.Linq;
    using System.Net;
    using Newtonsoft.Json;
    using Newtonsoft.Json.Linq;
    using VisualStudio.TestTools.UnitTesting;

    /// <summary>
    /// Class definition for HostNetAgentUnitTestCases
    /// </summary>
    [TestClass]
    public class HostNetAgentUnitTestCases
    {
        /// <summary>
        /// Guest OS Version.
        /// </summary>
        private static readonly string GuestOSVersion = "GuestOSVersion";

        /// <summary>
        /// Guest OS Family.
        /// </summary>
        private static readonly string GuestOSFamily = "GuestOSFamily";

        /// <summary>
        /// Driver install not attempted.
        /// </summary>
        private static readonly string DriverInstallNotAttempted = "DriverInstallNotAttempted";

        /// <summary>
        /// Driver install attempted.
        /// </summary>
        private static readonly string DriverInstallAttempted = "DriverInstallAttempted";

        /// <summary>
        /// Driver install succeeded string.
        /// </summary>
        private static readonly string DriverInstallSucceeded = "DriverInstallSucceeded";

        /// <summary>
        /// Driver install failed string.
        /// </summary>
        private static readonly string DriverInstallFailed = "DriverInstallFailed";

        /// <summary>
        /// Total VF devices count string.
        /// </summary>
        private static readonly string TotalVFDevicescount = "TotalVFDevicesCount";

        /// <summary>
        /// Non functional VF devices count string.
        /// </summary>
        private static readonly string NonFunctionalVFDevicesCount = "NonFunctionalVFDevicesCount";

        /// <summary>
        /// InstallationErrorCode string
        /// </summary>
        private static readonly string InstallationErrorCode = "InstallationErrorCode";

        /// <summary>
        /// Host net agent process name.
        /// </summary>
        private static readonly string HostNetAgentProcessName = "HostNetAgent";

        /// <summary>
        /// Location of executable.
        /// </summary>
        private readonly string exeDir;

        /// <summary>
        /// Version file path.
        /// </summary>
        private readonly string versionFile;

        /// <summary>
        /// Base Get Url.
        /// </summary>
        private readonly string baseGetUrl;

        /// <summary>
        /// Base Post Url.
        /// </summary>
        private readonly string basePostUrl;

        /// <summary>
        /// Base Post Url.
        /// </summary>
        private readonly string healthReportUrl;

        /// <summary>
        /// Get package version Url.
        /// </summary>
        private readonly string getVersionUrl;

        /// <summary>
        /// Get Package URIs path.
        /// </summary>
        private readonly string getPackageUrisUrl;

        /// <summary>
        /// Log path.
        /// </summary>
        private readonly string logPath;

        /// <summary>
        /// Document path.
        /// </summary>
        private readonly string docPath;

        /// <summary>
        /// Config file path.
        /// </summary>
        private readonly string configFile;

        /// <summary>
        /// Initializes a new instance of the <see cref="HostNetAgentUnitTestCases" /> class
        /// </summary>
        public HostNetAgentUnitTestCases()
        {
            this.exeDir = AppDomain.CurrentDomain.BaseDirectory;
            this.versionFile = Path.Combine(this.exeDir, "Documents", "version.txt");
            this.baseGetUrl = "http://localhost:8019/hostnetagent?container=1234";
            this.getVersionUrl = this.baseGetUrl + "&type=getpackageversion";
            this.getPackageUrisUrl = this.baseGetUrl + "&type=getpackageuris";
            this.logPath = Path.Combine(this.exeDir, "Logs");
            this.docPath = Path.Combine(this.exeDir, "Documents");
            this.basePostUrl = "http://localhost:8019/hostnetagent/container/1234";
            this.healthReportUrl = this.basePostUrl + "/healthreport";
            this.configFile = Path.Combine(this.exeDir, "NetAgentHostPlugin.PluginSpecific.xml");
        }

        /// <summary>
        /// Helper to stop host network agent.
        /// </summary>
        public static void StopHostNetAgent()
        {
            Process[] processesByName = Process.GetProcessesByName(HostNetAgentProcessName);
            foreach (Process process in processesByName)
            {
                process.Kill();
            }
        }

        /// <summary>
        /// Assembly cleanup.
        /// </summary>
        [AssemblyCleanup]
        public static void AssemblyCleanup()
        {
            StopHostNetAgent();
        }

        /// <summary>
        /// Helper to create health report object.
        /// </summary>
        /// <returns>Create health report object.</returns>
        public JObject CreateHealthReport()
        {
            return new JObject(
                new JProperty(GuestOSFamily, "Windows"),
                new JProperty(GuestOSVersion, "10.0"),
                new JProperty(DriverInstallNotAttempted, true),
                new JProperty(DriverInstallSucceeded, true),
                new JProperty(DriverInstallFailed, true),
                new JProperty(DriverInstallAttempted, true),
                new JProperty(NonFunctionalVFDevicesCount, 5),
                new JProperty(TotalVFDevicescount, 10),
                new JProperty(InstallationErrorCode, 101));
        }

        /// <summary>
        /// Test to validate version
        /// </summary>
        [TestMethod]
        public void TestValidateVersion()
        {
            this.StartHostNetAgent(this.logPath, this.docPath, this.configFile);
            string versionFromFile = System.IO.File.ReadAllText(this.versionFile);
            string version = this.DownloadString(this.getVersionUrl);
            Assert.AreEqual(version, versionFromFile, string.Format("Mismatch in version. Expected;{0} Recieved;{1}", versionFromFile, version));
        }

        /// <summary>
        /// Test to validate get package URI's
        /// </summary>
        [TestMethod]
        public void TestValidateGetPackageUris()
        {
            IEnumerable<string> docs = Directory.EnumerateFiles(this.docPath, "*.zip");
            this.StartHostNetAgent(this.logPath, this.docPath, this.configFile);
            var packageUris = this.DownloadSerializedJsonData<Dictionary<string, string>>(this.getPackageUrisUrl);
            foreach (string doc in docs)
            {
                Console.WriteLine("Expected docs:{0}", doc);
            }

            foreach (KeyValuePair<string, string> packageKvp in packageUris)
            {
                Console.WriteLine("Received Resource Name:{0}, Uri:{1}", packageKvp.Key, packageKvp.Value);
            }

            foreach (string doc in docs)
            {
                string fileName = doc.Substring(this.docPath.Length + 1);
                Assert.IsTrue(packageUris.ContainsKey(fileName));
            }
        }

        /// <summary>
        /// Test to validate packages
        /// </summary>
        [TestMethod]
        public void TestValidateGetPackage()
        {
            this.StartHostNetAgent(this.logPath, this.docPath, this.configFile);
            var packageUris = this.DownloadSerializedJsonData<Dictionary<string, string>>(this.getPackageUrisUrl);
            string fileToDownLoad = packageUris.First().Key;
            var getPackageUrl = this.baseGetUrl + string.Format("&type={0}", packageUris.First().Key);
            string expectedContent = System.IO.File.ReadAllText(Path.Combine(this.docPath, fileToDownLoad));
            string downloadPath = Path.GetTempPath();
            Directory.CreateDirectory(downloadPath);
            string downloadFilePath = Path.Combine(downloadPath, fileToDownLoad);
            this.DownloadJsonFile(getPackageUrl, downloadFilePath);
            string receivedContent = System.IO.File.ReadAllText(downloadFilePath);
            Assert.AreEqual(expectedContent, receivedContent);
        }

        /// <summary>
        /// Test to validate post health report URI's
        /// </summary>
        [TestMethod]
        public void TestValidatePostHealthReportUris()
        {
            this.StartHostNetAgent(this.logPath, this.docPath, this.configFile);
            JObject healthReportJObject = this.CreateHealthReport();
            using (var w = new WebClient())
            {
                w.Headers[HttpRequestHeader.ContentType] = "application/json";
                Console.WriteLine("Uplaoding {0} to url {1}", healthReportJObject, this.healthReportUrl);
                w.UploadString(this.healthReportUrl, healthReportJObject.ToString());
            }
        }

        /// <summary>
        /// Test to validate end to end workflow.
        /// </summary>
        [TestMethod]

        public void TestWorFlow()
        {
            this.StartHostNetAgent(this.logPath, this.docPath, this.configFile);
            var version = this.DownloadString(this.getVersionUrl);
            Console.WriteLine("Received version:{0}", version);

            var packageUris = this.DownloadSerializedJsonData<Dictionary<string, string>>(this.getPackageUrisUrl);
            string downloadPath = Path.Combine(Environment.CurrentDirectory, "temp");
            if (!Directory.Exists(downloadPath))
            {
                Directory.CreateDirectory(downloadPath);
            }

            foreach (KeyValuePair<string, string> packageKvp in packageUris)
            {
                Console.WriteLine("ResourceName:{0}, ResourceUri:{1}", packageKvp.Key, packageKvp.Value);
                if (string.IsNullOrEmpty(packageKvp.Value))
                {
                    var getPackageUrl = this.baseGetUrl + string.Format("&type={0}", packageKvp.Key);
                    Console.WriteLine("Downloading resource:{0}", packageKvp.Key);
                    this.DownloadJsonFile(getPackageUrl, Path.Combine(downloadPath, packageKvp.Key));
                }
            }
        }

        /// <summary>
        /// Helper to deserialize Json data.
        /// </summary>
        /// <typeparam name="T">Object type to deserialize</typeparam>
        /// <param name="url">URL to connect.</param>
        /// <returns>Deserialized object</returns>
        private T DownloadSerializedJsonData<T>(string url) where T : new()
        {
            using (var w = new WebClient())
            {
                string json_data;
                try
                {
                    json_data = w.DownloadString(url);
                }
                catch (Exception exception)
                {
                    Console.WriteLine(exception);
                    throw;
                }

                return !string.IsNullOrEmpty(json_data) ? JsonConvert.DeserializeObject<T>(json_data) : new T();
            }
        }

        /// <summary>
        /// Helper to deserialize Json string
        /// </summary>
        /// <param name="url">Url to connect.</param>
        /// <returns>Deserialized string</returns>
        private string DownloadString(string url)
        {
            using (var w = new WebClient())
            {
                string data;

                try
                {
                    data = w.DownloadString(url);
                }
                catch (Exception exception)
                {
                    Console.WriteLine(exception);
                    throw;
                }

                return data;
            }
        }

        /// <summary>
        /// Helper to download json file.
        /// </summary>
        /// <param name="url">Url to connect.</param>
        /// <param name="fileLocation">Location to download.</param>
        private void DownloadJsonFile(string url, string fileLocation)
        {
            using (var w = new WebClient())
            {
                try
                {
                    w.DownloadFile(url, fileLocation);
                }
                catch (Exception exception)
                {
                    Console.WriteLine(exception);
                    throw;
                }
            }
        }

        /// <summary>
        /// Helper to start net agent server
        /// </summary>
        /// <param name="logFolder">Log path</param>
        /// <param name="docFolder">Doc path</param>
        /// <param name="configFilePath">Location of the config file.</param>
        /// <param name="restart">Should restart</param>
        private void StartHostNetAgent(string logFolder, string docFolder, string configFilePath, bool restart = false)
        {
            Process[] processesByName = Process.GetProcessesByName(HostNetAgentProcessName);
            if (processesByName.Length > 0)
            {
                if (restart)
                {
                    StopHostNetAgent();
                }
                else
                {
                    return;
                }
            }

            Process process = new Process
            {
                StartInfo =
                {
                    FileName = Path.Combine(this.exeDir, HostNetAgentProcessName + @".exe"),
                    Arguments = string.Format("/LogPath {0} /PackagePath {1} /ConfigFile {2}", logFolder, docFolder, configFilePath),
                    WindowStyle = ProcessWindowStyle.Maximized
                }
            };
            process.Start();
        }
    }
}