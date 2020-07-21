// --------------------------------------------------------------------------------------------------------------------
// <copyright file="JwtBearerMiddlewareDiagnostics.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway
{
    using System;
    using System.Threading.Tasks;

    using Microsoft.AspNetCore.Authentication.JwtBearer;
    using Microsoft.Extensions.Logging;

    /// <summary>
    /// Diagnostics for the JwtBearer middleware (used in Web APIs)
    /// </summary>
    public class JwtBearerMiddlewareDiagnostics
    {
        /// <summary>
        /// Invoked if exceptions are thrown during request processing. The exceptions will be re-thrown after this event unless suppressed.
        /// </summary>
        static Func<AuthenticationFailedContext, Task> onAuthenticationFailed;

        /// <summary>
        /// Invoked when a protocol message is first received.
        /// </summary>
        static Func<MessageReceivedContext, Task> onMessageReceived;

        /// <summary>
        /// Invoked after the security token has passed validation and a ClaimsIdentity has been generated.
        /// </summary>
        static Func<TokenValidatedContext, Task> onTokenValidated;

        /// <summary>
        /// Invoked before a challenge is sent back to the caller.
        /// </summary>
        static Func<JwtBearerChallengeContext, Task> onChallenge;

        /// <summary>
        /// Subscribes to all the JwtBearer events, to help debugging, while
        /// preserving the previous handlers (which are called)
        /// </summary>
        /// <param name="events">Events to subscribe to</param>
        public static JwtBearerEvents Subscribe(JwtBearerEvents events)
        {
            if (events == null)
            {
                events = new JwtBearerEvents();
            }            

            onAuthenticationFailed = events.OnAuthenticationFailed;
            events.OnAuthenticationFailed = OnAuthenticationFailed;

            onMessageReceived = events.OnMessageReceived;
            events.OnMessageReceived = OnMessageReceived;

            onTokenValidated = events.OnTokenValidated;
            events.OnTokenValidated = OnTokenValidated;

            onChallenge = events.OnChallenge;
            events.OnChallenge = OnChallenge;

            return events;
        }

        public static ILogger JwtEventLog
        {
            get; set;
        }

        static async Task OnMessageReceived(MessageReceivedContext context)
        {
            LogInfo("OnMessageReceived");
            // Place a breakpoint here and examine the bearer token (context.Request.Headers.HeaderAuthorization / context.Request.Headers["Authorization"])
            // Use https://jwt.ms to decode the token and observe claims
            await onMessageReceived(context);
            LogError("OnMessageReceived", context.Result != null ? context.Result.Failure : null);
        }

        static async Task OnAuthenticationFailed(AuthenticationFailedContext context)
        {
            LogInfo("OnAuthenticationFailed");
            // Place a breakpoint here and examine context.Exception
            await onAuthenticationFailed(context);
            LogError("OnAuthenticationFailed", context.Exception);
        }

        static async Task OnTokenValidated(TokenValidatedContext context)
        {
            LogInfo("OnTokenValidated");
            await onTokenValidated(context);
            LogError("OnTokenValidated", context.Result != null ? context.Result.Failure : null);
        }

        static async Task OnChallenge(JwtBearerChallengeContext context)
        {
            LogInfo("OnChallenge");
            await onChallenge(context);
            LogError("OnChallenge", context.Error);
        }

        static void LogError(string eventName, string exception)
        {
            if (!string.IsNullOrWhiteSpace(exception) && JwtEventLog != null)
            {
                JwtEventLog.LogError("JwtBearerEvent:{0}, Error:{1}", eventName, exception);
            }
        }

        static void LogError(string eventName, Exception exception)
        {
            if (exception != null && JwtEventLog != null)
            {
                JwtEventLog.LogError("JwtBearerEvent:{0}, Error:{1}", eventName, exception);
            }
        }

        static void LogInfo(string format, params object[] args)
        {
            if (JwtEventLog != null)
            {
                JwtEventLog.LogInformation(format, args);
            }
        }
    }
}
