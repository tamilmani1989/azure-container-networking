/****************************** Module Header ******************************\
* Module Name:  ParserInterfaces.c
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Parses the interfaces description returned by wireserver
*
\***************************************************************************/

#include <string.h>
#include <libxml/parser.h>
#include <arpa/inet.h>
namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
unsigned int ConvertToIPAddress(unsigned char *ipAddressinStringFormat)
{
    struct in_addr ipaddr;
    int retval;
    retval = inet_pton(AF_INET, (char *)ipAddressinStringFormat, &ipaddr);
    if(retval == 1)
    {
        return ipaddr.s_addr;
    }
    else
    {
        fprintf(stderr, "Uable to convert to %s from string to ip adress.", ipAddressinStringFormat);
    }
    return 0;
}

void getSubnetIp(xmlNode *interfaceNode, char **subnetIp)
{
    xmlNode *node = NULL;
	char *temp=NULL;
    for (node = interfaceNode->children; node; node = node->next) 
    {
        if (node->type == XML_ELEMENT_NODE) 
        {
            if(strncmp((char *)node->name, "IPSubnet", 8) == 0)
			{
				struct _xmlAttr * props = node->properties;
				for(props = node->properties; props; props=props->next)
                {
  
                    if(strncmp((char *)props->name, "Prefix", 6)==0)
                    {
						xmlChar* value = xmlNodeListGetString(node->doc, props->children, 1);
						int len  = strlen((char *)value) + sizeof(char);
						temp = (char *) malloc(len);
						strncpy(temp, (char *)value, len);
						*subnetIp = temp;
                        xmlFree(value);
						break;
					}
				}
			}
		}
	}		
}

void RetrieveSubnetGatewayIp(xmlNode *interfaces, char **subnetIp, char **gatewayIp)
{
	xmlNode *node = NULL;
	int found=0;

    // printf("Retrieved count only as %d count=%d\n", countOnly, *count);
    // getchar();

    for (node = interfaces->children; node; node = node->next) 
    {
        if (node->type == XML_ELEMENT_NODE) 
        {
            if(strncmp((char *)node->name, "Interface", 9) == 0)
            {
                struct _xmlAttr * props = node->properties;

                for(props = node->properties; props; props=props->next)
                {
  
                    if(strncmp((char *)props->name, "IsPrimary", 9)==0)
                    {
						xmlChar* value = xmlNodeListGetString(node->doc, props->children, 1);
                        if(strncmp((char *)value, "true", 4) == 0)
                        {
							char *subTemp=NULL, *gwTemp=NULL;;
							xmlFree(value);
							getSubnetIp(node,subnetIp);
							int len = strlen(*subnetIp) + sizeof(char);
							*gatewayIp = (char *)malloc(len);
							
							int dotCount = 0;
							int k = 0;
							subTemp = *subnetIp;
							gwTemp = *gatewayIp;
							for (k = 0; k < len; k++)
							{
								if (subTemp[k] == '.') {
									dotCount++;
								}

								gwTemp[k] = subTemp[k];

								if (dotCount == 3)
								{
									break;
								}
							}
							gwTemp[++k] = '1';
							gwTemp[++k] = '\0';
							found=1;
							break;
                        }
                        xmlFree(value);
                    }                   
                }
            }
        }
		if(found)
			break;
    }
}


void CountOrRetrieveSecondaryIps(xmlNode *interfaces, int countOnly, unsigned int *count, unsigned int *ips)
{
    xmlNode *node = NULL;

    // printf("Retrieved count only as %d count=%d\n", countOnly, *count);
    // getchar();

    for (node = interfaces; node; node = node->next) 
    {
        if (node->type == XML_ELEMENT_NODE) 
        {
            if(strncmp((char *)node->name, "IPAddress", 9) == 0)
            {
                struct _xmlAttr * props = node->properties;
                uint isSecondary = 0;
                xmlChar *savedAddress = 0;
                for(props = node->properties; props; props=props->next)
                {
                    xmlChar* value = xmlNodeListGetString(node->doc, props->children, 1);
                    if(strncmp((char *)props->name, "IsPrimary", 9)==0)
                    {
                        if(strncmp((char *)value, "false", 5) == 0)
                        {
                            isSecondary = 1;
                        }
                        xmlFree(value);
                    }
                    else
                    if(strncmp((char *)props->name, "Address", 7)==0)
                    {
                        if(savedAddress)
                        {
                            xmlFree(savedAddress);
                        }
                        savedAddress = value;                        
                    }
                    else
                    {
                        xmlFree(value);
                    }
                    
                }
                if(isSecondary)
                {
                    if(!countOnly)
                    {
                        // may be we should check if passed memory is sufficient or not
                        ips[*count] = ConvertToIPAddress(savedAddress);
                        printf("Saving ip address %d at location %d\n", ips[*count], *count);
                    }
                    *count = (*count) + 1;
                }
                if(savedAddress)
                {
                    xmlFree(savedAddress);
                }
            }
        }

        CountOrRetrieveSecondaryIps(node->children, countOnly, count, ips);
    }
}

int ParseSecondaryIpAddresses( char *xmlResponse, int countOnly, unsigned int *ips)
{
    xmlDocPtr doc;
    xmlNode *xmlRoot = NULL;
    LIBXML_TEST_VERSION
    doc = xmlReadMemory(xmlResponse, strlen(xmlResponse), "include.xml", NULL, 0);
    xmlRoot = xmlDocGetRootElement(doc);
    unsigned int count = 0;
    CountOrRetrieveSecondaryIps(xmlRoot, countOnly, &count, ips);
    xmlFreeDoc(doc);
    xmlCleanupParser();
    printf("Parsing of ips finished\n");
    return count;
}

int ParseSubnetAndGatewayIp(char *xmlResponse, char **subnetIp, char **gatewayIp)
{
    xmlDocPtr doc;
    xmlNode *xmlRoot = NULL;
    LIBXML_TEST_VERSION
    doc = xmlReadMemory(xmlResponse, strlen(xmlResponse), "include.xml", NULL, 0);
    xmlRoot = xmlDocGetRootElement(doc);
    unsigned int count = 0;
    RetrieveSubnetGatewayIp(xmlRoot, subnetIp, gatewayIp);
    xmlFreeDoc(doc);
    xmlCleanupParser();
    return count;
}
} } } } }
