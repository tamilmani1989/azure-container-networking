// -----------------------------------------------------------------------------
// <copyright file="MacAddressRangeUnitTests.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager.UnitTests
{
    using System;
    using Xunit;

    public class MacAddressRangeUnitTests
    {
        [Fact]
        public void SuccessfulValidate()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = range.RangeStart;
            range.Validate();
        }

        [Fact]
        public void InvalidRangeStart()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeEnd = "AAAAAAAAAAAA";
            Assert.Throws<ArgumentNullException>(() => range.Validate());

            range.RangeStart = "A";
            Assert.Throws<ArgumentException>(() => range.Validate());

            range.RangeStart = "ZZZZZZZZZZZZ";
            Assert.Throws<ArgumentException>(() => range.Validate());
        }

        [Fact]
        public void InvalidRangeEnd()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            Assert.Throws<ArgumentNullException>(() => range.Validate());

            range.RangeEnd = "A";
            Assert.Throws<ArgumentException>(() => range.Validate());

            range.RangeEnd = "ZZZZZZZZZZZZ";
            Assert.Throws<ArgumentException>(() => range.Validate());
        }

        [Fact]
        public void EndRangeBeforeStart()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = "1AAAAAAAAAAA";
            Assert.Throws<ArgumentException>(() => range.Validate());
        }

        [Fact]
        public void LastUpdatedSet()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = range.RangeStart;
            range.LastUpdated = DateTime.UtcNow;
            Assert.Throws<ArgumentException>(() => range.Validate());
        }

        [Fact]
        public void IsFlattenedSet()
        {
            MacAddressRange range = new MacAddressRange();
            range.RangeStart = "AAAAAAAAAAAA";
            range.RangeEnd = range.RangeStart;
            range.LastUpdated = DateTime.UtcNow;
            Assert.Throws<ArgumentException>(() => range.Validate());
        }
    }
}
