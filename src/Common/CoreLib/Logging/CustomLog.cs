// -----------------------------------------------------------------------------
// <copyright file="CustomLog.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using Microsoft.Extensions.Logging;

    public static class CustomLog
    {
        public static void CLogInformation<T>(this ILogger<T> logger, string format, params object[] args)
        {
            string message = WriteToConsole(format, args);
            logger.LogInformation(message);
        }

        public static void CLogInformation(this ILogger logger, string format, params object[] args)
        {
            string message = WriteToConsole(format, args);
            logger.LogInformation(message);
        }

        public static void CLogWarning<T>(this ILogger<T> logger, string format, params object[] args)
        {
            string message = WriteToConsole(format, args);
            logger.LogWarning(message);
        }

        public static void CLogWarning(this ILogger logger, string format, params object[] args)
        {
            string message = WriteToConsole(format, args);
            logger.LogWarning(message);
        }

        public static void CLogError<T>(this ILogger<T> logger, string format, params object[] args)
        {
            string message = WriteToConsole(format, args);
            logger.LogError(message);
        }

        public static void CLogError(this ILogger logger, string format, params object[] args)
        {
            string message = WriteToConsole(format, args);
            logger.LogError(message);
        }

        private static string  WriteToConsole(string format, params object[] args)
        {
            string message = SafeFormat(format, args);
            Console.WriteLine(message);
            return message;
        }

        private static string SafeFormat(string format, object[] args)
        {
            if (args == null || args.Length == 0)
            {
                return format;
            }

            return string.Format(format, args);
        }
    }
}
