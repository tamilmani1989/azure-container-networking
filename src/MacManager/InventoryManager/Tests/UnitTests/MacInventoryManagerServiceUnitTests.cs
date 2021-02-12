// -----------------------------------------------------------------------------
// <copyright file="MacInventoryManagerServiceUnitTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager.UnitTests
{
    using System;
    using System.Collections.Generic;
    using System.Linq;
    using System.Threading.Tasks;
    using Microsoft.Azure.ContainerNetworking.MacInventoryManager.DataAccess;
    using Microsoft.Azure.ContainerNetworking.MacManager.Common;
    using Microsoft.Extensions.Logging;
    using Moq;
    using Xunit;

    public class MacInventoryManagerServiceUnitTests
    {
        private readonly MacInventoryManagerService service;
        private readonly Mock<ILogger<MacInventoryManagerService>> mockLogger;
        private readonly Mock<IMacAddressRangeComponent> rangeComponentMock;
        private readonly Mock<IMacAddressComponent> addressComponentMock;
        private readonly MacInventoryInitializationState initializationState = new MacInventoryInitializationState();

        public MacInventoryManagerServiceUnitTests()
        {
            this.mockLogger = new Mock<ILogger<MacInventoryManagerService>>();
            this.rangeComponentMock = new Mock<IMacAddressRangeComponent>();
            this.addressComponentMock = new Mock<IMacAddressComponent>();
            this.initializationState.IsInitialized = true;

            this.service = new MacInventoryManagerService(
                this.mockLogger.Object,
                this.rangeComponentMock.Object,
                this.addressComponentMock.Object,
                this.initializationState);

            this.rangeComponentMock
                .Setup(_ => _.GetRangesAsync())
                .ReturnsAsync(new List<MacAddressRange>());

            this.addressComponentMock
                .Setup(_ => _.GetMacAddressesAsync(It.IsAny<string>(), It.IsAny<string>()))
                .ReturnsAsync(new List<MacAddress>());
        }

        [Fact]
        public async Task UpdateRangeIsIdempotent()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = range.RangeStart;

            MacAddressRange existing = new MacAddressRange();
            existing.RangeStart = range.RangeStart;
            existing.RangeEnd = range.RangeEnd;

            List<MacAddressRange> existingRanges = new List<MacAddressRange>();
            existingRanges.Add(existing);

            this.rangeComponentMock
                .Setup(_ => _.GetRangesAsync())
                .ReturnsAsync(existingRanges);

            await service.UpdateRangeAsync(range);

            // try with the existing state as blocked.
            existing.BlockedReason = "test block";
            range.IsFlattened = false;
            await service.UpdateRangeAsync(range);

            // try with the new state as blocked as well.
            range.BlockedReason = "test block 2";
            range.IsFlattened = false;
            await service.UpdateRangeAsync(range);

            // try with the new state as blocked, existing as unblocked
            range.BlockedReason = "test block 2";
            existing.BlockedReason = null;
            range.IsFlattened = false;
            await service.UpdateRangeAsync(range);
        }

        [Fact]
        public async Task NewRangeIsContained()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = "CCCCCCCCCCCC";

            MacAddressRange existing = new MacAddressRange();
            existing.RangeStart = "AAAAAAAAAAA1";
            existing.RangeEnd = "AAAAAAAAAAAB";

            List<MacAddressRange> existingRanges = new List<MacAddressRange>();
            existingRanges.Add(existing);

            this.rangeComponentMock
                .Setup(_ => _.GetRangesAsync())
                .ReturnsAsync(existingRanges);

            await Assert.ThrowsAsync<InvalidOperationException>(() => { return service.UpdateRangeAsync(range); });

            existing.RangeStart = "CCCCCCCCCCCA";
            existing.RangeEnd = "CCCCCCCCCCCC";
            await Assert.ThrowsAsync<InvalidOperationException>(() => { return service.UpdateRangeAsync(range); });
        }

        [Fact]
        public async Task ExistingRangeIsContained()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = "CCCCCCCCCCCC";

            MacAddressRange existing = new MacAddressRange();
            existing.RangeStart = "BBBBBBBBBBBB";
            existing.RangeEnd = "DDDDDDDDDDDD";

            List<MacAddressRange> existingRanges = new List<MacAddressRange>();
            existingRanges.Add(existing);

            this.rangeComponentMock
                .Setup(_ => _.GetRangesAsync())
                .ReturnsAsync(existingRanges);

            await Assert.ThrowsAsync<InvalidOperationException>(() => { return service.UpdateRangeAsync(range); });

            range.RangeStart = "CCCCCCCCCCCC";
            range.RangeEnd = "DDDDDDDDDDDD";
            await Assert.ThrowsAsync<InvalidOperationException>(() => { return service.UpdateRangeAsync(range); });
        }

        [Fact]
        public async Task BlockedRangeMustBeContained()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = "AAAAAAAAAAAA";
            range.BlockedReason = "this is a problem";

            await Assert.ThrowsAsync<InvalidOperationException>(() => { return service.UpdateRangeAsync(range); });

            MacAddressRange existing = new MacAddressRange();
            existing.RangeStart = "AAAAAAAAAAAA";
            existing.RangeEnd = "AAAAAAAAAAAB";

            List<MacAddressRange> existingRanges = new List<MacAddressRange>();
            existingRanges.Add(existing);

            this.rangeComponentMock
                .Setup(_ => _.GetRangesAsync())
                .ReturnsAsync(existingRanges);

            // this should succeed
            await service.UpdateRangeAsync(range);

            // this should fail
            existing.BlockedReason = "test block";
            range.IsFlattened = false;
            await Assert.ThrowsAsync<InvalidOperationException>(() => { return service.UpdateRangeAsync(range); });
        }

        [Fact]
        public async Task UpdateUsesEtag()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = "AAAAAAAAAAAC";

            List<MacAddress> addresses = new List<MacAddress>();
            addresses.Add(new MacAddress()
            {
                Address = range.RangeStart,
                ETag = "testetag",
                IsBlocked = true
            });

            this.addressComponentMock
            .Setup(_ => _.GetMacAddressesAsync(range.RangeStart, range.RangeEnd))
            .ReturnsAsync(addresses);

            await service.UpdateRangeAsync(range);

            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(addresses[0]), Times.Once);
            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(It.IsAny<MacAddress>()), Times.Exactly(3));
        }

        [Fact]
        public async Task InitializationFails()
        {
            this.initializationState.IsInitialized = false;
            await Assert.ThrowsAsync<InvalidOperationException>(() => { return service.UpdateRangeAsync(null); });
            await Assert.ThrowsAsync<InvalidOperationException>(() => { return service.GetRangesAsync(); });
        }


        [Fact]
        public async Task UpdateIsNoOp()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = "AAAAAAAAAAAA";

            List<MacAddress> addresses = new List<MacAddress>();
            addresses.Add(new MacAddress()
            {
                Address = range.RangeStart,
                ETag = "testetag",
            });

            this.addressComponentMock
            .Setup(_ => _.GetMacAddressesAsync(range.RangeStart, range.RangeEnd))
            .ReturnsAsync(addresses);

            await service.UpdateRangeAsync(range);

            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(addresses[0]), Times.Never);
            this.addressComponentMock.Verify(_ => _.UpsertMacAddressAsync(It.IsAny<MacAddress>()), Times.Never);
        }

        [Fact]
        public async Task GetStats()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = "BAAAAAAAAAAA";
            range.BlockedReason = "test block";

            List<MacAddressRange> ranges = new List<MacAddressRange>();
            ranges.Add(range);
            this.rangeComponentMock.Setup(_ => _.GetRangesAsync()).ReturnsAsync(ranges);

            long reservedCount = new Random().Next();
            this.addressComponentMock
            .Setup(_ => _.GetReservedCountAsync(range.RangeStart, range.RangeEnd))
            .ReturnsAsync(reservedCount);

            IEnumerable<MacAddressStats> stats = await service.GetMacAddressStatsAsync();

            Assert.Single(stats);
            MacAddressStats stats1 = stats.First();
            Assert.Equal(range.RangeStart, stats1.RangeStart);
            Assert.Equal(range.RangeEnd, stats1.RangeEnd);
            Assert.Equal(range.BlockedReason, stats1.BlockedReason);

            (long start, long end) = range.ConvertRanges();
            Assert.Equal((end - start) + 1, stats1.TotalCount);
            Assert.Equal(reservedCount, stats1.ReservedCount);
        }
    }
}
