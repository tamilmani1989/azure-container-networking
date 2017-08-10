/*++

Copyright (c) Microsoft Corporation

Module Name:

ResponseParser.cpp

Abstract:

This cotains the implementation for methods to parse xml in Linux.

Author:

Tamilmani Manoharan (tamanoha) 10-April-2017

--*/

#include <string.h>
#include <libxml/parser.h>
#include <arpa/inet.h>
#include "../../Logging/Logging.h"
#include "../Common/NclRepositoryManager.h"
namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
extern void SetRepositoryPrimaryIp(unsigned int ip);

unsigned int convertToIPAddress(unsigned char *ipAddressinStringFormat)
/*++

Routine Description:

	converts an ip address in string format to unsigned int.

Arguments:

	ipAddressinStringFormat - ip address in string format.

Return Value:
	
	ip address in unsinged int. 0 in case of failure.
--*/

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

int getPrimaryIp(xmlNode *interfaceNode)
/*++

Routine Description:
	
	Retrives the primary ip from nmagent response and saves it in repository.

Arguments:

	interfaceNode - node to parse

Return Value:

	returns ip on success else 0 on failure
		
--*/
{
    xmlNode *node = NULL;
    //int primaryFlag=0;
    unsigned int ip=0;
    struct in_addr addr;
    for (node = interfaceNode->children; node; node = node->children) 
    {
        if (node->type == XML_ELEMENT_NODE) 
        {
            if(strncmp((char *)node->name, "IPAddress", 8) == 0)
            {
                struct _xmlAttr * props = node->properties;
                ip=0;
                for(props = node->properties; props; props=props->next)
                {
                    xmlChar* value = xmlNodeListGetString(node->doc, props->children, 1);
                    if(strncmp((char *)props->name, "IsPrimary", 9)==0)
                    {
                        if(strncmp((char *)value, "true", 4) == 0)
                        {
                            if(ip!=0) 
                            {
                                addr.s_addr=ip;
                                LOGINFO("primary ip of primary nic:%d\t%s\n",ip,inet_ntoa(addr));
                                xmlFree(value);
                                return ip;
                            }
                            else
                            {
                                LOGERR("Something went wrong. Primary Ip not found\n");
                            }
                        }
                    }
                    else if(strncmp((char *)props->name, "Address", 7)==0)
                    {

                        ip = inet_addr((char *)value);
                    }
                    xmlFree(value);
                }
            }
        }
    }
    return 0;
}

void getSubnetIp(
    xmlNode *interfaceNode, 
    char **subnetIp)
/*++

Routine Description:
	
	Retrives the subnet ip from nmagent response and saves it in the argument.

Arguments:

	interfaceNode - node to parse
    subnetIp - pointer to buffer where subnetIp will be stored

Return Value:
	N/A
		
--*/
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
                        LOGINFO("Plugin name %s", temp);
                        xmlFree(value);
						break;
					}
				}
			}
		}
	}		
}

void GetSubnetGatewayPrimaryIp(
    xmlNode *interfaces, 
    char **subnetIp, 
    char **gatewayIp,
    NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:

	Parses the nm agent response for gateway and subnet ip.
	Also persists primary ip in repository.

Arguments:

	xmlResponse - nm agent response to parse
	subnetIp - (out praram) will contain subnet ip
	gatewayIp - (out param) will contain gateway ip
	nclRepositoryManager - repository handle to save praimry CA

Return Value:

	N/A
		
--*/

{
	xmlNode *node = NULL;
    unsigned int ip=0;

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
                            ip=getPrimaryIp(node);
                            nclRepositoryManager->SetRepositoryPrimaryIp(ip);
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
                            return;
                        }
                        xmlFree(value);
                    }                   
                }
            }
        }
    }
}


void CountOrRetrieveSecondaryIps(
    xmlNode *interfaces, 
    int countOnly, 
    unsigned int *count, 
    unsigned int *ips)
/*++

Routine Description:

	Parses nm agent response for secondary IPs.

Arguments:

	xmlResponse - nm agent response
	countOnly - boolena indicating whether to only count secondary CAs or to also pupulate the out param
	count - (out parameter) count for seoncary CAs
	ips - (out parameter) values for secondary CAs

Return Value:

	N/A
		
--*/
{
    xmlNode *node = NULL;

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
                        ips[*count] = convertToIPAddress(savedAddress);
                        LOGINFO("Saving ip address %d at location %d\n", ips[*count], *count);
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

int ParseSecondaryIpAddresses( 
    char *xmlResponse, 
    int countOnly, 
    unsigned int *ips)
/*++

Routine Description:

	Initializez COM and calls method to parse nm agent response.

Arguments:

	xmlResponse - nm agent response
	countOnly - boolena indicating whether to only count secondary CAs or to also pupulate the out param

Return Value:

	number of secondary CAs parsed, 0 upon failure.
		
--*/
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
    return count;
}

void ParseSubnetGatewayPrimaryIp(
    char *xmlResponse, 
    char **subnetIp, 
    char **gatewayIp,
    NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:

	Initializes COM and calls method to parse nm agent response.
	
Arguments:

	xmlResponse - nm agent response to parse
	subnetIp - (out praram) will contain subnet ip
	gatewayIp - (out param) will contain gateway ip
    nclRepositoryManager - repository handle to save praimry CA

Return Value:

	N/A
		
--*/
{
    xmlDocPtr doc;
    xmlNode *xmlRoot = NULL;
    LIBXML_TEST_VERSION
    doc = xmlReadMemory(xmlResponse, strlen(xmlResponse), "include.xml", NULL, 0);
    xmlRoot = xmlDocGetRootElement(doc);
    GetSubnetGatewayPrimaryIp(xmlRoot, subnetIp, gatewayIp, nclRepositoryManager);
    xmlFreeDoc(doc);
    xmlCleanupParser();
}
} } } } }
