// -----------------------------------------------------------------------------
// <copyright file="Retry.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Diagnostics;
    using System.Threading.Tasks;

    /// <summary>
    /// Class which provides common retry logic.
    /// </summary>
    public static class AsyncRetry
    {
        /// <summary>
        /// Retries the given action until success or maximum retries are attempted.
        /// </summary>
        /// <param name="action">The action to attempt.</param>
        /// <param name="retryCount">Maximum attempts to perform the action.</param>
        /// <param name="retryInterval">How long to wait between attempts.</param>
        /// <param name="retryTimeout">The maximum retry execution time.</param>
        /// <param name="isRetryable">Specifies whether to retry on exception.</param>
        /// <param name="errorAction">How to handle errors.</param>
        public static Task Retry(
            Func<Task> action,
            int retryCount,
            TimeSpan retryInterval,
            TimeSpan? retryTimeout = null,
            Func<Exception, bool> isRetryable = null,
            Action<Exception> errorAction = null)
        {
            return AsyncRetry.Retry<bool>(
                action: async () =>
                {
                    await action().ConfigureAwait(continueOnCapturedContext: false);
                    return true;
                },
                retryCount: retryCount,
                retryInterval: retryInterval,
                retryTimeout: retryTimeout,
                isRetryable: isRetryable,
                errorAction: errorAction);
        }

        /// <summary>
        /// Retries the given action until success or maximum retries are attempted.
        /// </summary>
        /// <typeparam name="T">The type returned from the action.</typeparam>
        /// <param name="action">The action to attempt.</param>
        /// <param name="retryCount">Maximum attempts to perform the action.</param>
        /// <param name="retryInterval">How long to wait between attempts.</param>
        /// <param name="retryTimeout">The maximum retry execution time.</param>
        /// <param name="isRetryable">Specifies whether to retry on exception.</param>
        /// <param name="errorAction">How to handle errors.</param>
        public static async Task<T> Retry<T>(
            Func<Task<T>> action,
            int retryCount,
            TimeSpan retryInterval,
            TimeSpan? retryTimeout = null,
            Func<Exception, bool> isRetryable = null,
            Action<Exception> errorAction = null)
        {
            var stopwatch = Stopwatch.StartNew();
            for (int attempt = 1; attempt <= retryCount; ++attempt)
            {
                try
                {
                    return await action().ConfigureAwait(continueOnCapturedContext: false);
                }
                catch (Exception exception)
                {
                    if (attempt == retryCount || (retryTimeout.HasValue && stopwatch.ElapsedMilliseconds > retryTimeout.Value.TotalMilliseconds))
                    {
                        throw;
                    }

                    if (isRetryable != null && !isRetryable(exception))
                    {
                        throw;
                    }

                    if (errorAction != null)
                    {
                        errorAction(exception);
                    }
                }

                await Task.Delay(retryInterval).ConfigureAwait(continueOnCapturedContext: false);
            }

            // Should never get here because loop will throw on last retry, but compiler wasn't satisfied.
            throw new InvalidOperationException("Exhausted retries with no success.");
        }
    }
}
