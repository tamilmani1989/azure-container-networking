// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncFailureMessage.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class DncFailureMessage
    {
        public int ErrorCode
        {
            get; set;
        }

        public string Text
        {
            get; set;
        }

        public int HttpStatusCode
        {
            get; set;
        }

        public override string ToString()
        {
            return $"DNCFailureMessage: Code: {ErrorCode}, Text: {Text} " +
                $"HttpStatus: {HttpStatusCode}";
        }
    }
}
