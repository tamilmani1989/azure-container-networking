// --------------------------------------------------------------------------------------------------------------------
// <copyright file="TestLib.cs" company="Microsoft">
//   © Microsoft Corporation. All rights reserved.
// </copyright>
// --------------------------------------------------------------------------------------------------------------------

namespace Microsoft.Azure.ContainerNetworking.Gateway.TestLib
{
    using System;
    using System.IdentityModel.Tokens.Jwt;
    using System.Linq;
    using System.Net;
    using System.Net.Http;
    using System.Net.Http.Headers;
    using System.Security.Cryptography.X509Certificates;
    using Microsoft.Identity.Client;
    using Microsoft.IdentityModel.Tokens;

    /// <summary>
    /// This class internally creates a httpclient per instance. In this httpclient
    /// object certain properties that are not thread safe was changed in each call
    /// to methods of this class. For example - InvokeApi changes Authorization header
    /// of httpclient per invocation. So do not use same instance of this class from
    /// multiple threads. If your test is creating multiple threads and uses TestLib inside thread,
    /// create a separate TestLib instance per thread.
    /// </summary>
    public class TestLib
    {
        private HttpClient httpClient;
        private HttpClientHandler httpClientHandler;

        public TestLib()
        {
            this.httpClientHandler = new HttpClientHandler();
            this.httpClientHandler.ServerCertificateCustomValidationCallback = (message, cert, chain, sslPolicyErrors) => true;
            this.httpClient = new HttpClient(httpClientHandler);
            var defaultRequestHeaders = this.httpClient.DefaultRequestHeaders;

            if (defaultRequestHeaders.Accept == null ||
                !defaultRequestHeaders.Accept.Any(m => m.MediaType == "application/json"))
            {
                this.httpClient.DefaultRequestHeaders.Accept.Add(new
                  MediaTypeWithQualityHeaderValue("application/json"));
            }
        }

        public string FetchTokenWithConfidentialClient(TestConfig config)
        {
            IConfidentialClientApplication app;

            if (!config.UseClientCertificate)
            {
                app = ConfidentialClientApplicationBuilder.Create(config.ClientId)
                    .WithClientSecret(config.ClientSecret)
                    .WithAuthority(new Uri(config.Authority))
                    .Build();
            }
            else
            {
                X509Certificate2 cert = new X509Certificate2(config.ClientCertificatePath);
                app = ConfidentialClientApplicationBuilder.Create(config.ClientId)
                    .WithCertificate(cert)
                    .WithAuthority(new Uri(config.Authority))
                    .Build();
            }

            string[] ResourceIds = new string[] { config.ResourceID };

            try
            {
                AuthenticationResult result =
                    config.UseClientCertificate == false
                    ? app.AcquireTokenForClient(ResourceIds).ExecuteAsync().GetAwaiter().GetResult()
                    : app.AcquireTokenForClient(ResourceIds).WithSendX5C(true).ExecuteAsync().GetAwaiter().GetResult();

                if (config.DecryptToken)
                {
                    X509Certificate2 cert = new X509Certificate2(config.DecryptTokenCertificatePath);
                    var handler = new JwtSecurityTokenHandler();
                    var claimsPrincipal = handler.ValidateToken(
                        result.AccessToken,
                        new TokenValidationParameters
                        {
                            ValidateAudience = false,
                            ValidateIssuer = false,
                            ValidateIssuerSigningKey = false,
                            ValidateLifetime = false,
                            RequireExpirationTime = false,
                            RequireSignedTokens = false,
                            SignatureValidator = delegate (string token, TokenValidationParameters parameters)
                            {
                                var jwt = new JwtSecurityToken(token);

                                return jwt;
                            },
                            TokenDecryptionKey = new X509SecurityKey(cert)
                        },
                        out SecurityToken securityToken);


                    JwtSecurityToken jwtd = securityToken as JwtSecurityToken;

                    Console.ForegroundColor = ConsoleColor.Green;
                    Console.WriteLine("Decrypted token \n");
                    Console.WriteLine(jwtd.InnerToken.RawData);
                    Console.ResetColor();
                }

                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine("Token acquired \n");
                Console.WriteLine(result.AccessToken);
                Console.ResetColor();
                return result.AccessToken;
            }
            catch (MsalClientException ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine(ex.Message);
                Console.ResetColor();
            }

            return null;
        }

        public string FetchTokenWithPublicClient(TestConfig config)
        {
            IPublicClientApplication app = PublicClientApplicationBuilder.Create(config.ClientId)
                .WithAuthority(new Uri(config.Authority))
                .WithRedirectUri("http://localhost")
                .Build();

            string[] ResourceIds = new string[] { config.ResourceID };

            try
            {
                AuthenticationResult result = app.AcquireTokenInteractive(ResourceIds).ExecuteAsync().GetAwaiter().GetResult();
                Console.ForegroundColor = ConsoleColor.Green;
                Console.WriteLine("Token acquired \n");
                Console.WriteLine(result.AccessToken);
                Console.ResetColor();
                return result.AccessToken;
            }
            catch (MsalClientException ex)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.WriteLine(ex.Message);
                Console.ResetColor();
            }

            return null;
        }

        public bool InvokeApi(string requestUri, string token, HttpStatusCode expectedStatusCode, out string error, out double timeTaken)
        {
            error = null;
            timeTaken = 0;

            if (!string.IsNullOrEmpty(token))
            {
                if (!this.httpClient.DefaultRequestHeaders.Contains("Authorization"))
                {
                    this.httpClient.DefaultRequestHeaders.Remove("Authorization");
                }

                this.httpClient.DefaultRequestHeaders.Authorization =
                  new AuthenticationHeaderValue("bearer", token);

                DateTime startTime = DateTime.UtcNow;
                HttpResponseMessage response = this.httpClient.GetAsync(requestUri).GetAwaiter().GetResult();
                timeTaken = (DateTime.UtcNow - startTime).TotalSeconds;
                if (response.IsSuccessStatusCode)
                {
                    Console.ForegroundColor = ConsoleColor.Green;
                    string json = response.Content.ReadAsStringAsync().GetAwaiter().GetResult();
                    Console.WriteLine(json);
                }
                else
                {
                    Console.ForegroundColor = ConsoleColor.Red;
                    string content = response.Content.ReadAsStringAsync().GetAwaiter().GetResult();
                    error = $"Failed to call the Web Api: {response.StatusCode}. Content: {content}";
                    Console.WriteLine(error);
                }
                Console.ResetColor();

                return response.StatusCode == expectedStatusCode;
            }

            return false;
        }
    }
}
