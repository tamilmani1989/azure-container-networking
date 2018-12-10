//-----------------------------------------------------------------------
// <copyright file="AllocationHelper.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// <summary>
//   Implement AllocationHelper class.
// </summary>
//-----------------------------------------------------------------------

namespace NetworkManager
{
    using System;
    using System.Collections.Generic;
    using System.Net;
    using System.Net.Sockets;
    using System.Numerics;
    using System.Linq;
    using global::DNCService.API;
    using DNCService;

    /// <summary>
    /// Implements helper methods for allocations.
    /// </summary>
    internal class AllocationHelper
    {
        /// <summary>
        /// Default IPV4 prefix.
        /// </summary>
        const byte IPV4DefaultPrefix = 32;

        /// <summary>
        /// Allocate a subnet with a given prefix from a base subnet.
        /// </summary>
        /// <param name="baseSubnet">Supply the base subnet.</param>
        /// <param name="subnetsToIgnore">Supply the list of subnets to ignore.</param>
        /// <param name="subnetPrefixToAllocate">Supply the prefix szie to allocate.</param>
        /// <returns></returns>
        public static IPSubnet AllocateSubnetFromBaseSubnet(
            IPSubnet baseSubnet, 
            List<IPSubnet> subnetsToIgnore,
	        byte subnetPrefixToAllocate)
        {
            if (baseSubnet.IPAddress.AddressFamily != AddressFamily.InterNetwork)
            {
                throw new DncManagementException(new ErrorInfo(
                    ErrorCode.UnSupportedAddressFamily,
                    string.Format("Address family {0} not supproted", baseSubnet.IPAddress.AddressFamily),
                    ErrorType.Retriable));
            }

            BigInteger firstAllocatedIPAdressInBaseSubnet = IPHelpers.GetFirstAddressInSubnetAsBigInteger(baseSubnet);
            SortedList<BigInteger, byte> allocatedSubnets = new SortedList<BigInteger, byte>();
	        foreach(IPSubnet ipSubnet in subnetsToIgnore)
            {
                BigInteger firstAllocatedIPAdress = IPHelpers.GetFirstAddressInSubnetAsBigInteger(ipSubnet);
                allocatedSubnets[firstAllocatedIPAdress] = ipSubnet.Prefix;
            }

            IPSubnet newSubnet = AllocateSubnetFromBaseSubnetIPV4Only(
                firstAllocatedIPAdressInBaseSubnet + 4,
                baseSubnet.Prefix,
                allocatedSubnets,
                subnetPrefixToAllocate);

            return newSubnet;
        }

        /// <summary>
        /// Allocate a subnet from the unallocated address ranges of the VNet subnet.
        /// Allocate subnets for IPV4 only.
        /// </summary>
        /// <param name="vnetSubnet"> Supply the Base subnet to use.</param>
        /// <param name="allocatedSubnetList">
        /// Supply the list of allocated subnets. Note, there can be allocated subnet outside
        /// the specified VNet subnet. This can happen when the VNet space is composed of more
        /// than one subnet.
        /// </param>
        /// <param name="newSubnetPrefix"> Supply the prefix of the new subnet.</param>
        /// <returns> Allocated subnet if allocation is successful. Null, otherwise.</returns>
        private static IPSubnet AllocateSubnetFromBaseSubnetIPV4Only(
            BigInteger baseSubnetAddress,
            byte baseSubnetPrefix,
            SortedList<BigInteger, byte> allocatedSubnetList,
            byte newSubnetPrefix)
        {

            // Locate the first allocated subnet contained within the vnetSubnet.
            int index = 0;
            bool hasAllocatedSubnet = false;

            // Hardcoded IPV4 addressfamily
            BigInteger baseSubnetMask = IPHelpers.GetSubnetMask(baseSubnetPrefix, AddressFamily.InterNetwork);
            while (index < allocatedSubnetList.Count)
            {
                BigInteger allocatedSubnetAddress = allocatedSubnetList.Keys[index];
                ServiceEventSource.Current.Message($"baseSubnetAddress:{baseSubnetAddress} " +
                    $"Maskedaddress:{allocatedSubnetAddress & baseSubnetMask}");
                if ((baseSubnetAddress & baseSubnetMask) == (allocatedSubnetAddress & baseSubnetMask))
                {
                    hasAllocatedSubnet = true;
                    break;
                }

                index++;
            }

            // If there is no subnet allocated from this base subnet, directly allocate baseSubnet.
            if (hasAllocatedSubnet == false)
            {
                ServiceEventSource.Current.Message($"Has no allocated subnets");
                return AllocateSubnetFromAddressRangeIPV4Only(
                    baseSubnetAddress,
                    (baseSubnetAddress + IPHelpers.GetHostMask(baseSubnetPrefix, AddressFamily.InterNetwork) + 1),
                    newSubnetPrefix);
            }

            // Check the available address ranges between allocated subnets and see if
            // an address range can satisfy the new subnet allocation.
            BigInteger startingAddress = baseSubnetAddress;
            while (index < allocatedSubnetList.Count)
            {
                // If the allocated subnet is outside vnetSubnet, exit the loop.
                BigInteger allocatedSubnetAddress = allocatedSubnetList.Keys[index];
                if ((baseSubnetAddress & baseSubnetMask) != (allocatedSubnetAddress & baseSubnetMask))
                {
                    break;
                }

                // See if the gap between starting address and the beginning address of
                // the allocated subnet can satisfy the new allocation request.
                IPSubnet subnet = AllocateSubnetFromAddressRangeIPV4Only(
                    startingAddress,
                    allocatedSubnetAddress,
                    newSubnetPrefix);

                if (subnet != null)
                {
                    return subnet;
                }

                // Advance the starting address to the end of the allocated subnet.
                byte allocatedSubnetPrefix = allocatedSubnetList.Values[index];
                startingAddress =
                    (allocatedSubnetAddress + IPHelpers.GetHostMask(allocatedSubnetPrefix, AddressFamily.InterNetwork) + 1);

                index++;
            }

            // No allocation possible from the gaps between starting address and the beginning
            // address of each allocated subnet. Look at the gap from the ending address of 
            // the last allocated subnet to the end of vnetSubnet.
            index--;
            startingAddress =
                (allocatedSubnetList.Keys[index] +
                 (UInt32)IPHelpers.GetHostMask(allocatedSubnetList.Values[index], AddressFamily.InterNetwork) +
                 1);

            return AllocateSubnetFromAddressRangeIPV4Only(
                startingAddress,
                (UInt32)(baseSubnetAddress + IPHelpers.GetHostMask(baseSubnetPrefix, AddressFamily.InterNetwork) + 1),
                newSubnetPrefix);
        }
        
        public static List<IPAddress> GetDNSServerAddressesForSubnet(IPSubnet ipSubnet)
        {
            List<IPAddress>  dnsServerAddresses = new List<IPAddress>();
            BigInteger firstAddress = IPHelpers.GetFirstAddressInSubnetAsBigInteger(ipSubnet);
            dnsServerAddresses.Add(IPHelpers.ConvertBigIntegerToIPAddress(firstAddress + 2, ipSubnet.IPAddress.AddressFamily));
            dnsServerAddresses.Add(IPHelpers.ConvertBigIntegerToIPAddress(firstAddress + 3, ipSubnet.IPAddress.AddressFamily));

            return dnsServerAddresses;
        }

        /// <summary>
        /// Helper function to allocate a subnet from an address range. Note, the range includes
        /// the starting address, but not ending address, i.e. [startingAddress, endingAddress).
        /// </summary>
        /// <param name="startingAddress">Supply the starting address of the address range.</param>
        /// <param name="endingAddress">Supply the ending address of the address range.</param>
        /// <param name="subnetPrefix"> Supply the prefix of the subnet to allocate.</param>
        /// <returns>IPSubnet if allocation is successful. False, otherwise.</returns>
        private static IPSubnet AllocateSubnetFromAddressRangeIPV4Only(
            BigInteger startingAddress,
            BigInteger endingAddress,
            byte subnetPrefix)
        {
            // Determine the number of bits for host part of the IP address.
            byte hostBitCount = (byte)(IPHelpers.IPAddressByteCount * 8 - subnetPrefix);

            // Align the starting address to the size of the subnet to allocate.
            UInt32 subnetHostCount = (UInt32)IPHelpers.GetHostMask(subnetPrefix, AddressFamily.InterNetwork) + 1;
            startingAddress = ((startingAddress + subnetHostCount - 1) >> hostBitCount);
            startingAddress <<= hostBitCount;

            // Check if there are enough addresses in the range. 
            if ((startingAddress + subnetHostCount) <= endingAddress)
            {
                // There are enough addresses, create a new subnet and return it.
                IPSubnet subnet = new IPSubnet();
                subnet.Prefix = subnetPrefix;
                subnet.IPAddress = IPHelpers.ConvertBigIntegerToIPAddress(startingAddress, AddressFamily.InterNetwork);
                return subnet;
            }
            else
            {
                return null;
            }
        }

        /// <summary>
        /// Helepr to get list of addresses in a subnet.
        /// </summary>
        /// <param name="ipSubnet">Supply the IP subnet.</param>
        /// <returns>Collection of IP addresses in the subnet.</returns>
        public static List<IPAddress> GetAllAddressInSubnet(IPSubnet ipSubnet)
        {
            BigInteger firstAddress = IPHelpers.GetFirstAddressInSubnetAsBigInteger(ipSubnet);
            BigInteger endingAddress = IPHelpers.GetLastAddressInSubnetAsBigInteger(ipSubnet);
            List<IPAddress> addresses = new List<IPAddress>();
            for (BigInteger i = firstAddress; i < endingAddress; i++)
            {
                addresses.Add(IPHelpers.ConvertBigIntegerToIPAddress(i, ipSubnet.IPAddress.AddressFamily));
            }

            return addresses;
        }

        /// <summary>
        /// Gets the IP subnet for a given constraint.
        /// </summary>
        /// <param name="ipConstraint">Supply the IP constraint.(IPAddress)</param>
        /// <returns>IPSubent associated with the constraint.</returns>
        public static IPSubnet GetIPSubnetForIPAllocationConstraint(string ipConstraint)
        {
            bool isIPConstraintASubnet = ipConstraint.Contains(IPSubnet.PrefixDelimiter);
            string ipSubnetConstraint = ipConstraint;

            if (!isIPConstraintASubnet)
            {
                return new IPSubnet(IPAddress.Parse(ipConstraint));
            }

            return IPSubnet.Parse(ipConstraint);
        }

        /// <summary>
        /// Helper to check if a given subnet is in a base subnet.
        /// </summary>
        /// <param name="baseSubnet">Supply the base subnet</param>
        /// <param name="queriedSubnet">Supply the subnet that needs to be checked.</param>
        /// <returns>True if the queried subnet is contained in the base subnet.</returns>
        public static bool IsSubnetContainedInBaseSubnet(IPSubnet baseSubnet, IPSubnet queriedSubnet)
        {
            if (baseSubnet.Prefix <= queriedSubnet.Prefix)
            {
                if (IPHelpers.AreOverlappingSubnets(baseSubnet, queriedSubnet))
                {
                    return true;
                }
            }

            return false;
        }

        /// <summary>
        /// Helper to check if the requested subnet is free.
        /// </summary>
        /// <param name="baseIPSubnet">Supply the base subnet.</param>
        /// <param name="allocatedSubnets">Supply the list of allcoated subnets.</param>
        /// <param name="requestedSubnet">Supply the requested subnet.</param>
        /// <returns>True if the requested subnet is free.</returns>
        public static bool IsRequestedRangeAllocatable(
            IPSubnet baseIPSubnet,
            Dictionary<string, IPSubnet> allocatedSubnets, 
            IPSubnet requestedSubnet)
        {
            if (!IsSubnetContainedInBaseSubnet(baseIPSubnet, requestedSubnet))
            {
                return false;
            }

            foreach(IPSubnet allocatedSubnet in allocatedSubnets.Values)
            {
                if(IPHelpers.AreOverlappingSubnets(allocatedSubnet, requestedSubnet))
                {
                    return false;
                }
            }

            return true;
        }
    }
}
