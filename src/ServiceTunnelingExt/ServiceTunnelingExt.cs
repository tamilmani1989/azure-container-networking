
using System;
using System.Net;
using System.Net.NetworkInformation;

namespace Microsoft.WindowsAzure.Networking.Aquarius.ServiceTunnelingExt
{
    class ServiceTunnelingExt
    {
        static void Main(string[] args)
        {
            DisplayDnsAddresses();
            DisplayDnsRegistrationForAllRegions();
            DisplayDnsRegistrationAccordingToDeploymentXml("path");
        }


        private static void DisplayDnsRegistrationAccordingToDeploymentXml(string deploymentConfigPath)
        {
            //RnmDeployment deserializedDeploymentInfo = RnmDeployment.Deserialize(deploymentConfigPath);
            
        }

        private static void DisplayDnsRegistrationForAllRegions()
        {
            Console.ForegroundColor = ConsoleColor.DarkYellow;
            Console.BackgroundColor = ConsoleColor.Black;
            Console.WriteLine("__________________________");
            Console.WriteLine("   Production EndPoints   ");
            Console.WriteLine("\n");
            Console.ResetColor();

            PrintDnsDetails("australiaeast.rnm.core.windows.net");
            PrintDnsDetails("australiasoutheast.rnm.core.windows.net");
            PrintDnsDetails("useast.rnm.core.windows.net");
            PrintDnsDetails("useast2.rnm.core.windows.net");
            PrintDnsDetails("uswest.rnm.core.windows.net");
            PrintDnsDetails("usnorth.rnm.core.windows.net");
            PrintDnsDetails("ussouth.rnm.core.windows.net");
            PrintDnsDetails("uscentral.rnm.core.windows.net");
            PrintDnsDetails("europenorth.rnm.core.windows.net");
            PrintDnsDetails("europewest.rnm.core.windows.net");
            PrintDnsDetails("asiaeast.rnm.core.windows.net");
            PrintDnsDetails("asiasoutheast.rnm.core.windows.net");
            PrintDnsDetails("japaneast.rnm.core.windows.net");
            PrintDnsDetails("japanwest.rnm.core.windows.net");
            PrintDnsDetails("brazilsouth.rnm.core.windows.net");
            PrintDnsDetails("indiasouth.rnm.core.windows.net");
            PrintDnsDetails("indiawest.rnm.core.windows.net");
            PrintDnsDetails("indiacentral.rnm.core.windows.net");
            PrintDnsDetails("germanycentral.rnm.core.cloudapi.de");
            PrintDnsDetails("germanynortheast.rnm.core.cloudapi.de");
            PrintDnsDetails("canadacentral.rnm.core.windows.net");
            PrintDnsDetails("canadaeast.rnm.core.windows.net");
            PrintDnsDetails("chinanorth.rnm.core.chinacloudapi.cn");
            PrintDnsDetails("chinaeast.rnm.core.chinacloudapi.cn");
            PrintDnsDetails("usgoveast.rnm.core.usgovcloudapi.net");
            PrintDnsDetails("usgovcentral.rnm.core.usgovcloudapi.net");
            
            Console.ForegroundColor = ConsoleColor.DarkYellow;
            Console.BackgroundColor = ConsoleColor.Black;
            Console.WriteLine("*************************");
            Console.WriteLine("     Other EndPoints    ");
            Console.WriteLine("*************************\n");
            Console.ResetColor();

            PrintDnsDetails("co1.rnm.core.azure-test.net");
            PrintDnsDetails("usnorthprd01.rnm.core.windows.net");
            PrintDnsDetails("usnorthprd02.rnm.core.windows.net");
            PrintDnsDetails("usnorthprd03.rnm.core.windows.net");
            PrintDnsDetails("ussouthprd01.rnm.core.windows.net");   
            PrintDnsDetails("ussouthprd02.rnm.core.windows.net");
            PrintDnsDetails("ussouthprd03.rnm.core.windows.net");
        }

        private static void PrintDnsDetails(string domainname)
        {
            Console.ForegroundColor = ConsoleColor.DarkGreen;
            Console.BackgroundColor = ConsoleColor.Black;
            Console.Write("{0}\n", domainname);
            Console.ResetColor();
            IPHostEntry hostEntry = null;

            try
            {
                hostEntry = Dns.GetHostEntry(domainname);
            }
            catch (Exception)
            {
                Console.ForegroundColor = ConsoleColor.Red;
                Console.BackgroundColor = ConsoleColor.Black;
                Console.WriteLine(" Cannot obtain A record.");
                Console.ResetColor();
                Console.WriteLine("Press any key to continue");
                Console.ReadLine();
            }
            if (hostEntry != null)
            {
                foreach (var y in hostEntry.AddressList)
                {
                    IPHostEntry entries = null;
                    Console.Write(" (A){0} ", y);
                    try
                    {
                        entries = Dns.GetHostEntry(y);
                    }
                    catch (Exception)
                    {
                        //throw;
                    }
                    //Console.Write("    {0}", y);
                    if (entries != null)
                    {
                        string reverseDns = Dns.GetHostEntry(y).HostName;
                        Console.Write("   \t  (PTR){0}", reverseDns);
                        //also check if this reverse DNS has correct A record
                        try
                        {
                            IPHostEntry ip = Dns.GetHostEntry(reverseDns);
                            Console.WriteLine("\t (A){0}", ip.AddressList[0].ToString());
                            if (!Equals(ip.AddressList[0], y))
                            {
                                Console.ForegroundColor = ConsoleColor.Red;
                                Console.BackgroundColor = ConsoleColor.Black;
                                Console.WriteLine("Tenant domain name does not have correct A record. It should be equal to {0}", y);
                                Console.ResetColor();
                                Console.WriteLine("Press any key to continue");
                                Console.ReadLine();
                            }
                        }
                        catch (Exception)
                        {
                            Console.ForegroundColor = ConsoleColor.Red;
                            Console.BackgroundColor = ConsoleColor.Black;
                            Console.WriteLine("Tenant domain name does not have an A record");
                            Console.ResetColor();
                            Console.WriteLine("Press any key to continue");
                            Console.ReadLine();
                        }
                    }
                    else
                    {
                        Console.Write("   \t ");
                        Console.ForegroundColor = ConsoleColor.Red;
                        Console.BackgroundColor = ConsoleColor.Black;
                        Console.WriteLine(" PTR Record not found");
                        Console.ResetColor();
                        Console.WriteLine("Press any key to continue");
                        Console.ReadLine();
                    }

                    if (entries != null)
                    {
                        foreach (var a in entries.Aliases)
                        {
                            Console.WriteLine(a);
                        }
                    }
                }
                Console.WriteLine();
            }
        }

        public static void DisplayDnsAddresses()
        {
            NetworkInterface[] adapters = NetworkInterface.GetAllNetworkInterfaces();
            foreach (NetworkInterface adapter in adapters)
            {

                IPInterfaceProperties adapterProperties = adapter.GetIPProperties();
                IPAddressCollection dnsServers = adapterProperties.DnsAddresses;
                if (dnsServers.Count > 0)
                {
                    Console.WriteLine(adapter.Description);
                    foreach (IPAddress dns in dnsServers)
                    {
                        Console.WriteLine("  DNS Servers ............................. : {0}",
                            dns.ToString());
                    }
                    Console.WriteLine();
                }
            }
        }

    }
}
