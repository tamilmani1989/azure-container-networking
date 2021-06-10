using System;
using System.Collections.Generic;
using System.Linq;
using System.Threading.Tasks;

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class DncSettings
    {
        public int RetryCount
        {
            get; set;
        }

        public int RetryTimeoutInSeconds
        {
            get; set;
        }

        public int RetryIntervalInSeconds
        {
            get; set;
        }

        public string SubscriptionId
        {
            get; set;
        }

        public string TenantId
        {
            get; set;
        }

        public string ResourceGroupName
        {
            get; set;
        }

        public string VnetName
        {
            get; set;
        }

        public string SubnetName
        {
            get; set;
        }

        public DncSettings()
        {

        }
        public DncSettings(int retryCount, int retryInterval, int retryTimeout)
        {
            RetryCount = retryCount;
            RetryIntervalInSeconds = retryInterval;
            RetryTimeoutInSeconds = retryTimeout;
        }
    }
}
