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
                        if(strncmp((char *)value, "false", 4) == 0)
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