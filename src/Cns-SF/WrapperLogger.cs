using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace CnsWrapperService
{
    /// <summary>
    /// Interface for logger
    /// </summary>
    public interface IWrapperLogger : IDisposable
    {
        /// <summary>
        /// Start logger.
        /// </summary>
        void Start();

        /// <summary>
        /// Info logging
        /// </summary>
        /// <param name="line">Line to log</param>
        /// <param name="parameters">Additional parameters</param>
        void LogInfo(string line, params object[] parameters);

        /// <summary>
        /// Warning logging
        /// </summary>
        /// <param name="line">Line to log</param>
        /// <param name="parameters">Additional parameters</param>
        void LogWarning(string line, params object[] parameters);

        /// <summary>
        /// Error logging
        /// </summary>
        /// <param name="line">Line to log</param>
        /// <param name="parameters">Additional parameters</param>
        void LogError(string line, params object[] parameters);
    }

    /// <summary>
    /// Basic Logger
    /// </summary>
    class WrapperLogger : IWrapperLogger
    {
        /// <summary>
        /// Guarding access to mainThread
        /// </summary>
        private static Object _fileAccessLock = new Object();

        /// <summary>
        /// Log file path.
        /// </summary>
        private string LogFilePath { get; set; }

        /// <summary>
        /// Default constructor.
        /// </summary>
        public WrapperLogger()
        {
            LogFilePath = Constants.LogPath;
        }

        /// <summary>
        /// Constructor with parameter.
        /// </summary>
        /// <param name="filePath"></param>
        public WrapperLogger(string filePath)
        {
            LogFilePath = filePath;
        }

        /// <summary>
        /// Start logging.
        /// </summary>
        public void Start()
        {
            // For file logger no need to start anything.
        }

        /// <summary>
        /// Log info.
        /// </summary>
        /// <param name="line"></param>
        /// <param name="parameters"></param>
        public void LogInfo(string line, params object[] parameters)
        {
            var format = string.Format("[INFO] {0}", line);
            Log(format, parameters);
        }

        /// <summary>
        /// Log warning.
        /// </summary>
        /// <param name="line"></param>
        /// <param name="parameters"></param>
        public void LogWarning(string line, params object[] parameters)
        {
            var format = string.Format("[WARNING] {0}", line);
            Log(format, parameters);
        }

        /// <summary>
        /// Log error.
        /// </summary>
        /// <param name="line"></param>
        /// <param name="parameters"></param>
        public void LogError(string line, params object[] parameters)
        {
            var format = string.Format("[ERROR] {0}", line);
            Log(format, parameters);
        }

        /// <summary>
        /// Dispose.
        /// </summary>
        public void Dispose()
        {
        }

        /// <summary>
        /// Log helper method.
        /// </summary>
        /// <param name="format"></param>
        /// <param name="parameters"></param>
        private void Log(string format, params object[] parameters)
        {
            lock (_fileAccessLock)
            {
                File.AppendAllText(LogFilePath, string.Format(format + "\n", parameters));
            }
        }
    }
}
