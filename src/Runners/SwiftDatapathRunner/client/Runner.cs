//--------------------------------------------------------------------------
// <copyright file="DNCService.cs" company="Microsoft">
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
        private MeasureMetric4D nic2nicMetric;
        private MeasureMetric4D serviceTunnelMetric;
        private MeasureMetric4D nic2VmMetric;
        private MeasureMetric4D vnet2VnetGWMetric;
        private MeasureMetric4D vnet2VnetPeeringMetric;


        public Runner(string[] args)
            : base(args)
        {
            mdmErr = new ErrorContext();
            nic2nicMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "Nic2Nic",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            serviceTunnelMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "ServiceTunnel",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            nic2VmMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "Nic2VM",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            vnet2VnetGWMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "Vnet2VnetGW",
                            "Environment",
                            "RunnerInstance",
                            "DncEndPoint",
                            "PartitionKey",
                            ref mdmErr);
            vnet2VnetPeeringMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "Vnet2VnetPeering",
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

            if (nic2nicMetric == null || 
                nic2VmMetric == null || 
                serviceTunnelMetric == null || 
                vnet2VnetGWMetric == null || 
                vnet2VnetPeeringMetric == null)
            {
                Console.WriteLine("Exit 0: Failed to create metric(s).");
                return;
            }

            // We trigger datapath tests every half an hour
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
                        Console.Error.WriteLine("Logging error from tests: {0}", datapath.LastError);
                        if (!nic2nicMetric.LogValue(datapath.Nic2Nic ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit Nic2Nic metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!serviceTunnelMetric.LogValue(datapath.ServiceTunnel ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit ServiceTunnel metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!nic2VmMetric.LogValue(datapath.Nic2VM ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit Nic2VM metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!vnet2VnetGWMetric.LogValue(datapath.Vnet2VnetGW ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit Vnet2VnetGW metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!vnet2VnetPeeringMetric.LogValue(datapath.Vnet2VnetPeering ? 1 : 2,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                partitionKey,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit Vnet2VnetPeering metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
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
    }
}