//-----------------------------------------------------------------------
// <copyright file="MultiTenancyInfo.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// 
//-----------------------------------------------------------------------

using Newtonsoft.Json;

namespace Cns.Data
{
    /// <summary>
    /// Defines the MultiTenancyInfo class.
    /// </summary>
    public class MultiTenancyInfo
    {
        /// <summary>
        /// Encapuslation type.
        /// </summary>
        [JsonProperty("EncapType")]
        public string EncapType { get; private set; }

        /// <summary>
        /// This can be vlanid, vxlanid, gre-key etc. (depends on EnacapType).
        /// </summary>
        [JsonProperty("ID")]
        public int ID { get; private set; }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="encapType"></param>
        /// <param name="id"></param>
        public MultiTenancyInfo(string encapType, int id)
        {
            this.EncapType = encapType;
            this.ID = id;
        }

        /// <summary>
        /// ToString implementation
        /// </summary>
        /// <returns>String representation of the object.</returns>
        public override string ToString()
        {
            return $"[EncapType{this.EncapType}, ID:{this.ID}]";
        }
    }
}
