//
//    Copyright (C) Microsoft.  All rights reserved.
//

using DNCService.API;
using System;
using System.Net;

namespace DNCService.Rest
{
    /// <summary>
    /// Encapsulates URL parameters validation logic for different kind of DNC REST requests.
    /// </summary>
    class RequestUrlParametersValidator
    {
        /// <summary>
        /// Validates Node Request URL parameters. Throws exception if any validation fails. 
        /// </summary>
        /// <param name="nodeId">Id of the node.</param>
        /// <returns></returns>
        public static void ValidateNodeRequest(string nodeId)
        {
            if (string.IsNullOrEmpty(nodeId))
            {
                throw new DncNodeManagementException(
                    "",
                    new ErrorInfo(
                        ErrorCode.InvalidNodeID,
                        "NodeID cannot be null or empty",
                        (int)HttpStatusCode.BadRequest,
                        ErrorType.NonRetriable));
            }
        }

        /// <summary>
        /// Validates Network Request URL parameters. Throws exception if any validation fails. 
        /// </summary>
        /// <param name="networkId">Resource Id of the customer virtual network.</param>
        /// <returns></returns>
        public static void ValidateNetworkRequest(string networkId)
        {
            ErrorInfo errorInfo;

            if (!ValidateNetworkId(networkId, out errorInfo))
            {
                throw new DncNodeManagementException(
                    (networkId != null) ? networkId : "",
                    errorInfo);
            }
        }

        /// <summary>
        /// Validates Network Container Request URL parameters. Throws exception if any validation fails. 
        /// </summary>
        /// <param name="networkId">Resource Id of the customer virtual network.</param>
        /// <param name="subnetName">Name of the delegated subnet.</param>
        /// <returns></returns>
        public static void ValidateSubnetRequest(string networkId, string subnetName)
        {
            ErrorInfo errorInfo;

            if (!ValidateNetworkId(networkId, out errorInfo))
            {
                throw new DncNetworkManagementException(
                    (networkId != null) ? networkId : "",
                    errorInfo);
            }

            if (string.IsNullOrEmpty(subnetName))
            {
                throw new DncNetworkManagementException(
                    networkId,
                    new ErrorInfo(
                        ErrorCode.InvalidSubnetName,
                        "SubnetName cannot be null or empty",
                        (int)HttpStatusCode.BadRequest,
                        ErrorType.NonRetriable));
            }
        }

        /// <summary>
        /// Validates Network Container Request URL parameters. Throws exception if any validation fails. 
        /// </summary>
        /// <param name="networkId">Resource Id of the customer virtual network.</param>
        /// <param name="networkContainerId">Id of the network container.</param>
        /// <returns></returns>
        public static void ValidateNetworkContainerRequest(string networkId, string networkContainerId)
        {
            ErrorInfo errorInfo;

            if (!ValidateNetworkId(networkId, out errorInfo))
            {
                throw new DncNetworkContainerException(
                    (networkId != null) ? networkId : "",
                    (networkContainerId != null) ? networkContainerId : "",
                    errorInfo);
            }

            // Make sure provided network container id is not null or empty.
            if (string.IsNullOrEmpty(networkContainerId))
            {
                throw new DncNetworkContainerException(
                    networkId,
                    "",
                    new ErrorInfo(
                        ErrorCode.InvalidNetworkContainerID,
                        "NetworkContainerID cannot be null or empty",
                        (int)HttpStatusCode.BadRequest,
                        ErrorType.NonRetriable));
            }

            // Make sure provided network container id is a guid.
            Guid containerId;
            if (!Guid.TryParse(networkContainerId, out containerId))
            {
                throw new DncNetworkContainerException(
                    networkId,
                    networkContainerId,
                    new ErrorInfo(
                        ErrorCode.InvalidNetworkContainerID,
                        "NetworkContainerID value needs to be in GUID format.",
                        (int)HttpStatusCode.BadRequest,
                        ErrorType.NonRetriable));
            }
        }

        /// <summary>
        /// Validates Reservation Request URL parameters. Throws exception if any validation fails. 
        /// </summary>
        /// <param name="networkId">Id of the network profile.</param>
        /// <param name="reservationId">Reservation Id</param>
        /// <returns></returns>
        public static void ValidateReservationRequest(string networkId, string reservationId)
        {
            ErrorInfo errorInfo;
            if (!ValidateNetworkId(networkId, out errorInfo))
            {
                throw new DncNetworkReservationException(
                    (networkId != null) ? networkId : "",
                    (reservationId != null) ? reservationId : "",
                    errorInfo);
            }

            if (string.IsNullOrEmpty(reservationId))
            {
                throw new DncNetworkReservationException(
                    networkId,
                    "",
                    new ErrorInfo(
                        ErrorCode.InvalidReservationID,
                        "ReservationID cannot be null or empty",
                        (int)HttpStatusCode.BadRequest,
                        ErrorType.NonRetriable));
            }

            // Make sure provided reservation id is a guid.
            Guid reservationGuid;
            if (!Guid.TryParse(reservationId, out reservationGuid))
            {
                throw new DncNetworkReservationException(
                    networkId,
                    reservationId,
                    new ErrorInfo(
                        ErrorCode.InvalidReservationID,
                        "ReservationID value needs to be in GUID format.",
                        (int)HttpStatusCode.BadRequest,
                        ErrorType.NonRetriable));
            }
        }

        /// <summary>
        /// Validates Network Profile Request URL parameters. Throws exception if any validation fails. 
        /// </summary>
        /// <param name="networkProfileId">Id of the network profile.</param>
        /// <returns></returns>
        public static void ValidateNetworkProfileRequest(string networkProfileId)
        {
            if (string.IsNullOrEmpty(networkProfileId))
            {
                throw new DncManagementException(
                    new ErrorInfo(
                        ErrorCode.InvalidNetworkProfileId,
                        "NetworkProfileID cannot be null or empty",
                        (int)HttpStatusCode.BadRequest,
                        ErrorType.NonRetriable));
            }
        }

        /// <summary>
        /// Validates Network Id.
        /// </summary>
        /// <param name="networkId">Id of the custmer virtual network.</param>
        /// <param name="errorInfo">Out parameters, gets set in case th validation fails</param>
        /// <returns>true, if provided network is valid, false otherwise.</returns>
        static bool ValidateNetworkId(string networkId, out ErrorInfo errorInfo)
        {
            bool isValid = true;
            errorInfo = null;

            if (string.IsNullOrEmpty(networkId))
            {
                isValid = false;
                errorInfo = new ErrorInfo(
                    ErrorCode.InvalidNetworkID,
                    "NetworkID cannot be null or empty",
                    (int)HttpStatusCode.BadRequest,
                    ErrorType.NonRetriable);
            }

            // Make sure provided network id is a guid.
            Guid networkGuid;
            if (!Guid.TryParse(networkId, out networkGuid))
            {
                isValid = false;
                errorInfo = new ErrorInfo(
                    ErrorCode.InvalidNetworkID,
                    "NetworkID value needs to be in GUID format.",
                    (int)HttpStatusCode.BadRequest,
                    ErrorType.NonRetriable);
            }

            return isValid;
        }
    }
}
