using System.Threading;
using System.Threading.Tasks;
using Microsoft.WindowsAzure.Networking.Rnm.Utilities;
using Microsoft.WindowsAzure.Networking.Rnm.WinFabric;

namespace Microsoft.WindowsAzure.Networking.Rnm.VnetManager.UnitTests
{
    using System;
    using System.IO;
    using System.Net;
    using Microsoft.Cis.Fabric.Controller.InMemoryObjectStore;
    using Microsoft.Windows.Azure.Fabric.Cfx;
    using Microsoft.WindowsAzure.Networking.Rnm.Contracts.Services;
    using Microsoft.WindowsAzure.Networking.Rnm.InfrastructureServices;
    using Microsoft.WindowsAzure.Networking.Rnm.InternalContracts.Services;
    using Microsoft.WindowsAzure.Networking.Rnm.RoutingService;
    using Microsoft.WindowsAzure.Networking.Rnm.Tracing;
    using WEX.TestExecution.Markup;
    using VNM = Microsoft.WindowsAzure.Networking.Rnm.VnetManager;

    /// <summary>
    /// Unit test class.
    /// </summary>
    [TestClass]
    public partial class LogicalNetworkManagerUnitTest
    {
        private VNM.VirtualNetworkManager virtualNetworkManager;

        private IVirtualNetworkManager vnm;

        private IVnetAllocationUntransacted allocationManagement;

        /// <summary>
        /// IMOS repository of VNAM.
        /// </summary>
        internal IRepository Repository { get; private set; }

        const string rnmPartitonId = "RnmUnitTests";

        /// <summary>
        /// Constructor class is called during initialization to set up test environment.
        /// </summary>
        public LogicalNetworkManagerUnitTest()
        {
            LogicalNetworkManagerUnitTest.SetupIMOS();
            this.InitializeVnm();
        }
     
        /// <summary>
        /// Helper function to initialize logger.
        /// </summary>
        private void InitializeRnmLogger()
        {
            const string logFileName = "VNetManager.log";
            bool attachTraceUploader = false;
            bool uploadTracesToCloud = false;
            int autoFlushInterval = 5000;

            // Initialize the VnetManager tracer
            CfxTracer.Initialize(
                CfxRuntime.TraceSource.TargetDirectory,
                logFileName,
                CfxRuntime.TraceSource.MaxFileSize,
                uploadTracesToCloud,
                autoFlushInterval,
                attachTraceUploader);
        }

        /// <summary>
        /// Cleanup RSL and IMOS files from previous runs.
        /// </summary>
        private static void CleaupData()
        {
            string currDir = Path.Combine(Environment.CurrentDirectory, "repository");
            if (Directory.Exists(currDir))
            {
                Directory.Delete(currDir, true);
            }

            currDir = Path.Combine(Environment.CurrentDirectory, "rsldata");
            if (Directory.Exists(currDir))
            {
                Directory.Delete(currDir, true);
            }

            currDir = Path.Combine(Environment.CurrentDirectory, "rsl");
            if (Directory.Exists(currDir))
            {
                Directory.Delete(currDir, true);
            }

            currDir = Path.Combine(Environment.CurrentDirectory, "rsldebuglogs");
            if (Directory.Exists(currDir))
            {
                Directory.Delete(currDir, true);
            }

            currDir = Path.Combine(Environment.CurrentDirectory, "WinFabDataStore");
            if (Directory.Exists(currDir))
            {
                Directory.Delete(currDir, true);
            }
        }


        /// <summary>
        /// This function set is called by the test framework to set up unit test environment.
        /// </summary>
        private static void SetupIMOS()
        {
            CleaupData();

            InstallArtifacts.SetupWinFabric();
            Thread.Sleep(120000);

            //install package to deploy services
            DeployPackage();
        }

        private static void DeployPackage()
        {
            Helper.RunCommand("powershell", "Import-Module ServiceFabric");
            Helper.RunCommand("powershell", "Connect-ServiceFabricCluster localhost:19000");
            Helper.RunCommand("powershell",
                "Copy-ServiceFabricApplicationPackage -Verbose .\\RDPackage\\RnmWindowsFabricPackage  -ImageStoreConnectionString \"fabric:ImageStore\"");
            Helper.RunCommand("powershell",
                "Register-ServiceFabricApplicationType -ApplicationPathInImageStore \"RnmWindowsFabricPackage\"");
            Helper.RunCommand("powershell",
                "New-ServiceFabricApplication -ApplicationName fabric:/rnm -ApplicationTypeName Rnm -ApplicationTypeVersion 18.0");
            Helper.RunCommand("powershell",
                "New-ServiceFabricService -ApplicationName fabric:/rnm -ServiceName fabric:/rnm/svimanager -ServiceTypeName SVIManager -Stateful -PartitionSchemeSingleton -TargetReplicaSetSize 7  -MinReplicaSetSize 7 -HasPersistedState");

            Thread.Sleep(120000);
        }

        /// <summary>
        /// Helper function to convert an IPv4 address string to a host order UInt32.
        /// </summary>
        /// <param name="ipAddressString">Supply the IPv4 address string.</param>
        /// <returns>IPv4 address as UInt32 in host order.</returns>
        public static UInt32 ConvertIPAddressToUint32(string ipAddressString)
        {
            return ConvertIPAddressToUint32(IPAddress.Parse(ipAddressString));
        }

        /// <summary>
        /// Helper function to convert an IPv4 address to a host order UInt32.
        /// </summary>
        /// <param name="ipAddress">Supply the IPv4 address.</param>
        /// <returns>IPv4 address as UInt32 in host order.</returns>
        public static UInt32 ConvertIPAddressToUint32(IPAddress ipAddress)
        {
            Int32 addressInInt32 = BitConverter.ToInt32(ipAddress.GetAddressBytes(), 0);
            return (UInt32)(IPAddress.NetworkToHostOrder(addressInInt32));
        }

        /// <summary>
        /// Helper function to convert an UInt32 (host order) to IPAddress.
        /// </summary>
        /// <param name="ipAddressInUInt32">Supply the IP address in UInt32.</param>
        /// <returns>Converted IPAddress.</returns>
        public static IPAddress ConvertUInt32ToIPAddress(UInt32 ipAddressInUInt32)
        {
            return new IPAddress(
                BitConverter.GetBytes(IPAddress.HostToNetworkOrder((Int32)ipAddressInUInt32)));
        }

        /// <summary>
        /// Helper function that get the string representation of an IP address in UInt32 
        /// (host order).
        /// </summary>
        /// <param name="ipAddress">Supply the IPv4 address in UInt32.</param>
        /// <returns>String representation of the IPv4 address.</returns>
        public static string GetIPAddressString(UInt32 ipAddress)
        {
            return ConvertUInt32ToIPAddress(ipAddress).ToString();
        }

        public static Guid GenerateMacGuid()
        {
            int index;
            int start;
            int end;
            Guid nextMACGuid = Guid.NewGuid();           

            byte[] nextMacGuidBytes = nextMACGuid.ToByteArray();
            if (BitConverter.IsLittleEndian)            
            {
                start = 0;
                end = nextMacGuidBytes.Length - 6;
            }
            else
            {
                start = 6;
                end = nextMacGuidBytes.Length;
            }
            
            for (index = start; index < end; index++)
            {
                nextMacGuidBytes[index] = 0;
            }

            return new Guid(nextMacGuidBytes);
        }

        public static Guid GetNextGuid(Guid guid)
        {
            
            byte[] guidBytes = guid.ToByteArray();
            
            int iter;
            
            if (BitConverter.IsLittleEndian)
            {
                Array.Reverse(guidBytes);                
            }
            for (iter = 0; iter < 6; iter++)
            {
                guidBytes[iter]++;
                if (guidBytes[iter] != 0)
                {
                    break;
                }
            }
            if (BitConverter.IsLittleEndian)
            {
                Array.Reverse(guidBytes);
            }
            return new Guid(guidBytes);
        }
        
        /// <summary>
        /// Initialize LNM
        /// </summary>
        private void InitializeVnm()
        {
            this.InitializeRnmLogger();
            InternalInterfaceAccessor internalAccessor;            
            InfrastructureServicesManager infraServiceManager = InfrastructureServicesManager.Create();
            ServiceManager.ServiceManager serviceManager = ServiceManager.ServiceManager.Create();
            this.virtualNetworkManager = VirtualNetworkManager.Create();
            
            internalAccessor = new InternalInterfaceAccessor(
                serviceManager,
                this.virtualNetworkManager.GetVnetAllocationUntransactedInterface(),
                InfrastructureServicesManager.GetInfrastructureServiceUntransacted(),
                null,
                RepositoriesManager.RepositoriesManager.GetRepositoriesManagement(),
                null);
            
            serviceManager.Initialize(internalAccessor);
            this.virtualNetworkManager.Initialize(internalAccessor);
            this.vnm = this.virtualNetworkManager.GetVirtualNetworkManagerInterface();
            this.allocationManagement =
                            this.virtualNetworkManager.GetVnetAllocationUntransactedInterface();

            // Notify the LNM that it has becomes the primary. This function will not be called
            // after SetupUnitTests is called, which blocks until the current replica becomes
            // RSL primary.
            this.virtualNetworkManager.OnBecomePrimary();

            this.Repository = VirtualNetworkManager.SchemaManager.GetRepositoryByName(
                            VNAM.SchemaHelper.RepositoryName);
        }

        /// <summary>
        /// Helper function to start a read write session of VNAM repository.
        /// </summary>
        internal void BeginReadWriteSession()
        {
            VirtualNetworkManager.SchemaManager.BeginReadWriteSession(this.Repository,
                RepositoriesManager.RepositoriesManager.GetRepositoriesManagement().GetSmRepository());
        }

        /// <summary>
        /// Helper function to commit a read write session of VNAM repository.
        /// </summary>
        internal void CommitReadWriteSession()
        {
            Task t = VirtualNetworkManager.SchemaManager.CommitReadWriteSessionAsync();
            t.Wait();
        }

        /// <summary>
        /// Helper function to rollback a read write session of VNAM repository.
        /// </summary>
        internal void RollbackReadWriteSession()
        {
            VirtualNetworkManager.SchemaManager.RollbackReadWriteSession();
        }

        /// <summary>
        /// Helper function to start a read session of VNAM repository.
        /// </summary>
        internal void BeginReadSession()
        {
            VirtualNetworkManager.SchemaManager.BeginReadSession();
        }

        /// <summary>
        /// Helper function to end a read session of VNAM repository.
        /// </summary>
        internal void EndReadSession()
        {
            VirtualNetworkManager.SchemaManager.EndReadSession();
        }

    }
}
