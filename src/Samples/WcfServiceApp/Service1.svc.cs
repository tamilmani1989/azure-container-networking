// --------------------------------------------------------------------------------------------------------------------
// <copyright file="Service1.svc.cs" company="Microsoft">
//   Microsoft
// </copyright>
// <summary>
//   The service 1.
//   NOTE: You can use the "Rename" command on the "Refactor" menu to change the class name "Service1" in code, svc and config file together.
// </summary>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.EngSys.CoreXT.Samples
{
    using System;

    /// <summary>
    /// The service 1.
    /// NOTE: You can use the "Rename" command on the "Refactor" menu to change the class name "Service1" in code, svc and config file together.
    /// </summary>
    public class Service1 : IService1
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
        public string GetData(int value)
        {
            return string.Format("You entered: {0}", value);
        }

        /// <summary>
        /// The get data using data contract.
        /// </summary>
        /// <param name="composite">
        /// The composite.
        /// </param>
        /// <returns>
        /// The <see cref="CompositeType"/>.
        /// </returns>
        /// <exception cref="ArgumentNullException">
        /// Composite argument.
        /// </exception>
        public CompositeType GetDataUsingDataContract(CompositeType composite)
        {
            if (composite == null)
            {
                throw new ArgumentNullException("composite");
            }

            if (composite.BoolValue)
            {
                composite.StringValue += "Suffix";
            }

            return composite;
        }
    }
}
