//-----------------------------------------------------------------------
// <copyright file="IPHelpers.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// <summary>
//   Implement IPHelpers class.
// </summary>
//-----------------------------------------------------------------------

namespace NetworkManager
{
    using System;
    using System.Net;
    using System.Collections.Generic;
    using System.Numerics;

    using System.Net.Sockets;
    using global::DNCService.API;

    /// <summary>
    /// Utility class exposing functions to handle IP addresses.
    /// </summary>
    public class IPHelpers
    {
        /// <summary>
        /// Byte count of IPv4 address.
        /// </summary>
        internal const byte IPAddressByteCount = 4;

        /// <summary>
        /// Byte count of IPv6 address.
        /// </summary>
        internal const byte IPAddressByteCountV6 = 16;

        /// <summary>
        /// Count of bits in byte
        /// </summary>
        internal const byte ByteBitCount = 8;

        /// <summary>
        /// This helper function parses an input string array into an array of subnets. 
        /// Each string in the array has the following format
        ///   "10.0.0.0/8"
        /// Note, the full IP address must be specified instead "10/8".
        /// </summary>
        /// <param name="addressSpace">Supply the address space string array.</param>
        /// <returns>
        /// If the string array are parsed successfully, an array of IPSubnet is returned. 
        /// Otherwise, exception can be thrown due to parsing error.
        /// </returns>
        internal static IPSubnet[] ParseAddressSpace(string[] addressSpace)
        {
            IPSubnet[] subnets = new IPSubnet[addressSpace.Length];

            // Parse each string into a subnet.
            for (int index = 0; index < addressSpace.Length; index++)
            {
                subnets[index] = IPHelpers.ParseSubnet(addressSpace[index]);
            }

            return subnets;
        }

        /// <summary>
        /// This helper function parse an input string into an IP subnet. The format of the string
        /// is the following: IPAddress/Prefix. Note, the IPAddress must be the complete IP 
        /// address, instead of just the valid bits of the subnet. For example, 10/8 must be
        /// written as "10.0.0.0/8".
        /// </summary>
        /// <param name="subnetString">Supply the input string to parse.</param>
        /// <returns>
        /// If the string is parsed successfully, an IP subnet object is returned. 
        /// Otherwise, exception can be thrown due to parsing error.
        /// </returns>
        internal static IPSubnet ParseSubnet(string subnetString)
        {
            try
            {
                string[] tokens = subnetString.Split(IPSubnet.PrefixDelimiter);
                if (tokens.Length != 2)
                {
                    throw new FormatException("Subnet string is invalid.");
                }

                IPSubnet subnet = new IPSubnet
                {
                    Prefix = byte.Parse(tokens[1])
                };

                // Mask off any non-zero bit in host bits.
                IPAddress ipAddress = IPAddress.Parse(tokens[0]);
                BigInteger ipAddressInBigInteger = IPHelpers.ConvertIPAddressToBigInteger(ipAddress);
                subnet.IPAddress = IPHelpers.ConvertBigIntegerToIPAddress(ipAddressInBigInteger &
                    IPHelpers.GetSubnetMask(subnet.Prefix, ipAddress.AddressFamily), ipAddress.AddressFamily);

                return subnet;
            }
            catch (Exception exception)
            {
                throw new DncManagementException(new ErrorInfo(
                    ErrorCode.InvalidIPSubnet,
                    string.Format("ParseSubnet: Failed to parse as subnet string is invalid:{0} " +
                        "with exception", subnetString, exception),
                    ErrorType.Retriable));
            }
        }

        /// <summary>
        /// Wrapper function to convert an IPv4 or IPv6 address to a host order BigInteger.
        /// </summary>
        /// <param name="ipAddress">Supply the IPv4 or IPv6 address as string.</param>
        /// <returns>IPv4 address as BigInteger in host order.</returns>
        internal static BigInteger ConvertIPAddressToBigInteger(string ipAddress)
        {
            return ConvertIPAddressToBigInteger(IPAddress.Parse(ipAddress));
        }


        /// <summary>
        /// Helper function to convert an IPv4 or IPv6 address to a host order BigInteger.
        /// </summary>
        /// <param name="ipAddress">Supply the IPv4 or IPv6 address.</param>
        /// <returns>IPv4 address as BigInteger in host order.</returns>
        internal static BigInteger ConvertIPAddressToBigInteger(IPAddress ipAddress)
        {
            byte[] addressBytes = ipAddress.GetAddressBytes();

            byte[] paddedAddressBytes = GetUnsignedBigEndianBytes(addressBytes);

            return new BigInteger(paddedAddressBytes);
        }

        /// <summary>
        /// Helper function to convert BigInteger (host order) to IPAddress.
        /// </summary>
        /// <param name="ipAddressInBigInteger">Supply the IP address in BigInteger.</param>
        /// <param name="addressFamily">Address family of the IP address to parse.</param>
        /// <returns>Converted IPAddress.</returns>
        internal static IPAddress ConvertBigIntegerToIPAddress(BigInteger ipAddressInBigInteger, AddressFamily addressFamily)
        {
            // The individual bytes in the array returned by this method appear in little-endian order.
            // msdn.microsoft.com/en-us/library/system.numerics.biginteger.tobytearray(v=vs.110).aspx
            byte[] addressBytes = ipAddressInBigInteger.ToByteArray();
            byte[] unPaddedAddressBytes;

            // Remove padding and get the right size array based on addressfamily
            int arrayLength = (addressFamily == AddressFamily.InterNetworkV6) ? IPAddressByteCountV6 : IPAddressByteCount;
            unPaddedAddressBytes = new byte[arrayLength];

            // copying the sign byte 0x00 is fine because we anyways reverse the order.
            Array.Copy(addressBytes, unPaddedAddressBytes, addressBytes.Length <= arrayLength ? addressBytes.Length : arrayLength);

            // Reversing the order of the bytes is a quick way to get the BitConverter 
            // methods to convert the byte arrays in big-endian mode.
            Array.Reverse(unPaddedAddressBytes);

            // The Byte array is assumed to be in network byte order with the most significant byte first in index position 0.
            // msdn.microsoft.com/en-us/library/t4k07yby(v=vs.110).aspx
            return new IPAddress(unPaddedAddressBytes);
        }

        private static byte[] GetUnsignedBigEndianBytes(byte[] addressBytes)
        {
            if (BitConverter.IsLittleEndian)
            {
                //little-endian machines store multi-byte integers with the 
                //least significant byte first. this is a problem, as integer 
                //values are sent over the network in big-endian mode. reversing 
                //the order of the bytes is a quick way to get the BitConverter 
                //methods to convert the byte arrays in big-endian mode.
                Array.Reverse(addressBytes);
            }

            // Pad with zero to prevent negative numbers
            byte[] paddedAddressBytes = new byte[addressBytes.Length + 1];
            Array.Copy(addressBytes, paddedAddressBytes, addressBytes.Length);
            return paddedAddressBytes;
        }

        /// <summary>
        /// Helper function to return the host mask for the given IP address prefix.
        /// </summary>
        /// <param name="prefix">Supply the prefix to use.</param>
        /// <param name="addressFamily">Address family of subnet.</param>
        /// <returns>Host mask.</returns>
        internal static BigInteger GetHostMask(byte prefix, AddressFamily addressFamily)
        {
            // Set the host mask array to IPV4 prefix 0.
            byte[] hostMaskByteArray = new byte[] { 0xff, 0xff, 0xff, 0xff, /* padding for sign */ 0x00 };

            if (prefix > 0)
            {
                // Get inverse of subnetmask and trim sign byte into host byte array.
                byte[] subnetMaskByteArray = (~GetSubnetMask(prefix, addressFamily)).ToByteArray();
                hostMaskByteArray = new byte[subnetMaskByteArray.Length - 1];
                Array.Copy(subnetMaskByteArray, hostMaskByteArray, hostMaskByteArray.Length);
            }

            // If prefix is zero and IPV6, initialize hostmask array appropriately.
            else if (addressFamily == AddressFamily.InterNetworkV6)
            {
                hostMaskByteArray = new byte[] {
                    0xff, 0xff, 0xff, 0xff  ,  0xff, 0xff, 0xff, 0xff,
                    0xff, 0xff, 0xff, 0xff  ,  0xff, 0xff, 0xff, 0xff, /* padding for sign */ 0x00 };
            }
            return new BigInteger(hostMaskByteArray);
        }

        /// <summary>
        /// Helper function to return the subnet mask for the given prefix.
        /// </summary>
        /// <param name="prefix">Supply the prefix to use.</param>
        /// <returns>Subnet mask.</returns>
        internal static BigInteger GetSubnetMask(byte prefix, AddressFamily addressFamily)
        {
            byte ipAddressMaxPrefix = (byte)(addressFamily == AddressFamily.InterNetworkV6 ?
                IPAddressByteCountV6 * ByteBitCount : IPAddressByteCount * ByteBitCount);

            // Adjust prefix incase of overflow
            prefix = (prefix > ipAddressMaxPrefix) ? (byte)(prefix % ipAddressMaxPrefix) : prefix;

            // Initialize the mask and left shift prefix times.
            BigInteger mask = 1;
            mask = (mask << prefix) - 1;

            // Shift the resultant mask as a whole.
            mask <<= (ipAddressMaxPrefix - prefix);
            return mask;
        }

        /// <summary>
        /// This function calculates the lower and upper bound of the IP
        /// range of the given subnet including reserved IPs.
        /// </summary>
        /// <param name="ipAddress"> The IP Address part of the subnet.</param>
        /// <param name="subnetPrefix"> The prefix part of the subnet.</param>
        /// <param name="lowerBound"> Returns the lower bound of the range,
        /// inclusively.</param>
        /// <param name="upperBound"> Returns the upper bound of the range.
        /// inclusively.</param>
        /// <returns> Number of IP addresses in the given subnet.</returns>
        internal static BigInteger GetSubnetRange(
            IPAddress ipAddress,
            byte subnetPrefix,
            out BigInteger lowerBound,
            out BigInteger upperBound)
        {
            BigInteger hostMask = IPHelpers.GetHostMask(subnetPrefix, ipAddress.AddressFamily);
            BigInteger subnetMask = IPHelpers.GetSubnetMask(subnetPrefix, ipAddress.AddressFamily);
            BigInteger subnetAddress = (IPHelpers.ConvertIPAddressToBigInteger(ipAddress) & subnetMask);
            BigInteger totalAddressCount = hostMask + 1;

            lowerBound = subnetAddress;
            upperBound = subnetAddress + hostMask;

            return totalAddressCount;
        }

        /// <summary>
        /// Helper function to calculated the ip address prefix len.
        /// </summary>
        /// <param name="ip"></param>
        /// <returns>prefix length</returns>
        internal static byte GetIpAddressPrefixLen(IPAddress ip)
        {
            if (ip.AddressFamily == AddressFamily.InterNetwork)
            {
                return IPAddressByteCount * ByteBitCount;
            }

            return IPAddressByteCountV6 * ByteBitCount;
        }

        /// <summary>
        /// Helper function to convert IP address to IP subnet format.
        /// </summary>
        /// <param name="ip">ip address</param>
        /// <returns>IP address in subnet object</returns>
        internal static IPSubnet ConvertIPToIPSubnet(string ip)
        {
            IPAddress ipAddress = IPAddress.Parse(ip);
            byte prefixLen = GetIpAddressPrefixLen(ipAddress);

            return new IPSubnet(ipAddress, prefixLen);
        }

        /// <summary>
        /// Helper function to calculate the prefix of the smallest subnet that can provide the
        /// the specified number of IP address.
        /// Support IPV4 only.
        /// </summary>
        /// <param name="addressCount"></param>
        /// <returns></returns>
        internal static byte CalculateSubnetPrefixIPV4Only(UInt32 addressCount)
        {
            // Find the position of the most significant non-zero bit of addressCount.
            byte msbPosition = 0;
            UInt32 subnetAddressCount = addressCount;
            while (true)
            {
                subnetAddressCount >>= 1;
                if (subnetAddressCount == 0)
                {
                    break;
                }

                msbPosition++;
            }

            // If the addressCount is not a perfect power of two, need to add one more bit.
            subnetAddressCount = 1U << msbPosition;
            if (subnetAddressCount < addressCount)
            {
                msbPosition++;
            }

            // return IPV4 prefix.
            return (byte)(IPHelpers.IPAddressByteCount * 8 - msbPosition);
        }

        /// <summary>
        /// Helper function that get the string representation of an IP address
        /// in UInt32 (host order).
        /// </summary>
        /// <param name="addressArray">Supply the IPv4 address in byte array.</param>
        /// <returns>String representation of the IPv4 address.</returns>
        public static string GetIPAddressString(byte[] addressArray)
        {
            IPAddress ipAddress = new IPAddress(addressArray);

            return ipAddress.ToString();
        }

        /// <summary>
        /// Helper function to return the host mask for the given IP address prefix.
        /// </summary>
        /// <param name="prefix">Supply the prefix to use.</param>
        /// <returns>Host mask.</returns>
        public static UInt32 GetHostMask(byte prefix)
        {
            // This method is not in perf-sensitive code path
            if (prefix > 32)
            {
                throw new ArgumentOutOfRangeException("prefix", prefix, "Valid range of prefix is [0, 32]");
            }

            // Ensure the overflow does not happen
            return prefix == 0 ? 0xFFFFFFFF : ((1U << (IPHelpers.IPAddressByteCount * 8 - prefix)) - 1);
        }

        /// <summary>
        /// Helper function to return the subnet mask for the given prefix.
        /// </summary>
        /// <param name="prefix">Supply the prefix to use.</param>
        /// <returns>Subnet mask.</returns>
        public static UInt32 GetSubnetMask(byte prefix)
        {
            return ~IPHelpers.GetHostMask(prefix);
        }

        /// <summary>
        /// Helper function to calculate the prefix of the smallest subnet that can provide the
        /// the specified number of IP address.
        /// </summary>
        /// <param name="addressCount"></param>
        /// <returns></returns>
        public static byte CalculateSubnetPrefix(UInt32 addressCount)
        {
            // Find the position of the most significant non-zero bit of addressCount.
            byte msbPosition = 0;
            UInt32 subnetAddressCount = addressCount;
            while (true)
            {
                subnetAddressCount >>= 1;
                if (subnetAddressCount == 0)
                {
                    break;
                }

                msbPosition++;
            }

            // If the addressCount is not a perfect power of two, need to add one more bit.
            subnetAddressCount = 1U << msbPosition;
            if (subnetAddressCount < addressCount)
            {
                msbPosition++;
            }

            return (byte)(IPHelpers.IPAddressByteCount * 8 - msbPosition);
        }

        /// <summary>
        /// Helper function to determine if two IP subnets overlap.
        /// </summary>
        /// <param name="firstSubnet">Supply the first subnet.</param>
        /// <param name="secondSubnet">Supply the second subnet.</param>
        /// <returns>True if the two subnets overlap. False, otherwise.</returns>
        internal static bool AreOverlappingSubnets(
            IPSubnet firstSubnet,
            IPSubnet secondSubnet
            )
        {
            // If not same address family return false.
            if (firstSubnet.IPAddress.AddressFamily != secondSubnet.IPAddress.AddressFamily)
            {
                return false;
            }

            Byte prefixLength;

            // Pick the smallest prefix of the two subnet prefixes to compute the network address.
            if (firstSubnet.Prefix < secondSubnet.Prefix)
            {
                prefixLength = firstSubnet.Prefix;
            }
            else
            {
                prefixLength = secondSubnet.Prefix;
            }

            BigInteger firstSubnetNetworkAddress =
                (IPHelpers.ConvertIPAddressToBigInteger(firstSubnet.IPAddress) &
                 IPHelpers.GetSubnetMask(prefixLength, firstSubnet.IPAddress.AddressFamily));

            BigInteger secondSubnetNetworkAddress =
                (IPHelpers.ConvertIPAddressToBigInteger(secondSubnet.IPAddress) &
                 IPHelpers.GetSubnetMask(prefixLength, secondSubnet.IPAddress.AddressFamily));

            if (firstSubnetNetworkAddress == secondSubnetNetworkAddress)
            {
                return true;
            }
            else
            {
                return false;
            }
        }

        /// <summary>
        /// Helper function to check if the subnets in a list are exclusive (i.e.) non overlapping.
        /// </summary>
        /// <param name="subnets">Supply the list of subnets.</param>
        /// <returns>True if there are no overlapping subnets, False otherwise.</returns>
        public static bool AreSubnetsSelfContained(IPSubnet[] subnets)
        {
            int overlappedIndex1;
            int overlappedIndex2;

            return AreSubnetsSelfContained(subnets, out overlappedIndex1, out overlappedIndex2);
        }

        /// <summary>
        /// Helper function to check if the subnets in a list are exclusive (i.e.) non overlapping.
        /// If overlapping returns the indices of the overlapping subnets in the subnet array.
        /// </summary>
        /// <param name="subnets">Supply the list of subnets.</param>
        /// <param name="overlappedIndex1">
        /// If overlapping, contains the first index of the overlapping subnet in the subnet array.
        /// </param>
        /// <param name="overlappedIndex2">
        /// If overlapping, contains the second index of the overlapping subnet in the subnet array.
        /// subnet.        /// </param>
        /// <returns>True if there are no overlapping subnets, False otherwise.</returns>
        public static bool AreSubnetsSelfContained(
            IPSubnet[] subnets,
            out int overlappedIndex1,
            out int overlappedIndex2)
        {
            overlappedIndex2 = 0;
            for (overlappedIndex1 = 0; overlappedIndex1 < subnets.Length; overlappedIndex1++)
            {
                for (overlappedIndex2 = overlappedIndex1 + 1;
                    overlappedIndex2 < subnets.Length;
                    overlappedIndex2++)
                {
                    if (IPHelpers.AreOverlappingSubnets(
                        subnets[overlappedIndex1],
                        subnets[overlappedIndex2]))
                    {
                        return false;
                    }
                }
            }

            return true;
        }

        /// <summary>
        /// Helper function to check if a given subnet is contained within a subnet that is 
        /// part of a subnet collection.
        /// </summary>
        /// <param name="subnet">Supply the subnet.</param>
        /// <param name="subnetCollection">Supply the subnet collection.</param>
        /// <returns>
        /// True if there is a subnet in the collection that contains the given subnet range, 
        /// False otherwise.
        /// </returns>
        public static bool IsSubnetContainedInCollection(
            IPSubnet[] subnetCollection,
            IPSubnet subnet)
        {
            for (int index = 0; index < subnetCollection.Length; index++)
            {
                if (subnetCollection[index].Prefix <= subnet.Prefix)
                {
                    if (IPHelpers.AreOverlappingSubnets(subnetCollection[index], subnet))
                    {
                        return true;
                    }
                }
            }

            return false;
        }

        /// <summary>
        /// Helper function to determine if an IPAddress list has changed.
        /// </summary>
        /// <param name="existingIPAddresses">Supply the existing list of IP addresses.</param>
        /// <param name="newIPAddresses">Supply the new list of IP addresses.</param>
        /// <returns>True if the two are different. False, otherwise.</returns>
        internal static bool HasIPAddressListChanged(
            List<IPAddress> existingIPAddresses,
            List<IPAddress> newIPAddresses
            )
        {
            if (existingIPAddresses.Count != newIPAddresses.Count)
            {
                return true;
            }

            // Declare a hash table that maps BigInteger representation of the IP address.
            HashSet<BigInteger> addressSet = new HashSet<BigInteger>();

            // Populate the hash table based on the existing address space.
            foreach (IPAddress ipAddress in existingIPAddresses)
            {
                addressSet.Add(
                    IPHelpers.ConvertIPAddressToBigInteger(ipAddress));
            }

            // Check if the addresses from the new address list are contained in the existing
            // address list.
            foreach (IPAddress ipAddress in newIPAddresses)
            {
                if (addressSet.Contains(IPHelpers.ConvertIPAddressToBigInteger(ipAddress)) == false)
                {
                    return true;
                }
            }

            return false;
        }

        public static BigInteger GetFirstAddressInSubnetAsBigInteger(IPSubnet ipSubnet)
        {
            return (IPHelpers.ConvertIPAddressToBigInteger(ipSubnet.IPAddress) &
                IPHelpers.GetSubnetMask(ipSubnet.Prefix, ipSubnet.IPAddress.AddressFamily));
        }

        public static IPAddress GetFirstAddressInSubnet(IPSubnet ipSubnet)
        {
            BigInteger firstNetworkAddress = GetFirstAddressInSubnetAsBigInteger(ipSubnet);

            return IPHelpers.ConvertBigIntegerToIPAddress(
                firstNetworkAddress,
                ipSubnet.IPAddress.AddressFamily);
        }

        public static BigInteger GetLastAddressInSubnetAsBigInteger(IPSubnet ipSubnet)
        {
            return GetFirstAddressInSubnetAsBigInteger(ipSubnet) + IPHelpers.GetHostMask(ipSubnet.Prefix, AddressFamily.InterNetwork);
        }

        public static IPAddress GetLastAddressInSubnet(IPSubnet ipSubnet)
        {
            return IPHelpers.ConvertBigIntegerToIPAddress(
                GetLastAddressInSubnetAsBigInteger(ipSubnet),
                ipSubnet.IPAddress.AddressFamily);
        }
    }
}
