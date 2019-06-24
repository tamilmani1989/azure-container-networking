using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace CnsWrapperService
{
    /// <summary>
    /// Wrapper setup.
    /// </summary>
    class WrapperSetup : IDisposable
    {
        /// <summary>
        /// Logger.
        /// </summary>
        public IWrapperLogger m_logger;

        /// <summary>
        /// Start wrapper.
        /// </summary>
        /// <param name="wrapperLoggerType"></param>
        public void Start(string wrapperLoggerType)
        {
            Type typeToCreate = Type.GetType(wrapperLoggerType, true);
            m_logger = (IWrapperLogger)Activator.CreateInstance(typeToCreate);
        }

        /// <summary>
        /// Process error
        /// </summary>
        /// <param name="e"></param>
        public void ProcessError(Exception e)
        {
            m_logger.LogError("[Wrapper][Error] {0}", e);
        }

        /// <summary>
        /// Get wrapper logger.
        /// </summary>
        /// <returns></returns>
        public IWrapperLogger GetLogger()
        {
            return m_logger;
        }

        /// <summary>
        /// Dispose method.
        /// </summary>
        public void Dispose()
        {
            m_logger.Dispose();
        }
    }
}
