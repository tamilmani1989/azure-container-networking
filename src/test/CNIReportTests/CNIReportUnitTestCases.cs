//-----------------------------------------------------------------------
// <copyright file="CNIReportUnitTestCases.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
// 
// </summary>
// 
//-----------------------------------------------------------------------
namespace Microsoft.WindowsAzure.Networking.Aquarius.CNIReportTests
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
    /// Class definition CNIReportUnitTestCases
    /// </summary>
    [TestClass]
    public class CNIReportUnitTestCases
    {
        /// <summary>
        /// Name of Plugin.
        /// </summary>
        private static readonly string Name = "Name";
        
        /// <summary>
        /// Version of Plugin.
        /// </summary>
        private static readonly string Version = "Version";

        /// <summary>
        /// Error Message if plugin encounters any.
        /// </summary>
        private static readonly string ErrorMessage = "ErrorMessage";

        /// <summary>
        /// Context for the event.
        /// </summary>
        private static readonly string Context = "Context";

        /// <summary>
        /// SubContext for the event.
        /// </summary>
        private static readonly string SubContext = "SubContext";

        /// <summary>
        /// OrchestratorDetails Object.
        /// </summary>
        private static readonly string OrchestratorDetails = "OrchestratorDetails";

        /// <summary>
        /// Orchestrator Name.
        /// </summary>
        private static readonly string OrchestratorName = "OrchestratorName";

        /// <summary>
        /// Orchestrator Version.
        /// </summary>
        private static readonly string OrchestratorVersion = "OrchestratorVersion";

        /// <summary>
        /// OS Details Object.
        /// </summary>
        private static readonly string OSDetails = "OSDetails";

        /// <summary>
        /// OS Family.
        /// </summary>
        private static readonly string OSType = "OSType";

        /// <summary>
        /// OS Version.
        /// </summary>
        private static readonly string OSVersion = "OSVersion";

        /// <summary>
        /// OS Kernel Version.
        /// </summary>
        private static readonly string KernelVersion = "KernelVersion";

        /// <summary>
        /// OS Distribution.
        /// </summary>
        private static readonly string OSDistribution = "OSDistribution";

        /// <summary>
        /// OS Error Message.
        /// </summary>
        private static readonly string OSErrorMessage = "ErrorMessage";

        /// <summary>
        /// System Details Object.
        /// </summary>
        private static readonly string SystemDetails = "SystemDetails";

        /// <summary>
        /// Total Memory Space in MB.
        /// </summary>
        private static readonly string MemVMTotal = "MemVMTotal";

        /// <summary>
        /// Available Memory Space in MB.
        /// </summary>
        private static readonly string MemVMFree = "MemVMFree";

        /// <summary>
        /// Used Memory Space by Process in MB.
        /// </summary>
        private static readonly string MemUsedByProcess = "MemUsedByProcess";

        /// <summary>
        /// Total Disk Space in MB.
        /// </summary>
        private static readonly string DiskVMTotal = "DiskVMTotal";

        /// <summary>
        /// Available Disk Space in MB.
        /// </summary>
        private static readonly string DiskVMFree = "DiskVMFree";

        /// <summary>
        /// CPU Count.
        /// </summary>
        private static readonly string CPUCount = "CPUCount";

        /// <summary>
        /// System Error Message.
        /// </summary>
        private static readonly string SysErrorMessage = "ErrorMessage";

        /// <summary>
        /// Start Flag - to check how many VMs running this plugin.
        /// </summary>
        private static readonly string StartFlag = "StartFlag";

        /// <summary>
        /// Check how many VMs succeeded in running this plugin.
        /// </summary>
        private static readonly string CniSucceeded = "CniSucceeded";

        /// <summary>
        /// Interface Details Object.
        /// </summary>
        private static readonly string InterfaceDetails = "InterfaceDetails";

        /// <summary>
        /// Interface Type.
        /// </summary>
        private static readonly string InterfaceType = "InterfaceType";

        /// <summary>
        /// Interface Name.
        /// </summary>
        private static readonly string IfName = "Name";

        /// <summary>
        /// Subnet Address.
        /// </summary>
        private static readonly string Subnet = "Subnet";

        /// <summary>
        /// Primary CA.
        /// </summary>
        private static readonly string PrimaryCA = "PrimaryCA";

        /// <summary>
        /// MAC Address.
        /// </summary>
        private static readonly string MAC = "MAC";

        /// <summary>
        /// Secondary CA TotalCount.
        /// </summary>
        private static readonly string SecondaryCATotalCount = "SecondaryCATotalCount";

        /// <summary>
        /// Secondary CA used count.
        /// </summary>
        private static readonly string SecondaryCAUsedCount = "SecondaryCAUsedCount";

        /// <summary>
        /// Interface Error Message.
        /// </summary>
        private static readonly string IfErrorMessage = "ErrorMessage";

        /// <summary>
        /// HostNetAgent Process Name.
        /// </summary>
        private static readonly string HostNetAgentProcessName = "HostNetAgent";

        /// <summary>
        /// Location of executable.
        /// </summary>
        private readonly string exeDir;

        /// <summary>
        /// Base Post Url.
        /// </summary>
        private readonly string basePostUrl;

        /// <summary>
        /// Base Post Url.
        /// </summary>
        private readonly string cniReportUrl;

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
        /// Initializes a new instance of the <see cref="CNIReportUnitTestCases" /> class
        /// </summary>
        public CNIReportUnitTestCases()
        {
            this.exeDir = AppDomain.CurrentDomain.BaseDirectory;
            this.logPath = Path.Combine(this.exeDir, "Logs");
            this.docPath = Path.Combine(this.exeDir, "Documents");
            Console.WriteLine("exedir {0}", this.exeDir);
            this.basePostUrl = "http://localhost:8019/hostnetagent/container/1234";
            this.cniReportUrl = this.basePostUrl + "/cnireport";
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
        /// Helper to create CNI report object.
        /// </summary>
        /// <returns>Create CNI report object.</returns>
        public JObject CreateCNIReport()
        {
            JObject orchestratorDetailsValue = new JObject(
                new JProperty(OrchestratorName, "Kubernetes"),
                new JProperty(OrchestratorVersion, "1.8.0"));

            JObject operatingSystemDetailsValue = new JObject(
                new JProperty(OSType, "Linux"),
                new JProperty(OSVersion, "16.0.4"),
                new JProperty(KernelVersion, "4.4.11-azure"),
                new JProperty(OSDistribution, "Debian"),
                new JProperty(OSErrorMessage, string.Empty));

            JObject systemDetailsValue = new JObject(
                new JProperty(MemVMTotal, 45678),
                new JProperty(MemVMFree, 12345),
                new JProperty(MemUsedByProcess, 0),
                new JProperty(DiskVMTotal, 98756),
                new JProperty(DiskVMFree, 10756),
                new JProperty(CPUCount, 8),
                new JProperty(SysErrorMessage, string.Empty));

            JObject interfaceDetailsValue = new JObject(
                new JProperty(InterfaceType, "Primary"),
                new JProperty(Subnet, "172.0.1.0/24"),
                new JProperty(PrimaryCA, "172.0.1.4"),
                new JProperty(MAC, "ac:1d:d4:3e:e5:f1"),
                new JProperty(IfName, "eth0"),
                new JProperty(SecondaryCATotalCount, 0),
                new JProperty(SecondaryCAUsedCount, 0),
                new JProperty(IfErrorMessage, string.Empty));

            return new JObject(
                new JProperty(Name, "CNI"),
                new JProperty(Version, "v1.0.0"),
                new JProperty(ErrorMessage, string.Empty),
                new JProperty(Context, "AzureCNI"),
                new JProperty(SubContext, string.Empty),
                new JProperty(StartFlag, true),
                new JProperty(CniSucceeded, true),
                new JProperty(OrchestratorDetails, orchestratorDetailsValue),
                new JProperty(OSDetails, operatingSystemDetailsValue),
                new JProperty(SystemDetails, systemDetailsValue),
                new JProperty(InterfaceDetails, interfaceDetailsValue));
        }

        /// <summary>
        /// Test to validate post CNI report URI's
        /// </summary>
        [TestMethod]
        public void TestValidatePostCNIReportUris()
        {
            this.StartHostNetAgent(this.logPath, this.docPath, this.configFile);
            JObject cniReportJObject = this.CreateCNIReport();
            using (var w = new WebClient())
            {
                w.Headers[HttpRequestHeader.ContentType] = "application/json";
                Console.WriteLine("Uplaoding {0} to url {1}", cniReportJObject, this.cniReportUrl);
                w.UploadString(this.cniReportUrl, cniReportJObject.ToString());
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