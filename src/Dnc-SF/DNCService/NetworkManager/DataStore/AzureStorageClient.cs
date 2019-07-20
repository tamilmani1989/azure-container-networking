//-----------------------------------------------------------------------
// <copyright file="AzureStorageClient.cs" company="Microsoft">
//     Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
//-----------------------------------------------------------------------

using DNCService;
using System.Collections.Generic;
using System.Threading.Tasks;
using Microsoft.WindowsAzure.Storage;
using Microsoft.WindowsAzure.Storage.Table;
using System.Threading;
using DNCService.API;

namespace NetworkManager.DataModel
{
    /// <summary>
    /// Class definition for request watcher.
    /// Monitors and processes pending requests.
    /// </summary>
    ///
    public class AzureStorageClient
    {
        /// <summary>
        /// Azure storage account name
        /// </summary>
        public string AccountName { get; private set; }

        /// <summary>
        /// Azure storage account key
        /// </summary>
        public string AccountKey { get; private set; }

        /// <summary>
        /// Flag to enable shared subnet delegation feature
        /// </summary>
        public bool EnableSharedSubnetDelegation { get; private set; }

        /// <summary>
        /// Azure table client
        /// </summary>
        private CloudTableClient tableClient;

        /// <summary>
        /// Dictionary of tables in azure storage
        /// </summary>
        private Dictionary<string, CloudTable> tables;

        /// <summary>
        /// Cached cancellation token.
        /// </summary>
        private CancellationToken cancellationToken;

        /// <summary>
        /// Constructor.
        /// </summary>
        /// <param name="dncConfiguration">DNC configuration.</param>
        public AzureStorageClient(
            DncConfiguration dncConfiguration)
        {
            AccountName = dncConfiguration.DataStoreSettings.AzureStorageInfo.AccountName;
            AccountKey = dncConfiguration.DataStoreSettings.AzureStorageInfo.Key;
            EnableSharedSubnetDelegation =
                dncConfiguration.DataStoreSettings.AzureStorageInfo.EnableSharedSubnetDelegation;
            tables = new Dictionary<string, CloudTable>();
        }

        /// <summary>
        /// Initialize.
        /// </summary>
        /// <param name="cancellationToken">Cancellation token.</param>
        public async Task Init(
            CancellationToken cancellationToken)
        {
            this.cancellationToken = cancellationToken;

            if (IsSharedSubnetDelegationEnabled())
            {
                if (string.IsNullOrEmpty(AccountName))
                {
                    throw new DncAzureStorageException(
                      new ErrorInfo(ErrorCode.InvalidAzureStorageInfo,
                          "Failed to initialize AzureStorageClient - Invalid Account Name",
                          ErrorType.NonRetriable));
                }

                if (string.IsNullOrEmpty(AccountKey))
                {
                    throw new DncAzureStorageException(
                      new ErrorInfo(ErrorCode.InvalidAzureStorageInfo,
                          "Failed to initialize AzureStorageClient - Invalid Account Key",
                          ErrorType.NonRetriable));
                }

                string connectionString = "DefaultEndpointsProtocol=https;" +
                    $"AccountName={AccountName};" +
                    $"AccountKey={AccountKey};" +
                    "EndpointSuffix=core.windows.net";

                if (CloudStorageAccount.TryParse(connectionString,
                    out CloudStorageAccount storageAccount))
                {
                    try
                    {
                        tableClient = storageAccount.CreateCloudTableClient();

                        // Populate the table references
                        foreach (KeyValuePair<string, string> tableMap in Constants.TableToColumnMap)
                        {
                            var tableReference = tableClient.GetTableReference(tableMap.Key);
                            if (!await tableReference.ExistsAsync())
                            {
                                ServiceEventSource.Current.Message($"Table {tableMap.Key} " +
                                    $"doesn't exist. Creating one.");
                                await tableReference.CreateIfNotExistsAsync();

                                if (!await tableReference.ExistsAsync())
                                {
                                    ServiceEventSource.Current.Message($"Failed to create table {tableMap.Key}");
                                    throw new DncAzureStorageException(
                                        new ErrorInfo(ErrorCode.StatusInternalServerError,
                                            $"Failed to create table {tableMap.Key}. " +
                                            $"AzureStorageClient initialization failed.",
                                            ErrorType.NonRetriable));
                                }
                            }

                            tables.Add(tableMap.Key, tableReference);
                        }

                        ServiceEventSource.Current.Message("AzureStorageClient init complete.");
                    }
                    catch (StorageException exception)
                    {
                        string errorMsg = (exception.InnerException == null) ?
                            exception.Message : exception.InnerException.Message;
                        ServiceEventSource.Current.Message($"Failed to initialize AzureStorageClient" +
                            $" due to Exception: {exception}");
                        throw new DncAzureStorageException(
                            new ErrorInfo(ErrorCode.StatusInternalServerError,
                                $"Failed to initialize AzureStorageClient due to error: {errorMsg}",
                                ErrorType.NonRetriable));
                    }
                }
                else
                {
                    ServiceEventSource.Current.Message("Failed to parse Azure Storage Account details.");
                    throw new DncAzureStorageException(
                        new ErrorInfo(ErrorCode.InvalidAuthenticationToken,
                            "Failed to parse Azure Storage Account details.",
                            ErrorType.NonRetriable));
                }
            }
        }

        /// <summary>
        /// Get a dictionary of allocated subnets / IP addresses for the
        /// given subnet key which is the partition key in the table.
        /// Dictionary key is allocated IP and value is reservation key
        /// that the allocation belongs to.
        /// </summary>
        /// <param name="partitionKey">Partition Key for the table.</param>
        /// <returns>Dictionary of allocated subnets</returns>
        public async Task<IReadOnlyDictionary<string, string>> GetAllocatedSubnets(
            string partitionKey)
        {
            Dictionary<string, string> allocatedSubnets =
                new Dictionary<string, string>();
            try
            {
                TableQuery tableQuery = new TableQuery().Where(
                    TableQuery.GenerateFilterCondition(
                        "PartitionKey", QueryComparisons.Equal, partitionKey));
                TableContinuationToken continuationToken = null;

                do
                {
                    TableQuerySegment dynamicTableEntities =
                        await tables[Constants.IPAllocationsTableName].ExecuteQuerySegmentedAsync(
                            tableQuery,
                            continuationToken,
                            null,
                            null,
                            cancellationToken);

                    var enumerator = dynamicTableEntities.GetEnumerator();
                    while (enumerator.MoveNext())
                    {
                        if (!enumerator.Current.Properties.TryGetValue(
                            Constants.TableToColumnMap[Constants.IPAllocationsTableName],
                            out EntityProperty reservationKey))
                        {
                            ServiceEventSource.Current.Message($"Failed to read from table " +
                                $"{Constants.IPAllocationsTableName}");
                            throw new DncAzureStorageException(
                                new ErrorInfo(ErrorCode.StatusInternalServerError,
                                    $"Failed to read from table {Constants.IPAllocationsTableName}",
                                    ErrorType.Retriable));
                        }

                        allocatedSubnets.Add(enumerator.Current.RowKey, reservationKey.StringValue);
                    }
                } while (continuationToken != null);
            }
            catch (StorageException exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                ServiceEventSource.Current.Message($"Failed to retrieve allocated subnets" +
                    $" from IP allocations table due to {errorMsg}");
                throw new DncAzureStorageException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to read the exisiting allocated subnets",
                        ErrorType.Retriable));
            }

            return allocatedSubnets;
        }

        /// <summary>
        /// Try to insert the allocated IP into the table. The operation
        /// mat fail due to IP contention.
        /// </summary>
        /// <param name="partitionKey">Partition Key for the table.</param>
        /// <param name="allocatedIP">Allocated IP address to insert in the table.</param>
        /// <param name="reservationKey">Reservation Key to insert in the table.</param>
        /// <returns>Returns true if operation is successful, false otherwise</returns>
        public async Task<bool> TryInsertAllocatedIP(
            string partitionKey,
            string allocatedIP,
            string reservationKey)
        {
            try
            {
                DynamicTableEntity tableEntity = new DynamicTableEntity(
                    partitionKey, allocatedIP);
                EntityProperty entityProperty = new EntityProperty(reservationKey);
                tableEntity.Properties.Add(
                    Constants.TableToColumnMap[Constants.IPAllocationsTableName], entityProperty);
                TableOperation insertOperation = TableOperation.Insert(tableEntity);
                await tables[Constants.IPAllocationsTableName].ExecuteAsync(
                    insertOperation,
                    null,
                    null,
                    cancellationToken);

                return true;
            }
            catch (StorageException exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                ServiceEventSource.Current.Message($"Failed to insert allocated IP " +
                    $"into the table due to error: {errorMsg}");

                return false;
            }
        }

        /// <summary>
        /// Delete a row from IPAllocations table
        /// </summary>
        /// <param name="partitionKey">Partition Key for the table.</param>
        /// <param name="allocatedIP">Allocated IP address to search in the table.</param>
        /// <returns>None</returns>
        private async Task DeleteIPAllocationRecord(
            string partitionKey,
            string allocatedIP)
        {
            try
            {
                DynamicTableEntity tableEntity = new DynamicTableEntity(
                    partitionKey, allocatedIP);
                tableEntity.ETag = "*";
                TableOperation deleteOperation =
                    TableOperation.Delete(tableEntity);
                await tables[Constants.IPAllocationsTableName].ExecuteAsync(deleteOperation);
            }
            catch (StorageException exception)
            {
                string errorMsg = (exception.InnerException == null) ?
                    exception.Message : exception.InnerException.Message;
                ServiceEventSource.Current.Message($"Failed to delete allocated IP " +
                    $"into the table due to error: {errorMsg}");
            }
        }

        /// <summary>
        /// Delete IP allocation from IPAllocations table
        /// </summary>
        /// <param name="partitionKey">Partition Key for the table.</param>
        /// <param name="allocatedIP">Allocated IP address to delete from the table.</param>
        /// <param name="reservationKey">Reservation Key to validate the queried entry.</param>
        /// <returns>None</returns>
        public async Task DeleteAllocatedIP(
            string partitionKey,
            string allocatedIP,
            string reservationKey)
        {
            string savedReservationKey =
                await RetrieveReservationKeyForAllocatedIP(partitionKey, allocatedIP);

            if (!string.IsNullOrWhiteSpace(savedReservationKey))
            {
                if (reservationKey.Equals(savedReservationKey))
                {
                    await DeleteIPAllocationRecord(partitionKey, allocatedIP);
                }
                else
                {
                    ServiceEventSource.Current.Message($"Reservation key mismatch in " +
                        $"the table for allocated IP {allocatedIP}");
                }
            }
            else
            {
                ServiceEventSource.Current.Message($"Failed to find the allocated IP " +
                    $"{allocatedIP} in the table.");
            }
        }

        /// <summary>
        /// Retrieve the reservation key for the allocated IP from
        /// IPAllocations table.
        /// </summary>
        /// <param name="partitionKey">Partition Key for the table.</param>
        /// <param name="allocatedIP">Allocated IP address to search in the table.</param>
        /// <returns>Returns key if found, null otherwise</returns>
        private async Task<string> RetrieveReservationKeyForAllocatedIP(
            string partitionKey,
            string allocatedIP)
        {
            try
            {
                TableOperation retrieveOperation = TableOperation.Retrieve(partitionKey, allocatedIP);
                TableResult tableResult =
                    await tables[Constants.IPAllocationsTableName].ExecuteAsync(retrieveOperation);

                if (tableResult.Result != null)
                {
                    if (!((DynamicTableEntity)tableResult.Result).Properties.TryGetValue(
                        Constants.TableToColumnMap[Constants.IPAllocationsTableName],
                        out EntityProperty reservationKey))
                    {
                        ServiceEventSource.Current.Message($"Failed to extract ReservationKey from " +
                            "the retrieved table record.");
                        return null;
                    }

                    return reservationKey.StringValue;
                }
                else
                {
                    ServiceEventSource.Current.Message($"ReservationKey not found");
                    return null;
                }
            }
            catch (StorageException)
            {
                throw new DncAzureStorageException(
                    new ErrorInfo(ErrorCode.StatusInternalServerError,
                        $"Failed to retrieve ReservationKey for allocated IP",
                        ErrorType.Retriable));
            }
        }

        /// <summary>
        /// Returns if the shared subnet delegation is enabled
        /// </summary>
        /// <returns>True if enabled, false otherwise.</returns>
        public bool IsSharedSubnetDelegationEnabled()
        {
            return EnableSharedSubnetDelegation;
        }

        /// <summary>
        /// Encode the row key for IPAllocation table
        /// </summary>
        /// <param name="rowKey">Row key to encode.</param>
        /// <returns>Encoded string</returns>
        public string EncodeIPAllocationKey(
            string rowKey)
        {
            return rowKey.Replace('/', '_');
        }

        /// <summary>
        /// Decode the row key from IPAllocation table
        /// </summary>
        /// <param name="rowKey">Row key to decode.</param>
        /// <returns>Decoded string</returns>
        public string DecodeIPAllocationKey(
            string rowKey)
        {
            return rowKey.Replace('_', '/');
        }
    }
}
