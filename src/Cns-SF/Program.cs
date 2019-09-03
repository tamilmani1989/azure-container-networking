using System;
using System.Collections.Generic;
using System.Fabric;
using System.Fabric.Query;
using System.Globalization;
using System.Linq;
using System.Runtime.InteropServices;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

namespace CnsWrapperService
{
    internal static class Program
    {
        private delegate bool ControlEventHandler(ConsoleCtrlEvent consoleEvent);

        private static ControlEventHandler _controlEventHandler;

        private static AutoResetEvent _shutdownSignalled = new AutoResetEvent(false);

        private static AutoResetEvent _canCloseInstance = new AutoResetEvent(false);

        private static object _wrapperSetup;

        /// <summary>
        /// Gets event to get notified from service that shutdown is requested.
        /// </summary>
        internal static AutoResetEvent ShutdownSignalled
        {
            get
            {
                return _shutdownSignalled;
            }
        }

        /// <summary>
        /// Gets event to get notified that service cleaned up and that we can close ourselves.
        /// </summary>
        internal static AutoResetEvent CanCloseInstance
        {
            get
            {
                return _canCloseInstance;
            }
        }

        internal static WrapperSetup WrapperSetup
        {
            get
            {
                return (WrapperSetup)_wrapperSetup;
            }
        }

        /// <summary>
        /// Defines the entry point of the application.
        /// </summary>
        /// <param name="args">Arguments to main function</param>
        public static void Main(string[] args)
        {
            string wrapperSetupType = typeof(WrapperSetup).FullName;
            string wrapperLoggerType = typeof(WrapperLogger).FullName;

            if (args.Length > 0)
            {
                int i = 0;

                while (i < args.Length)
                {
                    if (string.Compare(args[i], Constants.WrapperSetupNameCmdLineArgument, CultureInfo.InvariantCulture,
                            CompareOptions.IgnoreCase) == 0)
                    {
                        wrapperSetupType = args[i + 1];
                    }
                    else if (string.Compare(args[i], Constants.WrapperLoggerNameCmdLineArgument, CultureInfo.InvariantCulture,
                                 CompareOptions.IgnoreCase) == 0)
                    {
                        wrapperLoggerType = args[i + 1];
                    }
                    else
                    {
                        throw new NotImplementedException("Passed argument is not available in this version.");
                    }

                    i += 2;
                }
            }

            Type typeToCreate = Type.GetType(wrapperSetupType, true);

            FabricRuntime fabricRuntime = null;
            try
            {
                _wrapperSetup = Activator.CreateInstance(typeToCreate);

                // Start logging.
                //
                typeToCreate.GetMethod("Start").Invoke(_wrapperSetup, new object[] {wrapperLoggerType});

                // Register for console events (Ctrl+C/Ctrl+Break/shutdown) and wait for an event to happen.
                //
                _controlEventHandler = new ControlEventHandler(ConsoleShutdownHandler);
                SetConsoleCtrlHandler(_controlEventHandler, true);

                fabricRuntime = FabricRuntime.Create();

                // Register ServiceFactory with the runtime
                //
                fabricRuntime.RegisterServiceType(Constants.ServiceType, typeof(CnsWrapperService));

                while (!CanCloseInstance.WaitOne(WrapperConfiguration.MainThreadShutdownCheck))
                {
                }
            }
            catch (Exception e)
            {
                if (_wrapperSetup != null)
                {
                    // Dispose logging.
                    //
                    typeToCreate.GetMethod("ProcessError").Invoke(_wrapperSetup, new [] {e});
                }
            }
            finally
            {
                try
                {
                    if (_wrapperSetup != null)
                    {
                        // Dispose logging.
                        //
                        typeToCreate.GetMethod("Dispose").Invoke(_wrapperSetup, null);
                    }
                }
                finally
                {
                    if (fabricRuntime != null)
                    {
                        fabricRuntime.Dispose();
                    }

                    if (_controlEventHandler != null)
                    {
                        SetConsoleCtrlHandler(_controlEventHandler, false);
                    }
                }
            }
        }

        /// <summary>
        /// Handler for the Ctrl+C/Ctrl+Break/shutdown events.
        /// </summary>
        /// <param name="consoleEvent">
        /// The console event that occured
        /// </param>
        private static bool ConsoleShutdownHandler(ConsoleCtrlEvent consoleEvent)
        {
            // Signal shutdown to all services and wait for them to quit.
            //
            _shutdownSignalled.Set();
            CnsWrapperService.WaitForWrappedService(WrapperConfiguration.ShutdownThreadWaitTimeout);
            return true;
        }

        #region External Members

        [DllImport("kernel32.dll")]
        [return: MarshalAs(UnmanagedType.Bool)]
        private static extern bool SetConsoleCtrlHandler(ControlEventHandler e, [MarshalAs(UnmanagedType.Bool)] bool add);

        /// <summary>
        /// All possible events for GenerateConsoleCtrlEvent, from kernel32.dll
        /// </summary>
        public enum ConsoleCtrlEvent
        {
            CTRL_C = 0,
            CTRL_BREAK = 1,
            CTRL_CLOSE = 2,
            CTRL_LOGOFF = 5,
            CTRL_SHUTDOWN = 6
        }

        #endregion External Members
    }
}
