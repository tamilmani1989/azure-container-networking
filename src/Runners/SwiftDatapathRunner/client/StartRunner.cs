//--------------------------------------------------------------------------
// <copyright file="DNCService.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
// <summary>
//   Starts the SwiftRunner instance.
// </summary>
//--------------------------------------------------------------------------

namespace SwiftRunner
{
    class StartRunner
    {
        static void Main(string[] args)
        {
            // Start Runner
            Runner runnerInstance = new Runner(args);
            runnerInstance.StartRunner();
        }
    }
}
