using System.Collections.Generic;

namespace NetworkManager.DataModel
{
    class Constants
    {
        public const string SubnetConfigurationTableName = "SubnetConfigurationTable";

        public const string NodeTableName = "NodeConfigurationTable";

        public const string IPReservationsTableName = "IPReservationsTable";

        public const string PendingRequestsTableName = "PendingRequestsTable";

        public const string FailedRequestsTableName = "FailedRequestsTable";

        public const string NetworkConfigurationTableName = "NetworkConfigurationTable";

        public const string NetworkContainerTableName = "NetworkContainerTable";

        public const string TagDefinitionTableName = "TagDefinitionsTable";

        public const string NetworkProfilesTableName = "NetworkProfilesTable";

        public const string NetworkKey = "Network_{0}";

        public const string SubnetConfigurationKey = "Network_{0}_Subnet_{1}";

        public const string IPReservationKey = "Network_{0}_Reservation_{1}";

        public const string NodeKey = "Node_{0}";

        public const string NCKey = "Network_{0}_NC_{1}";

        public const string TagDefinitionKey = "Tag_{0}_{1}";

        // Azure storage table constants
        public const string IPAllocationsTableName = "IPAllocationsTable";
        public const string IPAllocationsTableColumnName = "IPAllocation";

        public static IReadOnlyDictionary<string, string> TableToColumnMap =
            new Dictionary<string, string> {
                {IPAllocationsTableName, IPAllocationsTableColumnName}
            };
    }
}
