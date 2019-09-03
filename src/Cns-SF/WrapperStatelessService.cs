using System;
using System.Collections.Generic;
using System.Fabric;
using System.Fabric.Health;
using System.Globalization;
using System.Linq;
using System.Text;
using System.Threading;
using System.Threading.Tasks;

namespace CnsWrapperService
{
    public abstract class WrapperStatelessService : IStatelessServiceInstance
    {
        /// <summary>
        /// Service name.
        /// </summary>
        protected Uri m_serviceName;

        /// <summary>
        /// Service type name.
        /// </summary>
        protected string m_serviceTypeName;

        /// <summary>
        /// Application name.
        /// </summary>
        protected string m_applicationName;

        /// <summary>
        /// Code package name.
        /// </summary>
        protected string m_codePackageName;

        /// <summary>
        /// Partition Id.
        /// </summary>
        protected Guid m_partitionId;

        public WrapperStatelessService()
        {
            m_serviceTypeName = GetType().Name;
        }

        /// <summary>
        /// Init.
        /// </summary>
        /// <param name="initializationParameters"></param>
        public void Initialize(StatelessServiceInitializationParameters initializationParameters)
        {
            if (initializationParameters != null)
            {
                m_serviceName = initializationParameters.ServiceName;
                m_applicationName = initializationParameters.CodePackageActivationContext.ApplicationName;
                m_codePackageName = initializationParameters.CodePackageActivationContext.CodePackageName;
                m_serviceTypeName = initializationParameters.ServiceTypeName;
                m_partitionId = initializationParameters.PartitionId;
            }
        }

        public Task<string> OpenAsync(IStatelessServicePartition partition, CancellationToken cancellationToken)
        {
            TraceOpenBegin();

            return Task.Factory.StartNew<string>(() =>
                {
                    try
                    {
                        string serviceAddress = InstanceOpen(partition);

                        TraceOpenEnd();

                        return serviceAddress;
                    }
                    catch (Exception e)
                    {
                        TraceOpenFailed(e);

                        throw;
                    }
                }
            );
        }

        public Task CloseAsync(CancellationToken cancellationToken)
        {
            TraceCloseBegin();

            return Task.Factory.StartNew(() =>
                {
                    try
                    {
                        InstanceClose();

                        TraceCloseEnd();
                    }
                    catch (Exception e)
                    {
                        TraceCloseFailed(e);

                        throw;
                    }
                }
            );
        }

        public void Abort()
        {
            TraceAbortBegin();

            try
            {
                InstanceAbort();

                TraceAbortEnd();
            }
            catch (Exception e)
            {
                TraceAbortFailed(e);

                throw;
            }
        }


        /// <summary>
        /// Open partition.
        /// </summary>
        /// <param name="partition"></param>
        /// <returns></returns>
        protected virtual string InstanceOpen(IStatelessServicePartition partition)
        {
            return this.InstanceOpen();
        }

        /// <summary>
        /// Report SF health property
        /// </summary>
        /// <param name="e">Exception</param>
        /// <param name="isSevere">Severe error or warning</param>
        protected void ReportHealthProperty(Exception e = null, bool isSevere = false)
        {
            string errorDetails = e != null ? e.Message : string.Empty;

            // Determine health status. If there is no exception, it is OK. If there is exception,
            // we set health to either warning or error, depending how severe error is.
            //
            HealthState healthState = HealthState.Ok;

            if (e != null)
            {
                healthState = HealthState.Warning;

                if (isSevere)
                {
                    healthState = HealthState.Error;
                }
            }

            var source = string.Format(
                CultureInfo.InvariantCulture,
                "{0}, {1}, {2}",
                FabricRuntime.GetNodeContext().NodeName,
                m_applicationName,
                m_codePackageName);

            HealthInformation healthInfo = new HealthInformation(source, "WrapperService:Exception", healthState);
            healthInfo.Description = errorDetails;
            healthInfo.TimeToLive = TimeSpan.FromHours(1);
            healthInfo.RemoveWhenExpired = true;

            var report = new ServiceHealthReport(m_serviceName, healthInfo);

            var fabricClient = new FabricClient();
            fabricClient.HealthManager.ReportHealth(report);
        }

        /// <summary>
        /// Trace close begin.
        /// </summary>
        protected virtual void TraceCloseBegin()
        {
        }

        /// <summary>
        /// Trace close end.
        /// </summary>
        protected virtual void TraceCloseEnd()
        {
        }

        /// <summary>
        /// Trace close failure.
        /// </summary>
        /// <param name="exception"></param>
        protected virtual void TraceCloseFailed(Exception exception)
        {
        }

        /// <summary>
        /// Trace abort begin.
        /// </summary>
        protected virtual void TraceAbortBegin()
        {
        }

        /// <summary>
        /// Trace abort end.
        /// </summary>
        protected virtual void TraceAbortEnd()
        {
        }

        /// <summary>
        /// Trace abort failure.
        /// </summary>
        /// <param name="exception"></param>
        protected virtual void TraceAbortFailed(Exception exception)
        {
        }

        /// <summary>
        /// Trace open begin.
        /// </summary>
        protected virtual void TraceOpenBegin()
        {
        }

        /// <summary>
        /// Trace open end.
        /// </summary>
        protected virtual void TraceOpenEnd()
        {
        }

        /// <summary>
        /// Trace open failure.
        /// </summary>
        protected virtual void TraceOpenFailed(Exception exception)
        {
        }

        /// <summary>
        /// Instance open.
        /// </summary>
        /// <returns></returns>
        protected abstract string InstanceOpen();

        /// <summary>
        /// Instance close.
        /// </summary>
        protected abstract void InstanceClose();

        /// <summary>
        /// Instance abort.
        /// </summary>
        protected abstract void InstanceAbort();
    }
}
