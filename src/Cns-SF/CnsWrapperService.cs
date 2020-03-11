//-----------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation.  All rights reserved.
//-----------------------------------------------------------------------------

using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Fabric;
using System.Linq;
using System.Reflection;
using System.Runtime.ExceptionServices;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

namespace CnsWrapperService
{
    class CnsWrapperService : WrapperStatelessService
    {
        // By having this as static, we allow only one thread to run.
        // This might be OK now, but if SF is interweaving InstanceOpen/Close calls, we might be misbehaving.
        //
        private static Thread _mainThread = null;

        /// <summary>
        /// Guarding access to mainThread
        /// </summary>
        private static Object _mainThreadLock = new Object();

        /// <summary>
        /// Event to wait for instance to be closed
        /// </summary>
        private AutoResetEvent _waitInstanceClose = new AutoResetEvent(false);

        /// <summary>
        /// Signals that Cns thread is done executing
        /// </summary>
        private AutoResetEvent _azureCnsClosed = new AutoResetEvent(false);

        /// <summary>
        /// Azure CNS process
        /// </summary>
        private Process _azureCnsProcess;

        /// <summary>
        /// Azure CNS port
        /// </summary>
        private int _azureCnsPort;

        /// <summary>
        /// Additional cmdline azure cns arguments
        /// </summary>
        private string _additionalCmdLineArguments;

        protected override void InstanceAbort()
        {
            // Signal to close instance and wait for it to be closed.
            //
            _waitInstanceClose.Set();

            WaitForWrappedService(WrapperConfiguration.ThreadWaitTimeout);
        }

        protected override void InstanceClose()
        {
            InstanceAbort();
        }

        protected override string InstanceOpen()
        {
            lock (_mainThreadLock)
            {
                if (_mainThread != null)
                {
                    throw new InvalidOperationException("Thread already exists. Try again");
                }

                string endpoint = "AzureCnsEndpoint";
                if (WrapperConfiguration.EnablePortOverride)
                {
                    endpoint = "AzureCnsEndpointOverride";
                }

                _azureCnsPort = FabricRuntime.GetActivationContext().GetEndpoint(endpoint).Port;
                _additionalCmdLineArguments = string.Empty;

                if (WrapperConfiguration.SetupHnsAtStartup)
                {
                    _additionalCmdLineArguments += string.Format(" -defaultextnetworktype {0}", WrapperConfiguration.DefaultExtNetworkType);
                }

                // Note : Bool parameter values are passed as follows : <bool-parameter>=<bool-parameter value> 
                _additionalCmdLineArguments += string.Format(
                    " -telemetry={0} -httpcontimeout {1} -httprespheadertimeout {2} -startcnm={3}",
                    WrapperConfiguration.EnableTelemetry,
                    WrapperConfiguration.HttpConnectionTimeoutInSeconds,
                    WrapperConfiguration.HttpResponseHeaderTimeoutInSeconds,
                    WrapperConfiguration.StartCnm);

                string storeFilePath = WrapperConfiguration.StoreFilePath;
                if (!string.IsNullOrEmpty(storeFilePath))
                {
                    _additionalCmdLineArguments += string.Format(" -storefilepath {0}", storeFilePath);
                }

                // Creates cns_config.json file from settings specified in package XML files
                TelemetrySettingsService.CreateCNSConfigFile();

                _additionalCmdLineArguments.Trim();

                _mainThread = new Thread(this.Run);

                // Make it a background thread so we terminate even if it is hanging around.
                //
                _mainThread.IsBackground = true;
                _mainThread.Start();
            }

            return string.Format("{0},cns:{1}", Environment.MachineName, _azureCnsPort);
        }

        /// <summary>
        /// Trace close begin.
        /// </summary>
        protected override void TraceCloseBegin()
        {
            Program.WrapperSetup.GetLogger().LogInfo("[Close] Begin");
        }

        /// <summary>
        /// Trace close end.
        /// </summary>
        protected override void TraceCloseEnd()
        {
            Program.WrapperSetup.GetLogger().LogInfo("[Close] End");
        }

        /// <summary>
        /// Trace close failure.
        /// </summary>
        /// <param name="exception"></param>
        protected override void TraceCloseFailed(Exception exception)
        {
            Program.WrapperSetup.GetLogger().LogError("[Close] Failed; {0} {1}", exception.Message, exception.StackTrace);
        }

        /// <summary>
        /// Trace abort begin.
        /// </summary>
        protected override void TraceAbortBegin()
        {
            Program.WrapperSetup.GetLogger().LogInfo("[Abort] Begin");
        }

        /// <summary>
        /// Trace abort end.
        /// </summary>
        protected override void TraceAbortEnd()
        {
            Program.WrapperSetup.GetLogger().LogInfo("[Abort] End");
        }

        /// <summary>
        /// Trace abort failure.
        /// </summary>
        /// <param name="exception"></param>
        protected override void TraceAbortFailed(Exception exception)
        {
            Program.WrapperSetup.GetLogger().LogError("[Abort] Failed; {0} {1}", exception.Message, exception.StackTrace);
        }

        /// <summary>
        /// Trace open begin.
        /// </summary>
        protected override void TraceOpenBegin()
        {
            Program.WrapperSetup.GetLogger().LogInfo("[Open] Begin");
        }

        /// <summary>
        /// Trace open end.
        /// </summary>
        protected override void TraceOpenEnd()
        {
            Program.WrapperSetup.GetLogger().LogInfo("[Open] End");
        }

        /// <summary>
        /// Trace open failure.
        /// </summary>
        protected override void TraceOpenFailed(Exception exception)
        {
            Program.WrapperSetup.GetLogger().LogError("[Open] Failed; {0} {1}", exception.Message, exception.StackTrace);
        }

        /// <summary>
        /// Instance main implementation.
        /// </summary>
        private void Run()
        {
            try
            {
                Program.WrapperSetup.GetLogger().LogInfo("Started main thread on {0}", FabricRuntime.GetActivationContext().ServiceListenAddress);

                Thread t = new Thread(AzureCnsWrapper);
                t.IsBackground = true;
                t.Start();

                Program.WrapperSetup.GetLogger().LogInfo("Waiting in the main thread loop");

                WaitHandle[] waitingHandles = { _azureCnsClosed, _waitInstanceClose, Program.ShutdownSignalled };

                int signalIndex = -1;

                while (true)
                {
                    signalIndex = WaitHandle.WaitAny(waitingHandles, WrapperConfiguration.MainLoopTimeout);
                    if (signalIndex != WaitHandle.WaitTimeout)
                    {
                        break;
                    }
                }

                Program.WrapperSetup.GetLogger().LogInfo("Out of the main thread loop");

                // Notify that process can be closed.
                //
                Program.CanCloseInstance.Set();

                // Remove-ServiceFabricService calls CloseAsync on the service and after CloseAsync is finished it sends Ctrl+C to the process. 
                // Ctrl+C has to be handled successfully in order to finish successfully Remove-ServiceFabricService.
                // When azure-cns.exe exited or shutdown is sent to the main process we are setting that main process should shutdown.
                // As explained, CloseAsync (_waitInstanceClose in this logic) will be followed with Ctrl+C (ShutdownSignalled.Set()).
                // 
                if (signalIndex == 0 || signalIndex == 2)
                {
                    Program.ShutdownSignalled.Set();
                }
            }
            catch (Exception e)
            {
                // By convention, all error coming outside of cmdlet are treated as severe ones.
                // Errors happening inside Cns are "business logic" errors and treated as non-severe ones.
                //
                ReportHealthProperty(e, isSevere: true);

                throw;
            }
        }

        /// <summary>
        /// Waits for wrapped service to be closed
        /// </summary>
        /// <param name="timeout">Waiting timeout</param>
        public static void WaitForWrappedService(TimeSpan timeout)
        {
            lock (_mainThreadLock)
            {
                // Since we have only one active service in mainThread, we just wait for that one.
                //
                if (_mainThread != null)
                {
                    _mainThread.Join(timeout);
                    _mainThread = null;
                }
            }
        }

        /// <summary>
        /// Main thread that starts azure-cns.exe and blocks until it is done or throws exception.
        /// </summary>
        private void AzureCnsWrapper()
        {
            // Reset health property of any previous errors
            //
            ReportHealthProperty();

            try
            {
                // Invoke process record now that we have cmdlet
                //
                try
                {
                    Program.WrapperSetup.GetLogger().LogInfo("Invoking azure-cns");

                    this._azureCnsProcess = new Process();
                    this._azureCnsProcess.StartInfo.CreateNoWindow = false;
                    this._azureCnsProcess.StartInfo.UseShellExecute = false;
                    this._azureCnsProcess.StartInfo.RedirectStandardOutput = true;
                    this._azureCnsProcess.StartInfo.FileName = "azure-cns.exe";
                    this._azureCnsProcess.StartInfo.Arguments = string.Format("-c tcp://{0}:{1} -o {2} {3}", "0.0.0.0",
                        _azureCnsPort, FabricRuntime.GetActivationContext().LogDirectory, _additionalCmdLineArguments);
                    this._azureCnsProcess.OutputDataReceived += new DataReceivedEventHandler((sender, e) =>
                    {
                        if (!string.IsNullOrEmpty(e.Data))
                        {
                            Program.WrapperSetup.GetLogger().LogInfo(e.Data);
                        }
                    });

                    this._azureCnsProcess.Start();

                    while (!this._azureCnsProcess.WaitForExit(1000))
                    {
                        // wait for exit to happen.
                        //
                    }
                }
                catch (TargetInvocationException e)
                {
                    Program.WrapperSetup.GetLogger().LogError(e.InnerException.StackTrace);
                    ExceptionDispatchInfo.Capture(e.InnerException).Throw();
                }
            }
            catch (Exception e)
            {
                Program.WrapperSetup.GetLogger().LogError(e.StackTrace);
            }
            finally
            {
                // Run garbage collector to reclaim the memory azure-cns consumed
                //
                GC.Collect();

                // Signal that this thread is done
                //
                _azureCnsClosed.Set();
            }
        }
    }
}
