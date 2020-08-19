// -----------------------------------------------------------------------------
// <copyright file="Retry.cs" company="Microsoft">
//   Copyright (c) Microsoft Corporation.  All rights reserved.
// </copyright>
//
// -----------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.CoreLib
{
    using System;
    using System.Threading;

    public static class Retry
    {
        /// <summary>
        /// Execute the specified action with retries.
        /// </summary>
        /// <param name="maxTryCount">Maximum number of tries to perform.</param>
        /// <param name="failureDelay">Delay before retries.</param>
        /// <param name="action">Action to execute.</param>
        /// <param name="shouldRetry">Action to determine if retry can be done.</param>
        public static void WithRetryCount(
            int maxTryCount,
            TimeSpan failureDelay,
            Action action,
            Func<Exception, bool> shouldRetry = null)
        {
            Exception lastException = null;

            if (maxTryCount < 1)
            {
                throw new ArgumentOutOfRangeException(nameof(maxTryCount), maxTryCount, "Value must be at least 1");
            }

            for (int i = 0; i < maxTryCount; i++)
            {
                if (i > 0)
                {
                    Thread.Sleep(failureDelay);
                }

                try
                {
                    action();
                    return;
                }
                catch (Exception ex)
                {
                    lastException = ex;
                    if (shouldRetry != null && !shouldRetry(ex))
                    {
                        throw;
                    }
                }
            }

            throw new Exception($"Operation failed after {maxTryCount} retries. {lastException}");
        }

        /// <summary>
        /// Execute the specified action with retries.
        /// </summary>
        /// <param name="maxTryCount">Maximum number of tries to perform.</param>
        /// <param name="action">Action to execute.</param>
        /// <param name="shouldRetry"> Action to determine if retry can be done. </param>
        public static void WithRetryCount(int maxTryCount, Action action, Func<Exception, bool> shouldRetry = null)
        {
            WithRetryCount(maxTryCount, TimeSpan.FromSeconds(5), action, shouldRetry);
        }

        /// <summary>
        /// Execute the specified action with retries.
        /// </summary>
        /// <param name="maxTryCount">Maximum number of tries to perform.</param>
        /// <param name="failureDelay">Delay before retries.</param>
        /// <param name="action">Action to execute.</param>
        /// <param name="shouldRetry"> Action to determine if retry can be done. </param>
        public static T WithRetryCount<T>(
            int maxTryCount,
            TimeSpan failureDelay,
            Func<T> action,
            Func<Exception, bool> shouldRetry = null)
        {
            T returnValue = default;
            WithRetryCount(maxTryCount, failureDelay, () => { returnValue = action(); }, shouldRetry);
            return returnValue;
        }

        /// <summary>
        /// Execute the specified action with retries.
        /// </summary>
        /// <param name="maxTryCount">Maximum number of tries to perform.</param>
        /// <param name="failureDelay">Delay before retries.</param>
        /// <param name="action">Action to execute.</param>
        /// <param name="result"> retunred result</param>
        /// /// <param name="exception">exception if occured any</param>
        /// <param name="shouldRetry"> Action to determine if retry can be done. </param>
        public static bool WithRetryCountNoThrow<T>(
            int maxTryCount,
            TimeSpan failureDelay,
            Func<T> action,
            out T result,
            out Exception exception,
            Func<Exception, bool> shouldRetry = null)
        {
            result = default;
            exception = null;

            try
            {
                T resultValue = default;
                WithRetryCount(maxTryCount, failureDelay, () => { resultValue = action(); }, shouldRetry);
                result = resultValue;

                return true;
            }
            catch (Exception e)
            {
                exception = e;
            }

            return false;
        }

        /// <summary>
        /// Execute the specified action with retries.
        /// </summary>
        /// <param name="maxTryCount"> Maximum number of tries to perform. </param>
        /// <param name="action"> Action to execute. </param>
        /// <param name="shouldRetry"> Action to determine if retry can be done.  </param>
        /// <returns> The <see cref="T"/>. result of action  </returns>
        public static T WithRetryCount<T>(int maxTryCount, Func<T> action, Func<Exception, bool> shouldRetry = null)
        {
            return WithRetryCount(maxTryCount, TimeSpan.FromSeconds(5), action, shouldRetry);
        }
    }
}
