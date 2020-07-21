// --------------------------------------------------------------------------------------------------------------------
// <copyright file="Program.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.GatewayClient
{
    using System;
    using System.IO;

    using Microsoft.Azure.ContainerNetworking.Gateway.TestLib;

    /// <summary>
    /// A client program to authenticate with gateway. Developers can use this program to fetch a valid
    /// AAD token and as well as invoke a service (that validates AzureAD tokens) with a valid token.    
    /// Run .\GatewayClient.exe -h to see usage details of this program
    /// </summary>
    class Program
    {
        static void Main(string[] args)
        {
            CmdArguments cmdArgs = new CmdArguments();
            if (!cmdArgs.ParseArgs(args))
            {
                return;
            }

            Console.WriteLine("--> Auth mechanism: {0}", cmdArgs.AuthMechanism.ToString());
            Console.WriteLine("--> Action requested: {0}", cmdArgs.ActionToPerform.ToString());

            TestConfig config = TestConfig.ReadFromJsonFile(Path.Combine(Directory.GetCurrentDirectory(), "testconfig.json"));
            TestLib testLib = new TestLib();
            string error;

            switch (cmdArgs.ActionToPerform)
            {
                case ClientActions.FetchToken:
                    FetchToken(testLib, config, cmdArgs.AuthMechanism);
                    break;
                case ClientActions.InvokeApi:
                    string token = FetchToken(testLib, config, cmdArgs.AuthMechanism);
                    testLib.InvokeApi(config.BaseAddress, token, System.Net.HttpStatusCode.OK, out error);
                    break;
                default:
                    break;
            }
        }

        private static string FetchToken(TestLib testLib, TestConfig config, ClientAuth authMechanism)
        {
            if (authMechanism == ClientAuth.User)
            {
                return testLib.FetchTokenWithPublicClient(config);
            }
            else if (authMechanism == ClientAuth.ClientApp)
            {
                return testLib.FetchTokenWithConfidentialClient(config);
            }

            return null;
        }
    }
}
