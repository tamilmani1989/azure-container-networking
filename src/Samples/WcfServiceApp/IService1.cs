// --------------------------------------------------------------------------------------------------------------------
// <copyright file="IService1.cs" company="Microsoft">
//   Microsoft
// </copyright>
// <summary>
//   The Service1 interface.
//   NOTE: You can use the "Rename" command on the "Refactor" menu to change the interface name "IService1" in both code and config file together.
// </summary>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.EngSys.CoreXT.Samples
{
    using System.Runtime.Serialization;
    using System.ServiceModel;

    /// <summary>
    /// The Service1 interface.
    /// NOTE: You can use the "Rename" command on the "Refactor" menu to change the interface name "IService1" in both code and config file together.
    /// </summary>
    [ServiceContract]
    public interface IService1
    {
        /// <summary>
        /// The get data.
        /// </summary>
        /// <param name="value">
        /// The value.
        /// </param>
        /// <returns>
        /// The <see cref="string"/>.
        /// </returns>
        [OperationContract]
        string GetData(int value);

        /// <summary>
        /// The get data using data contract.
        /// </summary>
        /// <param name="composite">
        /// The composite.
        /// </param>
        /// <returns>
        /// The <see cref="CompositeType"/>.
        /// </returns>
        [OperationContract]
        CompositeType GetDataUsingDataContract(CompositeType composite);

        // TODO: Add your service operations here
    }

    /// <summary>
    /// The composite type.
    /// Use a data contract as illustrated in the sample below to add composite types to service operations.
    /// </summary>
    [DataContract]
    public class CompositeType
    {
        /// <summary>
        /// Initializes a new instance of the <see cref="CompositeType"/> class.
        /// </summary>
        public CompositeType()
        {
            this.StringValue = "Hello ";
            this.BoolValue = true;
        }

        /// <summary>
        /// Gets or sets a value indicating whether boolean value.
        /// </summary>
        [DataMember]
        public bool BoolValue { get; set; }

        /// <summary>
        /// Gets or sets the string value.
        /// </summary>
        [DataMember]
        public string StringValue { get; set; }
    }
}
