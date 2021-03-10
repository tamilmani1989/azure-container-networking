// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncResponseStatus.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class DncResponseStatus
    {
        public DncResponseStatus()
        {
        }

        public string State
        {
            get; set;
        }

        public string Status
        {
            get; set;
        }

        public DncFailureMessage FailureDetail
        {
            get; set;
        }

        public override string ToString()
        {
            return $"DncResponseStatus: State: {State}, Status: {Status} " +
                $"FailureDetail: {FailureDetail}";
        }
    }
}
