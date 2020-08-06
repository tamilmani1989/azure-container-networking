//--------------------------------------------------------------------------
// <copyright file="Runner.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
//   Runner periodically pings dnc endpoint to trigger tests and retrieves results.
// </summary>
//--------------------------------------------------------------------------

namespace SwiftRunner
{
    using System;
    using System.IO;
    using System.Net;
    using System.Net.Http;
    using System.Threading;
    using System.Threading.Tasks;
    using Microsoft.Cis.Monitoring.Runners;
    using Microsoft.Cloud.InstrumentationFramework;
    using Newtonsoft.Json;

    /// <summary>
    /// Runner inherits from RunnerAuthorBase.
    /// </summary>
    internal class Runner : RunnerAuthorBase
    {
        private const string MDMAccountName = "MDMAccountName";
        private const string Namespace = "Namespace";
        private const string K8sEPKey = "K8sEP";
        private const string PartitionKey = "PartitionKey";
        private const string DatapathStatusURLFmt = "http://swiftrunner.{0}.cloudapp.azure.com/swiftRunner";
        private string k8sEP;
        private string partitionKey;
        private string region;
        private ErrorContext mdmErr;
        private MeasureMetric4D ServiceTunnelMetric;
        private MeasureMetric4D Nc2VmMetric;
        private MeasureMetric4D Nc2NcMetric;
        private MeasureMetric4D Vnet2VnetGWMetric;
        private MeasureMetric4D Vnet2VnetPeeringMetric;
        private MeasureMetric4D NonRFCVnet2VnetPeeringMetric;
        private MeasureMetric4D NsgMetric;
        private MeasureMetric4D NsgwTagsMetric;
        private MeasureMetric4D UdrMetric;
        private MeasureMetric4D PrivateLinkMetric;
        private MeasureMetric4D ExternalDNSMetric;
        private MeasureMetric4D PrivateDNSMetric;
        private MeasureMetric4D MNATMetric;
        private MeasureMetric4D LBMetric;

        public Runner(string[] args)
            : base(args)
        {
            mdmErr = new ErrorContext();
            Nc2NcMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "Nc2Nc",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            ServiceTunnelMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "ServiceTunnel",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            Nc2VmMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "Nc2VM",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            Vnet2VnetGWMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "Vnet2VnetGW",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            Vnet2VnetPeeringMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "Vnet2VnetPeering",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            NonRFCVnet2VnetPeeringMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "NonRFCVnet2VnetPeering",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            NsgMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "NSG",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            NsgwTagsMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "NSGwTags",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            UdrMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "UDR",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            PrivateLinkMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "PrivateLink",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            ExternalDNSMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "ExternalDNS",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            PrivateDNSMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "PrivateDNS",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            MNATMetric = MeasureMetric4D.Create(
                this.GetConfigurationValue(MDMAccountName),
                this.GetConfigurationValue(Namespace),
                "MNAT",
                "Environment",
                "RunnerInstance",
                "DncEndPoint",
                "PartitionKey",
                ref mdmErr);

            LBMetric = MeasureMetric4D.Create(
                this.GetConfigurationValue(MDMAccountName),
                this.GetConfigurationValue(Namespace),
                "LB",
                "Environment",
                "RunnerInstance",
                "DncEndPoint",
                "PartitionKey",
                ref mdmErr);
        }

        public override void StartRunner()
        {
            this.WriteRunnerInfo();
            this.ReadConfigurationValues();

            if (!ValidMetrics())
            {
                Console.Error.WriteLine("Exit 0: Failed to create metric(s).");
                return;
            }

            // We trigger e2e tests every half an hour
            for (;;)
            {
                SwiftRunner.contracts.DncDataPathResponse datapath = null;

                try
                {
                    datapath = GetDataPathStatus();
                }
                catch (Exception e)
                {
                    Console.Error.WriteLine("Logging exception from GetNetworkPerformance: {0}", e.Message);
                    this.LogDegraded(e.ToString());
                }
                finally
                {
                    if (datapath != null)
                    {
                        this.LogHealthy("SwiftDatapathRunner is healthy.");

                        if (datapath.LastError != "") {
                            this.LogDegraded("Logging error from tests: {0}", datapath.LastError);
                        }
                        
                        if (!Nc2NcMetric.LogValue(datapath.Nc2Nc ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit Nc2Nc metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!ServiceTunnelMetric.LogValue(datapath.ServiceTunnel ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit ServiceTunnel metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!Nc2VmMetric.LogValue(datapath.Nc2VM ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit Nc2VM metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!Vnet2VnetGWMetric.LogValue(datapath.Vnet2VnetGW ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit Vnet2VnetGW metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!Vnet2VnetPeeringMetric.LogValue(datapath.Vnet2VnetPeering ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit Vnet2VnetPeering metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!NonRFCVnet2VnetPeeringMetric.LogValue(datapath.NonRFCVnet2VnetPeering ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit NonRFCVnet2VnetPeering metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!NsgMetric.LogValue(datapath.NSG ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit NSG metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!NsgwTagsMetric.LogValue(datapath.NSGwTags ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit NSGwTags metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!UdrMetric.LogValue(datapath.UDR ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit UDR metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!PrivateLinkMetric.LogValue(datapath.PrivateLink ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit PrivateLink metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!ExternalDNSMetric.LogValue(datapath.ExternalDNS ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit ExternalDNS metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!PrivateDNSMetric.LogValue(datapath.PrivateDNS ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit PrivateDNS metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!MNATMetric.LogValue(datapath.MNAT ? 1 : 2,
                            this.Environment,
                            this.RunnerInstance,
                            k8sEP,
                            partitionKey,
                            ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit MNAT metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!LBMetric.LogValue(datapath.LB ? 1 : 2,
                            this.Environment,
                            this.RunnerInstance,
                            k8sEP,
                            partitionKey,
                            ref mdmErr))
                        {
                            Console.Error.WriteLine("Failed to emit LB metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }
                    }
                    else
                    {
                        this.LogUnhealthy("Connection to cluster failed.");
                    }

                    // update to 30 min after testing...
                    Thread.Sleep(TimeSpan.FromMinutes(15));
                }
            }
        }

        private contracts.DncDataPathResponse GetDataPathStatus()
        {
            if (!string.IsNullOrEmpty(k8sEP))
            {
                string triggerStatusUrl = string.Format(
                    DatapathStatusURLFmt,
                    region);
                HttpClient httpClient = new HttpClient();
                httpClient.Timeout = new TimeSpan(0, 4, 0);
                HttpResponseMessage response;
                string responseBody;
                // Send a post request to trigger datapath tests
                try
                {
                    Console.WriteLine("Triggering swift datapath tests...");
                    response = Task.Run<HttpResponseMessage>(async () => await httpClient.PostAsync(triggerStatusUrl, null)).Result;
                    Console.WriteLine("Triggered swift datapath tests.");
                    switch (response.StatusCode)
                    {
                        case (HttpStatusCode)202:
                        case HttpStatusCode.OK:
                            HttpResponseMessage getResponse;
                            Console.WriteLine("Retrieving swift datapath results...");
                            do {
                                Thread.Sleep(5000);
                                getResponse = Task.Run<HttpResponseMessage>(async () => await httpClient.GetAsync(triggerStatusUrl)).Result;
                            } while (getResponse.StatusCode == (HttpStatusCode)202);
                            Console.WriteLine("Retrieved swift datapath results.");
                            if (getResponse.StatusCode == HttpStatusCode.OK)
                            {
                                responseBody = Task.Run<string>(async () => await getResponse.Content.ReadAsStringAsync()).Result;
                                contracts.DncDataPathResponse getDataPathStatusResponse = JsonConvert.DeserializeObject<contracts.DncDataPathResponse>(
                                    responseBody);
                                Console.WriteLine("Swift datapath results. {0}", getDataPathStatusResponse.ToString());
                                return new contracts.DncDataPathResponse(getDataPathStatusResponse);
                            }
                            break;
                        default:
                            Console.WriteLine("Failed to trigger/complete swift datapath validation.");
                            break;
                    }
                }
                catch (WebException e)
                {
                    if (e.Response != null)
                    {
                        string responseContent = new StreamReader(e.Response.GetResponseStream()).ReadToEnd();
                        Console.WriteLine("WebException: {0}", responseContent);
                        throw new Exception($"Response :{responseContent}", e);
                    }
                }
                catch (AggregateException ae)
                {
                    foreach (Exception aex in ae.InnerExceptions)
                    {
                        Console.WriteLine("AggregateException: {0}", aex.Message);
                    }
                    throw new Exception($"Response :{ae.Message}");
                }
            }
            
            return null;
        }

        private void WriteRunnerInfo()
        {
            Console.WriteLine("RunnerName = {0} RunnerInstance = {1} Environment = {2}",
                this.RunnerName,
                this.RunnerInstance,
                this.Environment);
        }

        private void ReadConfigurationValues()
        {
            k8sEP = this.GetConfigurationValue(K8sEPKey);
            partitionKey = this.GetConfigurationValue(PartitionKey);
            string[] words = partitionKey.Split('-');
            region = words[0];
            Console.WriteLine("Configurations read: K8sEP = {0} PartitionKey = {1}",
                k8sEP,
                partitionKey);
        }

        private bool ValidMetrics()
        {
            return Nc2NcMetric != null &&
                Nc2VmMetric != null && 
                ServiceTunnelMetric != null &&
                Vnet2VnetGWMetric != null &&
                Vnet2VnetPeeringMetric != null &&
                NonRFCVnet2VnetPeeringMetric != null &&
                NsgMetric != null &&
                NsgwTagsMetric != null &&
                UdrMetric != null &&
                PrivateLinkMetric != null &&
                ExternalDNSMetric != null &&
                PrivateDNSMetric != null &&
                MNATMetric != null &&
                LBMetric != null;
        }
    }
}