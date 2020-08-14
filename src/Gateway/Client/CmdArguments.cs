// --------------------------------------------------------------------------------------------------------------------
// <copyright file="CmdArguments.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.GatewayClient
{
    using System;
    using System.Linq;
    using Microsoft.Extensions.CommandLineUtils;

    internal enum ClientAuth
    {
        Unknown = 0,
        User,
        ClientApp
    };

    internal enum ClientActions
    {
        Unknown = 0,
        FetchToken,
        InvokeApi
    };


    internal class CmdArguments
    {
        public ClientAuth AuthMechanism
        {
            get; private set;
        }

        public ClientActions ActionToPerform
        {
            get; private set;
        }

        public bool DebugMode
        {
            get; private set;
        }

        public bool ParseArgs(string[] args)
        {
            int failed = -1;
            int passed = 0;
            bool printHelp = true;

            CommandLineApplication cmdLineApp = new CommandLineApplication(true);
            cmdLineApp.Description = "GatewayClient help";
            cmdLineApp.Name = "GatewayClient";
            CommandOption helpOption = cmdLineApp.HelpOption("-?|-h|--help");

            CommandOption authOption = cmdLineApp.Option(
                "-au|--auth",
                "Authentication mechanism for AzureAD. Valid auth values are user, clientapp . 1) User authentication 2) Client app id/credentials (from appsettings.json)",
                CommandOptionType.SingleValue);

            CommandOption actionOption = cmdLineApp.Option(
              "-a|--action",
              "Action to perform. Valid actions are fetchtoken, invokeapi. 1) fetchtoken - will return token. +" +
              " 2) invokeapi - will fetch token first using specified method (-u or -c) and then invoke BaseUrl in appsettings.json",
              CommandOptionType.SingleValue);

            CommandOption debugOption = cmdLineApp.Option(
             "-d|--debug",
             "Deubg mode. Wait to attach a debugger to the process",
             CommandOptionType.NoValue);

            cmdLineApp.OnExecute(() =>
            {
                if (helpOption.HasValue())
                {
                    cmdLineApp.ShowHelp();
                    printHelp = false;
                    return failed;
                }

                if (debugOption.HasValue())
                {
                    this.DebugMode = true;
                }

                if (authOption.HasValue())
                {
                    ClientAuth returnVal;
                    string authValue = authOption.Values.First();
                    if (!Enum.TryParse<ClientAuth>(authValue, true, out returnVal))
                    {
                        Console.WriteLine("Invalid authentication mechanism : {0}", authValue);
                        return failed;
                    }

                    this.AuthMechanism = returnVal;
                }

                if (actionOption.HasValue())
                {
                    ClientActions returnVal;
                    string actionValue = actionOption.Values.First();
                    if (!Enum.TryParse<ClientActions>(actionValue, true, out returnVal))
                    {
                        Console.WriteLine("Invalid action : {0}", actionValue);
                        return failed;
                    }

                    this.ActionToPerform = returnVal;
                }

                return passed;
            });

            int execResult = cmdLineApp.Execute(args);
            if (execResult == failed && printHelp)
            {
                cmdLineApp.ShowHelp();
            }

            return execResult == passed;
        }
    }
}
