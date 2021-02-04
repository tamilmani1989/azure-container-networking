// -----------------------------------------------------------------------------
// <copyright file="MacAllocatorServiceUnitTest.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

using Microsoft.Extensions.Logging;

namespace Microsoft.Azure.ContainerNetworking.MacAllocator.UnitTests
{
    using System;
    using System.Collections.Generic;
    using System.Threading.Tasks;

    using Microsoft.Azure.ContainerNetworking.CoreLib;
    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Azure.Cosmos;
    using Microsoft.Extensions.Options;

    using Moq;
    using Xunit;

    public class MacAllocatorServiceUnitTest
    {
        private readonly MacAllocatorService service;
        private readonly Mock<ILogger<MacAllocatorService>> mockLogger;
        private readonly Mock<IMacAddressComponent> addressComponentMock;
        private readonly Mock<IMetricsProvider> metricsProvider;

        public MacAllocatorServiceUnitTest()
        {
            this.mockLogger = new Mock<ILogger<MacAllocatorService>>();
            this.addressComponentMock = new Mock<IMacAddressComponent>();
            this.metricsProvider = new Mock<IMetricsProvider>();
            MacAllocatorSetting setting = new MacAllocatorSetting()
            {
                AllocationRetryCount = "5",
            };
            IOptions<MacAllocatorSetting> options = Options.Create<MacAllocatorSetting>(setting);
            this.service = new MacAllocatorService(this.mockLogger.Object, this.addressComponentMock.Object, options, this.metricsProvider.Object);
        }

        [Fact]
        public async Task TestMacAllocation()
        {
            List<MacAddress> addresses = new List<MacAddress>();
            addresses.Add(new MacAddress()
            {
                Address = "AAAAAAAAAAAA",
                ReservedBy = string.Empty,
            });

            this.addressComponentMock
                .Setup(_ => _.GetMacAddressesAsync(It.IsAny<QueryDefinition>()))
                .ReturnsAsync((QueryDefinition q) => 
                {
                    // return empty list when get method is called to get MAC addressed reserved by this allocation id.
                    return q.QueryText.Contains("SELECT * FROM") ? new List<MacAddress>() : addresses;
                });

            MacAllocatorRequest r = new MacAllocatorRequest()
            {
                AllocationId = "testid"
            };

            MacAddress macAddress = await service.AllocateMacAddressAsync(r);

            Assert.Equal("AAAAAAAAAAAA", macAddress.Address);
            Assert.Equal("testid", macAddress.ReservedBy);
            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(addresses[0]), Times.Once);
        }

        [Fact]
        public async Task TestMacAllocationMacAlreadyReservedbyOtherId()
        {
            List<MacAddress> addresses = new List<MacAddress>();
            addresses.Add(new MacAddress()
            {
                Address = "AAAAAAAAAAAA",
                ReservedBy = "testid1",
            });
            addresses.Add(new MacAddress()
            {
                Address = "AAAAAAAAAAA1",
                ReservedBy = "testid2",
            });

            this.addressComponentMock
                .Setup(_ => _.GetMacAddressesAsync(It.IsAny<QueryDefinition>()))
                .ReturnsAsync((QueryDefinition q) =>
                {
                    // return empty list when get method is called to get MAC addressed reserved by this allocation id.
                    return q.QueryText.Contains("SELECT * FROM") ? new List<MacAddress>() : addresses;
                });

            MacAllocatorRequest r = new MacAllocatorRequest()
            {
                AllocationId = "testid"
            };

            await Assert.ThrowsAsync<InvalidOperationException>(async () => { await service.AllocateMacAddressAsync(r); });
            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(It.IsAny<MacAddress>()), Times.Never);
        }

        [Fact]
        public async Task TestMacAllocationMacAlreadyReserved()
        {
            List<MacAddress> addresses = new List<MacAddress>();
            addresses.Add(new MacAddress()
            {
                Address = "AAAAAAAAAAAA",
                ReservedBy = "testid",
            });

            this.addressComponentMock
            .Setup(_ => _.GetMacAddressesAsync(It.IsAny<QueryDefinition>()))
            .ReturnsAsync(addresses);

            MacAllocatorRequest r = new MacAllocatorRequest()
            {
                AllocationId = "testid"
            };

            MacAddress macAddress = await service.AllocateMacAddressAsync(r);

            Assert.Equal("AAAAAAAAAAAA", macAddress.Address);
            Assert.Equal("testid", macAddress.ReservedBy);
            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(It.IsAny<MacAddress>()), Times.Never);
        }

        [Fact]
        public async Task TestMacAllocationWithNoMacAvailable()
        {
            this.addressComponentMock
            .Setup(_ => _.GetMacAddressesAsync(It.IsAny<QueryDefinition>()))
            .ReturnsAsync(new List<MacAddress>());

            MacAllocatorRequest r = new MacAllocatorRequest()
            {
                AllocationId = "testid"
            };

            await Assert.ThrowsAsync<InvalidOperationException>(async () => { await service.AllocateMacAddressAsync(r); });
        }

        [Fact]
        public async Task TestMacDeallocation()
        {
            List<MacAddress> addresses = new List<MacAddress>();
            addresses.Add(new MacAddress()
            {
                Address = "AAAAAAAAAAAA",
                ReservedBy = string.Empty,
                ReservedDateTime = DateTime.MinValue,
            });

            this.addressComponentMock
            .Setup(_ => _.GetMacAddressesAsync(It.IsAny<QueryDefinition>()))
            .ReturnsAsync(addresses);

            MacAllocatorRequest r = new MacAllocatorRequest()
            {
                AllocationId = "testid"
            };

            await service.DeallocateMacAddressAsync(r);
            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(addresses[0]), Times.Once);
        }

        [Fact]
        public async Task TestMacDeallocationWithNoMacReserved()
        {
            this.addressComponentMock
            .Setup(_ => _.GetMacAddressesAsync(It.IsAny<QueryDefinition>()))
            .ReturnsAsync(new List<MacAddress>());

            MacAllocatorRequest r = new MacAllocatorRequest()
            {
                AllocationId = "testid"
            };

            await service.DeallocateMacAddressAsync(r);
            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(It.IsAny<MacAddress>()), Times.Never);
        }

        [Fact]
        public async Task TestMacDeallocationFailed()
        {
            List<MacAddress> addresses = new List<MacAddress>();
            addresses.Add(new MacAddress()
            {
                Address = "AAAAAAAAAAAA",
                ReservedBy = string.Empty,
                ReservedDateTime = DateTime.MinValue,
            });
            addresses.Add(new MacAddress()
            {
                Address = "AAAAAAAAAAA1",
                ReservedBy = string.Empty,
                ReservedDateTime = DateTime.MinValue,
            });

            this.addressComponentMock
            .Setup(_ => _.GetMacAddressesAsync(It.IsAny<QueryDefinition>()))
            .ReturnsAsync(addresses);

            this.addressComponentMock
                .Setup(_ => _.UpsertMacAddressAsync(It.IsAny<MacAddress>()))
                .Callback((MacAddress a) => 
                {
                    if (string.Equals(a.Address, "AAAAAAAAAAA1")) throw new InvalidOperationException();
                });

            MacAllocatorRequest r = new MacAllocatorRequest()
            {
                AllocationId = "testid"
            };

            await Assert.ThrowsAsync<AggregateException>(async () => { await service.DeallocateMacAddressAsync(r); });
            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(addresses[0]), Times.Once);
        }
    }
}
