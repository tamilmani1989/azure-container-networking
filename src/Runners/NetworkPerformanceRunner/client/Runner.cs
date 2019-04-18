//--------------------------------------------------------------------------
// <copyright file="DNCService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
//   Runner periodically pings dnc endpoint to trigger tests and retrieves results.
// </summary>
//--------------------------------------------------------------------------

namespace NetworkPerformanceRunner
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
        private const string NetworkingTypeKey = "NetworkingType";
        private const string DatapathStatusURLFmt = "http://{0}.eastus2euap.cloudapp.azure.com/networkPerformanceRunner";
        private string k8sEP;
        private string networkingType;
        private ErrorContext mdmErr;
        private MeasureMetric4D cpuSenderMetric;
        private MeasureMetric4D cpuReceiverMetric;
        private MeasureMetric4D throughputMetric;
        private MeasureMetric4D minLatencyMetric;
        private MeasureMetric4D maxLatencyMetric;
        private MeasureMetric4D avgLatencyMetric;

        public Runner(string[] args)
            : base(args)
        {
            mdmErr = new ErrorContext();
            cpuSenderMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "CPUUsageSender",
                            "Environment",
                            "RunnerInstance",
                            "K8sEndpoint",
                            "NetworkingType",
                            ref mdmErr);
            cpuReceiverMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "CPUUsageReceiver",
                            "Environment",
                            "RunnerInstance",
                            "K8sEndpoint",
                            "NetworkingType",
                            ref mdmErr);
            throughputMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "ThroughputBitsPerSec",
                            "Environment",
                            "RunnerInstance",
                            "K8sEndpoint",
                            "NetworkingType",
                            ref mdmErr);
            minLatencyMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "MinLatencyInMicroSec",
                            "Environment",
                            "RunnerInstance",
                            "K8sEndpoint",
                            "NetworkingType",
                            ref mdmErr);
            maxLatencyMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "MaxLatencyInMicroSec",
                            "Environment",
                            "RunnerInstance",
                            "K8sEndpoint",
                            "NetworkingType",
                            ref mdmErr);
            avgLatencyMetric = MeasureMetric4D.Create(
                            this.GetConfigurationValue(MDMAccountName),
                            this.GetConfigurationValue(Namespace),
                            "AvgLatencyInMicroSec",
                            "Environment",
                            "RunnerInstance",
                            "K8sEndpoint",
                            "NetworkingType",
                            ref mdmErr);
        }

        public override void StartRunner()
        {
            this.WriteRunnerInfo();
            this.ReadConfigurationValues();

            if (cpuSenderMetric == null || cpuReceiverMetric == null || throughputMetric == null || minLatencyMetric == null || maxLatencyMetric == null || avgLatencyMetric == null)
            {
                Console.WriteLine("Exit 0: Failed to create metric(s).");
                return;
            }

            // We trigger results tests every half an hour
            for (;;)
            {
                NetworkPerformanceRunner.contracts.NetworkPerformanceResults results = null;

                try
                {
                    results = GetNetworkPerformance();
                }
                catch (Exception e)
                {
                    Console.Error.WriteLine("Logging exception from GetNetworkPerformance: {0}", e.Message);
                    this.LogDegraded(e.ToString());
                }
                finally
                {
                    if (results != null)
                    {
                        this.LogHealthy("NetworkPerformanceRunner is healthy.");
                        Console.WriteLine("Network Performance Results {0}", results.ToString());
                        if (!cpuSenderMetric.LogValue(results.CPUSender,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                networkingType,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit CPUSender metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!cpuReceiverMetric.LogValue(results.CPUReceiver,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                networkingType,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit CPUReceiver metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!throughputMetric.LogValue(results.Throughput,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                networkingType,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit Throughput metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!minLatencyMetric.LogValue(results.LatencyMin,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                networkingType,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit LatencyMin metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!maxLatencyMetric.LogValue(results.LatencyMax,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                networkingType,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit LatencyMax metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
                        }

                        if (!avgLatencyMetric.LogValue(results.LatencyAvg,
                                this.Environment,
                                this.RunnerInstance,
                                k8sEP,
                                networkingType,
                                ref mdmErr))
                        {
                            Console.WriteLine("Failed to emit LatencyAvg metric, error code is {0:X}, error: {1}", mdmErr.ErrorCode, mdmErr.ErrorMessage);
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

        private contracts.NetworkPerformanceResults GetNetworkPerformance()
        {
            if (!string.IsNullOrEmpty(k8sEP))
            {
                string triggerStatusUrl = string.Format(
                    DatapathStatusURLFmt,
                    k8sEP);
                HttpClient httpClient = new HttpClient();
                httpClient.Timeout = new TimeSpan(0, 4, 0);
                HttpResponseMessage response;
                string responseBody;
                // Send a post request to trigger results tests
                try
                {
                    Console.WriteLine("Triggering network performance tests...");
                    response = Task.Run<HttpResponseMessage>(async () => await httpClient.PostAsync(triggerStatusUrl, null)).Result;
                    Console.WriteLine("Triggered network performance tests.");
                    switch (response.StatusCode)
                    {
                        case (HttpStatusCode)202:
                        case HttpStatusCode.OK:
                            HttpResponseMessage getResponse;
                            Console.WriteLine("Retrieving network performance results...");
                            do {
                                Thread.Sleep(5000);
                                getResponse = Task.Run<HttpResponseMessage>(async () => await httpClient.GetAsync(triggerStatusUrl)).Result;
                            } while (getResponse.StatusCode == (HttpStatusCode)202);
                            Console.WriteLine("Retrieved network performance results.");
                            if (getResponse.StatusCode == HttpStatusCode.OK)
                            {
                                responseBody = Task.Run<string>(async () => await getResponse.Content.ReadAsStringAsync()).Result;
                                contracts.NetworkPerformanceResults networkPerformanceResults = JsonConvert.DeserializeObject<contracts.NetworkPerformanceResults>(
                                    responseBody);
                                Console.WriteLine("Network performance results. {0}", networkPerformanceResults.ToString());
                                return new contracts.NetworkPerformanceResults(networkPerformanceResults);
                            }
                            break;
                        default:
                            Console.WriteLine("Exit 2: Failed to get network performance results.");
                            break;
                    }
                }
                catch (WebException e)
                {
                    if (e.Response != null)
                    {
                        string responseContent = new StreamReader(e.Response.GetResponseStream()).ReadToEnd();
                        Console.WriteLine("Exit 3 {0}", responseContent);
                        throw new Exception($"Response :{responseContent}", e);
                    }
                }
                catch (AggregateException ae)
                {
                    foreach (Exception aex in ae.InnerExceptions)
                    {
                        Console.WriteLine("Exit 4 {0}", aex.Message);
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
            networkingType = this.GetConfigurationValue(NetworkingTypeKey);
            Console.WriteLine("Configurations read: k8sEP = {0} networkingType = {1}",
                k8sEP,
                networkingType);
        }
    }
}