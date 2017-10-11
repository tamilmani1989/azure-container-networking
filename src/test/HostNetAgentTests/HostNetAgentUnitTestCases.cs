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
    using VisualStudio.TestTools.UnitTesting;

    /// <summary>
    /// Class definition for HostNetAgentUnitTestCases
    /// </summary>
    [TestClass]
    public class HostNetAgentUnitTestCases
    {
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
        /// Base Url.
        /// </summary>
        private readonly string baseUrl;

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
        /// Initializes a new instance of the <see cref="HostNetAgentUnitTestCases" /> class
        /// </summary>
        public HostNetAgentUnitTestCases()
        {
            this.exeDir = AppDomain.CurrentDomain.BaseDirectory;
            this.versionFile = Path.Combine(this.exeDir, "Documents", "version.txt");
            this.baseUrl = "http://localhost:8019/hostnetagent?container=1234";
            this.getVersionUrl = this.baseUrl + "&type=getpackageversion";
            this.getPackageUrisUrl = this.baseUrl + "&type=getpackageuris";
            this.logPath = Path.Combine(this.exeDir, "Logs");
            this.docPath = Path.Combine(this.exeDir, "Documents");
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
        /// Test to validate version
        /// </summary>
        [TestMethod]
        public void TestValidateVersion()
        {
            this.StartHostNetAgent(this.logPath, this.docPath);
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
            this.StartHostNetAgent(this.logPath, this.docPath);
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
            this.StartHostNetAgent(this.logPath, this.docPath);
            var packageUris = this.DownloadSerializedJsonData<Dictionary<string, string>>(this.getPackageUrisUrl);
            string fileToDownLoad = packageUris.First().Key;
            var getPackageUrl = this.baseUrl + string.Format("&type={0}", packageUris.First().Key);
            string expectedContent = System.IO.File.ReadAllText(Path.Combine(this.docPath, fileToDownLoad));
            string downloadPath = Path.GetTempPath();
            Directory.CreateDirectory(downloadPath);
            string downloadFilePath = Path.Combine(downloadPath, fileToDownLoad);
            this.DownloadJsonFile(getPackageUrl, downloadFilePath);
            string receivedContent = System.IO.File.ReadAllText(downloadFilePath);
            Assert.AreEqual(expectedContent, receivedContent);
        }

        /// <summary>
        /// Test to validate end to end workflow.
        /// </summary>
        [TestMethod]

        public void TestWorFlow()
        {
            this.StartHostNetAgent(this.logPath, this.docPath);
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
                    var getPackageUrl = this.baseUrl + string.Format("&type={0}", packageKvp.Key);
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
        /// <param name="restart">Should restart</param>
        private void StartHostNetAgent(string logFolder, string docFolder, bool restart = false)
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
                    Arguments = string.Format("{0} {1}", logFolder, docFolder),
                    WindowStyle = ProcessWindowStyle.Maximized
                }
            };
            process.Start();
        }
    }
}