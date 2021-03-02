// --------------------------------------------------------------------------------------------------------------------
// <copyright file="DncFailureMessage.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.DncRP
{
    public class DncFailureMessage
    {
        public int Code
        {
            get; set;
        }

        public string Text
        {
            get; set;
        }

        public int HttpStatus
        {
            get; set;
        }

        public override string ToString()
        {
            return $"DNCFailureMessage: Code: {Code}, Text: {Text} " +
                $"HttpStatus: {HttpStatus}";
        }
    }
}
