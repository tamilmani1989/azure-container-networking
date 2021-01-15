// -----------------------------------------------------------------------------
// <copyright file="MacAddressRange.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.MacInventoryManager
{
    using System;
    using System.Globalization;
    using System.Text;
    using Newtonsoft.Json;

    public sealed class MacAddressRange
    {
        /// <summary>
        /// Start of the mac address range.
        /// </summary>
        public string RangeStart { get; set; }

        /// <summary>
        /// End of the mac address range.
        /// </summary>
        public string RangeEnd { get; set; }

        /// <summary>
        /// Datetime of last update.
        /// </summary>
        public DateTime LastUpdated { get; set; }

        /// <summary>
        /// If set the range will not be used.
        /// </summary>
        public string BlockedReason { get; set; }

        /// <summary>
        /// Internal id for cosmosdb serialization.
        /// </summary>
        [JsonProperty("id")]
        internal string Id => RangeStart + "-" + RangeEnd;

        /// <summary>
        /// Whether the range has been flattened to the store or not.
        /// </summary>
        internal bool IsFlattened { get; set; }

        [JsonIgnore]
        internal bool IsBlocked
        {
            get
            {
                return !String.IsNullOrEmpty(BlockedReason);
            }
        }

        internal (long, long) ConvertRanges()
        {
            if (!long.TryParse(RangeStart, NumberStyles.HexNumber, provider: null, out long start))
            {
                throw new ArgumentException($"{RangeStart} is not a valid mac address", nameof(RangeStart));
            }

            if (!long.TryParse(RangeEnd, NumberStyles.HexNumber, provider: null, out long end))
            {
                throw new ArgumentException($"{RangeEnd} is not a valid mac address", nameof(RangeEnd));
            }

            return (start, end);
        }

        internal void Validate()
        {
            if (string.IsNullOrEmpty(RangeStart))
            {
                throw new ArgumentNullException(nameof(RangeStart));
            }

            if (RangeStart.Length != 12)
            {
                throw new ArgumentException($"{RangeStart} is not a valid mac address", nameof(RangeStart));
            }

            if (string.IsNullOrEmpty(RangeEnd))
            {
                throw new ArgumentNullException(nameof(RangeEnd));
            }

            if (RangeEnd.Length != 12)
            {
                throw new ArgumentException($"{RangeEnd} is not a valid mac address", nameof(RangeEnd));
            }

            (long start, long end) = ConvertRanges();

            if (start > end)
            {
                throw new ArgumentException($"{nameof(RangeStart)}:{RangeStart} must be <= {nameof(RangeEnd)}:{RangeEnd}");
            }

            if (LastUpdated != DateTime.MinValue)
            {
                throw new ArgumentException($"{nameof(LastUpdated)} cannot be set.", nameof(LastUpdated));
            }

            if (IsFlattened)
            {
                throw new ArgumentException($"{nameof(IsFlattened)} cannot be set to true.", nameof(IsFlattened));
            }

            RangeStart = RangeStart.ToUpperInvariant();
            RangeEnd = RangeEnd.ToUpperInvariant();
        }

        public override string ToString()
        {
            StringBuilder builder = new StringBuilder();
            builder.Append($"RangeStart={RangeStart},");
            builder.Append($"RangeEnd={RangeEnd},");
            builder.Append($"BlockedReason={BlockedReason}");
            return builder.ToString();
        }
    }
}
