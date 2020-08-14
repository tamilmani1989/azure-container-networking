// --------------------------------------------------------------------------------------------------------------------
// <copyright file="ServiceTests.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway.ServiceTests
{
    using System;
    using System.Collections.Generic;
    using System.IO;
    using System.Linq;
    using System.Net;
    using System.Text;
    using System.Threading;

    using Microsoft.Azure.ContainerNetworking.Gateway.TestLib;
    using Xunit;

    public class ThreadInOut
    {
        public int Id;
        public int Iterations;
        public string Error;
        public TestConfig Config;
        public TestLib Lib;
        public double Average;
    }

    public class ServiceTests
    {
        [Fact]
        public void InvokeApi_OK_Response()
        {
            TestConfig config = TestConfig.ReadFromJsonFile(Path.Combine(Directory.GetCurrentDirectory(), "TestConfigSuccess.json"));
            TestLib testLib = new TestLib();
            string error;

            string token = testLib.FetchTokenWithConfidentialClient(config);
            Assert.True(testLib.InvokeApi(config.BaseAddress, token, HttpStatusCode.OK, out error, out double timetaken));
        }

        [Fact]
        public void InvokeApi_Forbidden_Response()
        {
            // 403 (Forbidden) test - User is authenticated but is not authrorized to access the resource.
            TestConfig config = TestConfig.ReadFromJsonFile(Path.Combine(Directory.GetCurrentDirectory(), "TestConfigForbidden.json"));
            TestLib testLib = new TestLib();
            string error;

            string token = testLib.FetchTokenWithConfidentialClient(config);
            Assert.True(testLib.InvokeApi(config.BaseAddress, token, HttpStatusCode.Forbidden, out error, out double timetaken));
        }

        [Fact]
        public void InvokeApi_Unauthorized_Response()
        {
            string expiredToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiIsIng1dCI6IlNzWnNCTmhaY0YzUTlTNHRycFFCVEJ5TlJSSSIsImtpZCI6IlNzWnNCTmhaY0YzUTlTNHRycFFCVEJ5TlJSSSJ9.eyJhdWQiOiJhcGk6Ly8yYTUwMDFiMy0zMmU0LTQwYzYtYTA4Yi03ZTA2YjgwNTk1MTYiLCJpc3MiOiJodHRwczovL3N0cy53aW5kb3dzLm5ldC83MzA3MzA5NC1mMDUxLTRiZTAtYjc3OS1jZTgwMmRkM2I5MTAvIiwiaWF0IjoxNTkzOTM3MTA1LCJuYmYiOjE1OTM5MzcxMDUsImV4cCI6MTU5Mzk0MTAwNSwiYWlvIjoiRTJCZ1lPQVM4ZlVPVWx5MTUvUGFoT2VyOXUxaUJRQT0iLCJhcHBpZCI6ImFmMTNmNDc5LTMyM2ItNDZhOS05NGM3LTFlNTc2OGRlYTBjYyIsImFwcGlkYWNyIjoiMSIsImlkcCI6Imh0dHBzOi8vc3RzLndpbmRvd3MubmV0LzczMDczMDk0LWYwNTEtNGJlMC1iNzc5LWNlODAyZGQzYjkxMC8iLCJvaWQiOiI1NmI4Nzk2ZC1kZWY3LTQ4YTMtOGVlOC04MjZjOWNkZjliZDMiLCJyb2xlcyI6WyJhY2Nlc3NfZG5jX2FzX2FwcGxpY2F0aW9uIl0sInN1YiI6IjU2Yjg3OTZkLWRlZjctNDhhMy04ZWU4LTgyNmM5Y2RmOWJkMyIsInRpZCI6IjczMDczMDk0LWYwNTEtNGJlMC1iNzc5LWNlODAyZGQzYjkxMCIsInV0aSI6IlQtRlJVdHZTRVVhRVZfR1JpTXNvQUEiLCJ2ZXIiOiIxLjAifQ.eNva2nbzhrmrkDhxaTQZK8Bn9xrQsxD8M5wRTF3a0CrWQs0E29_rdkHGwaVa2QqxaI0No8eTPk7GEqmYDVJ4ylqs6yaUTjQrszHkUlEMKDFUsA906TFml92wLP6ymxhrryhpFDX7lKw6yYakFvNaCHeIJ99rqm9m8VL3DvzPJ03KS38MNDtmKqRRv45GfjPK721QlfNHL6O5RVzGuX_bD2Q5oDUqgVxwxyearhwpPv-Us4K7G_LRgZ6ifVKzqDvszW1_GlIqP8sr6sG37NY0yLToGV2n7Y8miIa06QwI8JretTHqkp0aBnHBj427zcZ_MsFKDzP9zi-ILcGyZLk3jQ";

            // 401 (Unauthorized)  test - Means user is not authenticated.
            TestConfig config = TestConfig.ReadFromJsonFile(Path.Combine(Directory.GetCurrentDirectory(), "TestConfigSuccess.json"));
            TestLib testLib = new TestLib();
            string error;

            Assert.True(testLib.InvokeApi(config.BaseAddress, expiredToken, HttpStatusCode.Unauthorized, out error, out double timetaken));
        }

        [Fact]
        public void InvokeApi_Stress_Sequential()
        {
            TestConfig config = TestConfig.ReadFromJsonFile(Path.Combine(Directory.GetCurrentDirectory(), "TestConfigSuccess.json"));
            TestLib testLib = new TestLib();
            string error;
            double totalTime = 0;

            string token = testLib.FetchTokenWithConfidentialClient(config);

            for (int i = 0; i < config.StressTestIterations; ++i)
            {
                DateTime startTime = DateTime.UtcNow;
                bool success = testLib.InvokeApi(config.BaseAddress, token, HttpStatusCode.OK, out error, out double timetaken);
                totalTime += (DateTime.UtcNow - startTime).TotalSeconds;
                Assert.True(success, error);
            }

            double avg = totalTime / config.StressTestIterations;
            File.WriteAllText(Path.Combine(Directory.GetCurrentDirectory(), "Stress_Sequential.txt"), avg.ToString());
        }

        [Fact]
        public void InvokeApi_Stress_Parallel()
        {
            StressParallel("TestConfigSuccess.json", "Stress_Parallel.txt");
        }

        // TODO: Keep commented for now. This can be uncommented and used when testing
        // TODO: token encrypted vs token unencrypted scenario.
        //[Fact]
        //public void InvokeApi_Stress_Parallel_Encrypted()
        //{
        //    StressParallel("TestConfigEncrypted.json", "Stress_Parallel_Encrypted.txt");
        //}

        private void StressParallel(string testConfigFileName, string timeOutputFileName)
        {
            TestConfig config = TestConfig.ReadFromJsonFile(Path.Combine(Directory.GetCurrentDirectory(), testConfigFileName));
            int count = config.StressTestIterations;
            int threadCount = config.StressTestThreadCount;

            if (threadCount == 0)
            {
                threadCount = 10;
            }

            // Each thread uses its own instances of http client since each thread fetches its own token
            if (threadCount > 1000)
            {
                throw new Exception("Thread count for test InvokeApi_Stress_Parallel is too high. This test doesn't use threadpool for threads.");
            }

            ThreadInOut[] threadInOuts = new ThreadInOut[threadCount];
            int iterationPerThread = count > threadCount ? (count / threadCount) : 1;
            List<Thread> threadList = new List<Thread>();

            for (int i = 0; i < threadCount; ++i)
            {
                Thread thread = new Thread(ThreadEntry);
                threadList.Add(thread);

                threadInOuts[i] = new ThreadInOut { Iterations = iterationPerThread, Config = config, Id = i, Lib = new TestLib() };
                thread.Start(threadInOuts[i]);
            }

            foreach (Thread t in threadList)
            {
                t.Join();
            }

            double average = threadInOuts.Sum(x => x.Average) / threadInOuts.Count();
            Console.WriteLine("Average time for api invoke: {0}", average);
            File.WriteAllText(Path.Combine(Directory.GetCurrentDirectory(), timeOutputFileName), average.ToString());

            ThreadInOut tio = threadInOuts.FirstOrDefault(x => !string.IsNullOrWhiteSpace(x.Error));
            if (tio != null)
            {
                int failedCount = threadInOuts.Count(x => !string.IsNullOrWhiteSpace(x.Error));
                throw new Exception($"Error seen in parallel stress. Test failed: {failedCount}/{count}, Error: {tio.Error}");
            }
        }

        private void ThreadEntry(object data)
        {
            ThreadInOut threadInOut = (ThreadInOut)data;
            TestConfig config = threadInOut.Config;
            TestLib testLib = threadInOut.Lib;
            int count = threadInOut.Iterations;
            double totalTime = 0;
            StringBuilder sb = new StringBuilder();

            try
            {
                string token = testLib.FetchTokenWithConfidentialClient(config);
                string error;
                for (int i = 0; i < count; ++i)
                {
                    try
                    {
                        bool success = testLib.InvokeApi(config.BaseAddress, token, HttpStatusCode.OK, out error, out double timetaken);
                        totalTime += timetaken;

                        Assert.True(success, error);
                    }
                    catch (Exception e)
                    {
                        sb.AppendLine(e.ToString());
                    }
                }

                threadInOut.Error = sb.ToString();
                threadInOut.Average = count > 0 ? totalTime / count : 0;
            }
            catch (Exception ex)
            {
                threadInOut.Error = ex.ToString();
            }
        }
    }
}
