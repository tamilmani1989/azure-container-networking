//-----------------------------------------------------------------------
// <copyright file="IPSubnet.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// <summary>
//   Implement IPSubnet class.
// </summary>
//-----------------------------------------------------------------------
namespace NetworkManager
{
    using System;
    using System.IO;
    using System.Text;
    using System.Net;
    using System.Runtime.Serialization;

    /// <summary>
    /// This class represents an IP subnet.
    /// </summary>
    [DataContract]
    public class IPSubnet : IEquatable<IPSubnet>
    {
        /// <summary>
        /// Prefix delimiter.
        /// </summary>
        public const char PrefixDelimiter = '/';

        /// <summary>
        /// Max prefix length for IPV4 addreeses
        /// </summary>
        public const byte MaxIPv4PrefixLength = 32;

        /// <summary>
        /// Max prefix length for IPV6 addreeses
        /// </summary>
        public const byte MaxIPv6PrefixLength = 128;

        /// <summary>
        /// Default IPV4 address.
        /// </summary>
        public static readonly IPSubnet ThisNetwork = new IPSubnet(IPAddress.Any, 0);

        /// <summary>
        /// IP address of the subnet.
        /// </summary>
        [DataMember]
        public IPAddress IPAddress { get; set; }

        /// <summary>
        /// Prefix of the subnet. For a valid IPv4 subnet, the value ranges from 0 to 32
        /// inclusively.
        /// </summary>
        [DataMember]
        public byte Prefix { get; set; }

        [DataMember]
        public string Name { get; set; }

        /// <summary>
        /// Default constructor.
        /// </summary>
        public IPSubnet()
        {
            IPAddress = null;
            Prefix = 0;
        }

        /// <summary>
        /// Cosntructor.
        /// </summary>
        /// <param name="ipAddress"></param>
        /// <param name="prefix"></param>
        public IPSubnet(IPAddress ipAddress, byte prefix)
        {
            IPAddress = ipAddress;
            Prefix = prefix;
        }

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="ipAddress"></param>
        public IPSubnet(IPAddress ipAddress)
        {
            IPAddress = ipAddress;
            Prefix = ipAddress.AddressFamily == System.Net.Sockets.AddressFamily.InterNetwork
                ? MaxIPv4PrefixLength : MaxIPv6PrefixLength;
        }

        /// <summary>
        /// ToString implementation.
        /// </summary>
        /// <returns>ToString representation of the IPSubnet object.</returns>
        public override string ToString()
        {
            if (this.IPAddress != null)
            {
                StringBuilder sb = new StringBuilder();
                sb.AppendFormat("{0}{1}{2}",
                                this.IPAddress,
                                IPSubnet.PrefixDelimiter,
                                Prefix);
                return sb.ToString();
            }
            else
            {
                return "null";
            }
        }

        /// <summary>
        /// Converts string represenation of IPSubnet to IPSubnet object. 
        /// </summary>
        /// <param name="subnetString"></param>
        /// <returns></returns>
        public static IPSubnet Parse(string subnetString)
        {
            if (string.IsNullOrEmpty(subnetString))
            {
                throw new InvalidDataException(string.Format("Input string:{0} is not in correct format.", subnetString));
            }

            string[] subnetStrings = subnetString.Split('/');
            if (subnetStrings.Length != 2)
            {
                IPAddress ip;
                if (IPAddress.TryParse(subnetString, out ip))
                {
                    byte prefixLength = ip.AddressFamily == System.Net.Sockets.AddressFamily.InterNetwork
                        ? MaxIPv4PrefixLength : MaxIPv6PrefixLength;
                    return new IPSubnet(ip, prefixLength);
                }

                throw new InvalidDataException(string.Format("Input string:{0} is not in correct format.", subnetString));
            }

            IPAddress subnetIpAddress;
            byte subnetPrefix;

            if (IPAddress.TryParse(subnetStrings[0], out subnetIpAddress)
                && byte.TryParse(subnetStrings[1], out subnetPrefix))
            {
                return new IPSubnet(subnetIpAddress, subnetPrefix);
            }
            else
            {
                throw new InvalidDataException(string.Format("Input string:{0} is not in correct format.", subnetString));
            }
        }

        #region IEquatable methods
        public bool Equals(IPSubnet other)
        {
            if (Object.ReferenceEquals(other, null))
            {
                return false;
            }

            if (Object.ReferenceEquals(this, other))
            {
                return true;
            }

            return IPAddress.Equals(other.IPAddress) && Prefix.Equals(other.Prefix);
        }

        public override int GetHashCode()
        {
            int hashPrefix = Prefix.GetHashCode();
            int hashIPAddress = (IPAddress == null) ? 0 : IPAddress.GetHashCode();
            return hashPrefix ^ hashIPAddress;
        }

        #endregion

    }
}
