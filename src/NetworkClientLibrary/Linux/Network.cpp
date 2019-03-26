#ifdef __linux__
/*++

Copyright (c) Microsoft Corporation

Module Name:

Network.cpp

Abstract:

This file contains methods that can be used to setup container network.

Author:

Tamilmani Manoharan (tamanoha) 3-April-2017

--*/

#include "../Common/aquanetworkclient.h"
#include <stdlib.h>
#include "../Common/nclcore.h"
#include "../../Logging/Logging.h"
#include "../Common/NetworkCommon.h"
#include "../Common/NclRepositoryManager.h"
#include <curl/curl.h>
#include <string.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#define UNREFERENCED_PARAMETER(param) do { (void)(param); } while (0)
#define DockerHostNetwork "http://127.0.0.1:2375/networks/"
#define NAME_CONST_LEN 9
#define DRIVER_CONST_LEN 11


namespace Microsoft { namespace Azure {	namespace Networking {	namespace Aqua { namespace ClientLibrary {
extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern void ParseSubnetGatewayPrimaryIp( char *xmlResponse, char **subnetIp, char **gatewayIp, NclRepositoryManager *nclRepositoryManager);

extern DllExport unsigned int GetRepositoryPrimaryIp();
extern DllExport NclRepositoryManager *GetIpRepository();


typedef struct string
/*++
	ptr - character array used to store receive data from curl
	len - size of character array
--*/
{
	char *ptr;
	size_t len;
}string;


void initString(string *s)
/*++
Routine Description:
	
	Allocates ptr in the string structure. 

Arguments:

	s - string structure pointer

Return Value:

--*/
{
	s->len = 0;
	s->ptr = (char *)malloc(s->len + 1);
	if (s->ptr == NULL)
	{
		LOGERR("malloc() failed\n");
		exit(EXIT_FAILURE);
	}
	s->ptr[0] = '\0';
}

void freeStringMemory(string *s)
/*++
Routine Description:
	
	Free ptr in the string structure. 

Arguments:

	s - string structure pointer

Return Value:

--*/
{
	if (s->len>0)
	{
		free(s->ptr);
	}
}

size_t saveData(
	void *source, 
	size_t size, 
	size_t nmemb, 
	string *target)
/*++
Routine Description:

	curl callback function for writing(saving) received data. The userdata (target) argument is set with CURLOPT_WRITEDATA option.

Arguments:

	source - points to data received by curl
	size - size of block
	nmemb - number of blocks
	target - userdata pointer 

Return Value:

	size of the received data

--*/
{
	size_t newLength = target->len + size*nmemb;
	target->ptr = (char*)realloc(target->ptr, newLength + 1);
	if (target->ptr == NULL)
	{
		fprintf(stderr, "realloc() failed\n");
		exit(EXIT_FAILURE);
	}
	memcpy(target->ptr + target->len, source, size*nmemb);
	target->ptr[newLength] = '\0';
	target->len = newLength;

	return size*nmemb;
}

int GetIpsFromNmAgent(unsigned int *ips)
/*++

Routine Description:

	Retrieves interface config from nmagent.

Arguments:

	ips - if null, only ip count is returned. otherwise, all secondary ips are populated in it.

Return Value:

	count - number of secondary CAs from nm agent.

--*/
{
	CURL *handle;
	CURLcode res;
	handle = curl_easy_init();
	int retval = 1;
	int ipCount = 0;

	LOGINFO("Going to get ips from nm agent\n");

	if (handle)
	{
		curl_easy_setopt(handle, CURLOPT_URL, "http://168.63.129.16/machine/plugins?comp=nmagent&type=getinterfaceinfov1");
		curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, saveData);
		string s;
		initString(&s);
		curl_easy_setopt(handle, CURLOPT_WRITEDATA, &s);
		res = curl_easy_perform(handle);
		if (res != CURLE_OK)
		{
			LOGERR("curl_easy_perform() failed %d\n", res);
			retval=0;
		}

		if (retval && ips == NULL)
		{
			ipCount = ParseSecondaryIpAddresses(s.ptr, 1, ips);
			LOGINFO("Received only ip count as %d\n", ipCount);
		}
		else if(retval)
		{
			ipCount = ParseSecondaryIpAddresses(s.ptr, 0, ips);
			LOGINFO("Received ips with ip count as %d\n", ipCount);
		}
		freeStringMemory(&s);
		curl_easy_cleanup(handle);
	}

	return ipCount;
}

int getSubnetGwIpAndSetPrimary(
	char **subnetIp, 
	char **gatewayIp,
	NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:

	retrieves interface descriptionf rom nm agent, and saves primary CA, subnet and gateway.

Arguments:

	subnetIp - subnet ip address
	gwIp - ip address of the gateway
	nclRepositoryManager - repository manager handle to set primary ip

Return Value:

	0 on success, 1 on failure.

--*/
{
	CURL *handle;
	CURLcode res;
	handle = curl_easy_init();
	int retval = 1;

	LOGINFO("Going to get ips from nm agent\n");

	if (handle)
	{
		curl_easy_setopt(handle, CURLOPT_URL, "http://168.63.129.16/machine/plugins?comp=nmagent&type=getinterfaceinfov1");
		curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, saveData);
		string s;
		long httpCode=0;
		initString(&s);
		curl_easy_setopt(handle, CURLOPT_WRITEDATA, &s);
		res = curl_easy_perform(handle);
		if (res != CURLE_OK)
		{
			LOGERR("curl_easy_perform() failed %d\n", res);
			retval=0;
		}
		
		if(retval)
		{
			curl_easy_getinfo (handle, CURLINFO_RESPONSE_CODE, &httpCode);
			
			if(httpCode!=200)
			{
				LOGERR("Http Error:%ld\n",httpCode);
				LOGERR("Http Response:%s\n",s.ptr);
				retval=0;
			}
		}
		
		if(retval)
			ParseSubnetGatewayPrimaryIp(s.ptr, subnetIp, gatewayIp, nclRepositoryManager);

		freeStringMemory(&s);
		curl_easy_cleanup(handle);
	}
	return retval;
}

int getSubnetGwIpfromConfig(
	ClusterConfig *clusterConfig, 
	char *subnetCidr, 
	int subnetSize, 
	char *nodeGwIp, 
	int gwSize)
/*++

Routine Description:

	retrieves subnetIp from cluster config and finds gateway ip using subnet ip.

Arguments:

	clusterConfig - ClusterConfig structure which contains subnet ip
	subnetCidr - buffer that points where subnet ip to be stored
	subnetSize - allocated size of subnetCidr buffer
	nodeGwIp - buffer for gateway ip
	gwSize - allocated size of nodeGwIp buffer

Return Value:

	0 on success, 1 on failure.

--*/
{
	int i;
	bool found=false;
	struct in_addr addr;
	char *subnetIpTemp=NULL;
	char nodeSubnetIp[32];
	int k,dotCount=0,len=0;
	
	for (i=0; i<clusterConfig->nodeCount; i++)
	{
		if(clusterConfig->localNodeIp == clusterConfig->nodeConfigs[i].nodeIp)
		{
			found = true;
			break;
		}
	}
	
	if(found)
	{
		addr.s_addr = clusterConfig->nodeConfigs[i].subnet->ipAddress;
		subnetIpTemp = inet_ntoa(addr);
		strncpy(nodeSubnetIp,subnetIpTemp, sizeof(nodeSubnetIp)-1);
		snprintf(subnetCidr, subnetSize, "%s/%d",nodeSubnetIp,  clusterConfig->nodeConfigs[i].subnet->prefixLength);
		len = strlen(nodeSubnetIp);
		for (k = 0; k < len && k < gwSize; k++)
		{
			if (nodeSubnetIp[k] == '.') {
				dotCount++;
			}

			nodeGwIp[k] = nodeSubnetIp[k];

			if (dotCount == 3)
			{
				break;
			}
		}
	
		nodeGwIp[++k] = '1';
		nodeGwIp[++k] = '\0';
	}
		
	else
		return 0;
	
	return 1;
}

int networkNotExists(char *networkName)
/*++

Routine Description:
		
	Talks to docker and see if network exists or not.

Arguments:

	networkName - name of the network that is to be verified for existence

Return Value:

	1 if it didn't exist, 0 if network already exists or error

--*/
{
	CURL *handle;
	CURLcode res;
	int retval=1;
	handle = curl_easy_init();

	if (handle)
	{
		string s;
		char *getUrl=NULL;
		long httpCode=0;
		int len = strlen(DockerHostNetwork) + strlen(networkName) + sizeof(char);
		getUrl = (char *)malloc(len);
		snprintf(getUrl, len,"%s%s",DockerHostNetwork,networkName);
		LOGVERBOSE("geturl:%s\n", getUrl);
		
		curl_easy_setopt(handle, CURLOPT_URL, getUrl);
		curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, saveData);
	
		initString(&s);
		curl_easy_setopt(handle, CURLOPT_WRITEDATA, &s);
		res = curl_easy_perform(handle);
		if (res != CURLE_OK)
		{
			LOGERR("curl_easy_perform() failed %d\n", res);
			retval = 0;
		}
		
		if(retval)
		{
			curl_easy_getinfo (handle, CURLINFO_RESPONSE_CODE, &httpCode);
			if(httpCode!=404)
			{
				if(httpCode == 200) {
					LOGWARN("Network Initialized already\n");
				}
				else {
					LOGERR("Http Error:%ld\n",httpCode);
				}
				retval=0;
			}
		}
	
		freeStringMemory(&s);
		
		if(getUrl)
			free(getUrl);
		curl_easy_cleanup(handle);

	}

	return retval;
}

int createDockerNetworkDriver(char *postFields)
/*++

Routine Description:

	Creates a network with docker. The network driver is azure-vnet for underlay and azure-overlay for overlay and is not input.

Arguments:

	postFields - contains http header fields and request body.

Return Value:

	0 on success, error code otherwise.

--*/

{
	CURL *handle;
	CURLcode res;
	int retval=1;
	handle = curl_easy_init();
	
	if (handle)
	{
		string s;	
		long httpCode=0;		
		const char *createStr = "create";		
		char *url=NULL;
		struct curl_slist *headers=NULL;
		headers = curl_slist_append(headers, "Content-Type: application/json");
		int len = strlen(DockerHostNetwork) + strlen(createStr) + sizeof(char);
		
		url = (char *)malloc(len);
		snprintf(url, len,"%s%s",DockerHostNetwork,createStr);
		
		LOGINFO("geturl:%s postfields:%s\n", url,postFields);
		
		curl_easy_setopt(handle, CURLOPT_URL, url);
		curl_easy_setopt(handle, CURLOPT_POST, 1);
		curl_easy_setopt(handle, CURLOPT_HTTPHEADER, headers);
		curl_easy_setopt(handle, CURLOPT_POSTFIELDS, postFields);
		curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, saveData);
	
		initString(&s);
		curl_easy_setopt(handle, CURLOPT_WRITEDATA, &s);
		res = curl_easy_perform(handle);
		if (res != CURLE_OK)
		{
			LOGERR("curl_easy_perform() failed %d\n", res);
			retval=0;
		}
		if(retval)
		{
			curl_easy_getinfo (handle, CURLINFO_RESPONSE_CODE, &httpCode);
			
			if(httpCode!=201)
			{
				LOGERR("Http Error:%ld\n",httpCode);
				LOGERR("create network response:%s\n",s.ptr);
				retval=0;
			}			
		}
		
		curl_slist_free_all(headers);
			
		freeStringMemory(&s);
		
		if(url)
			free(url);
		curl_easy_cleanup(handle);	
	}

	return 1;
}

DWORD SetupUnderlayInAzure(
	ClusterConfig *clusterConfig,
	NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:
	
	Sets up the underlay network in Azure.

Arguments:
	
	clusterConfig - contains name of the network to be created.
	nclRepositoryManager - repository where metadata about network will be persisted.

Return Value:
	0 on success, error code otherwise.

--*/
	
{
  int retval=0;
  char *subnetIp=NULL, *gatewayIp=NULL;
  size_t conLength = 0, ipamLength = 0;
  char *postFields = NULL;
  char ipamConfig[2048];
  const char *driver="azure-vnet";
		
  if(!getSubnetGwIpAndSetPrimary(&subnetIp, &gatewayIp, nclRepositoryManager))
  {
  	 retval=1;
	 LOGERR("Error in retrieving subnet and gw ip\n");
	 goto cleanup_setupunderlay;
  }
  if(networkNotExists(clusterConfig->networkName))
  {

	  	LOGINFO("Netowrk %s does not exist, going to create it\n", clusterConfig->networkName);

		LOGINFO("retrieved subnet and gateway %s %s\n", subnetIp, gatewayIp);

		snprintf(ipamConfig, sizeof(ipamConfig), "\"IPAM\":{\"Config\":[{\"Subnet\":\"%s\",\"Gateway\":\"%s\"}]}", subnetIp, gatewayIp);

		ipamLength = strlen(ipamConfig);
		conLength = strlen(clusterConfig->networkName) + NAME_CONST_LEN + DRIVER_CONST_LEN + 8 + ipamLength + strlen(driver) ;
		postFields = (char *)malloc(conLength + 1);

		snprintf(postFields, conLength + 1, "\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}",clusterConfig->networkName, driver, ipamConfig);

		LOGINFO("Now going to get create network driver... cmd:%s\n", postFields);
		
		if(!createDockerNetworkDriver(postFields))
		{
			LOGERR("Creating Underlay Network in Azure failed\n");
			retval=1;
		}
		else
			LOGINFO("Docker underlay network create succeeded\n");
  }

cleanup_setupunderlay:
  if(subnetIp)
	  free(subnetIp);
  if(gatewayIp)
	  free(gatewayIp);
  if(postFields)
	  free(postFields);
	
  return retval;
}

char* constructOptionFields(ClusterConfig *clusterConfig)
/*++

Routine Description:
	
	This function constructs option parameters that needs to be passed to azure-overlay plugin via -o option

Arguments:
	
	clusterConfig - Option parameters will be retrieved from ClusterConfig structure 

Return Value:

	returns char pointer to option format msg if succeeds els returns NULL in case of error 

--*/

{
	char *nodeList=NULL;
	char *nodeListOption = NULL;
	int nodeListLen=0,nodeListOptionLen=0;
	int i=0;
	char *nodeId=NULL;
	char nodeSubnetIp[32];
	char nodeRemoteIp[64];
	char **nodeInfoOption;
	int nodeInfoOptionLen=0;
	char *remoteIpTemp=NULL;
	char *subnetIpTemp=NULL;
	struct in_addr addr;
	int totalNodeInfoLen=0;
	char *optionFields=NULL;
	int optionFieldsLen=0;
	char *mergeNodeInfo=NULL;
	int mergeNodeInfoLen=0;

	LOGVERBOSE("Node count:%d\n",clusterConfig->nodeCount);
	nodeListLen = clusterConfig->nodeCount * 256 + clusterConfig->nodeCount-1 + sizeof(char);
	nodeList = (char *)malloc(nodeListLen);
	memset(nodeList,0,nodeListLen);
	for(i=0;i<clusterConfig->nodeCount-1;i++)
	{
		nodeId = clusterConfig->nodeConfigs[i].nodeId;
		if(clusterConfig->localNodeIp == clusterConfig->nodeConfigs[i].nodeIp)
			continue;
		strcat(nodeList,nodeId );
		strcat(nodeList, ",");
	}
	
	if(clusterConfig->localNodeIp != clusterConfig->nodeConfigs[i].nodeIp)
		strcat(nodeList,clusterConfig->nodeConfigs[clusterConfig->nodeCount-1].nodeId);
	else
		nodeList[strlen(nodeList)-1]='\0';
	
	LOGVERBOSE("nodeList:%s\n", nodeList);
	
	nodeListOptionLen = strlen(nodeList) + 16;
	nodeListOption = (char *) malloc(nodeListOptionLen);
	snprintf(nodeListOption, nodeListOptionLen, "\"nodeList\":\"%s\"",nodeList);
	
	nodeInfoOption = (char **) malloc(clusterConfig->nodeCount * sizeof(char *));
	memset(nodeInfoOption, 0, clusterConfig->nodeCount * sizeof(char *));
	
	for(i=0;i<clusterConfig->nodeCount;i++)
	{
		addr.s_addr = clusterConfig->nodeConfigs[i].nodeIp;
		
		if(clusterConfig->localNodeIp == addr.s_addr)
		{
				continue;
		}
		
		remoteIpTemp = inet_ntoa(addr);
		strncpy(nodeRemoteIp, remoteIpTemp, sizeof(nodeRemoteIp)-1);
		
		addr.s_addr = clusterConfig->nodeConfigs[i].subnet->ipAddress;
		subnetIpTemp = inet_ntoa(addr);
		strncpy(nodeSubnetIp,subnetIpTemp, sizeof(nodeSubnetIp)-1);
		
		nodeInfoOptionLen = strlen(clusterConfig->nodeConfigs[i].nodeId) + strlen(nodeSubnetIp) + strlen(nodeRemoteIp) + 16;
		nodeInfoOption[i] = (char *)malloc(nodeInfoOptionLen);
		snprintf(nodeInfoOption[i], nodeInfoOptionLen, "\"%s\":\"%s,%s/%d\"", clusterConfig->nodeConfigs[i].nodeId, nodeRemoteIp, nodeSubnetIp,
					clusterConfig->nodeConfigs[i].subnet->prefixLength);
		totalNodeInfoLen += strlen(nodeInfoOption[i]);			
		LOGINFO("node info:%s\n", nodeInfoOption[i]);
	}
	

	mergeNodeInfo = (char *)malloc(totalNodeInfoLen + sizeof(char) + clusterConfig->nodeCount-1);
	memset(mergeNodeInfo, 0, totalNodeInfoLen + sizeof(char));
	
	for(i=0;i<clusterConfig->nodeCount;i++)
	{
		if(nodeInfoOption[i]){
			strcat(mergeNodeInfo, nodeInfoOption[i]);
			strcat(mergeNodeInfo, ",");
		}
	}
	
	mergeNodeInfoLen = strlen(mergeNodeInfo);
	
	mergeNodeInfo[mergeNodeInfoLen-1]='\0';


	optionFieldsLen = strlen(nodeListOption) + mergeNodeInfoLen-1 + 16 + sizeof(char);
	optionFields = (char *) malloc(optionFieldsLen);
	snprintf(optionFields, optionFieldsLen,"\"Options\":{%s,%s}",nodeListOption,mergeNodeInfo);


	if(nodeList)
	  free(nodeList);	
	if(nodeListOption)
	  free(nodeListOption);	
	for(i=0;i<clusterConfig->nodeCount;i++)
	{
	  if(nodeInfoOption[i])
		  free(nodeInfoOption[i]);
	}
	if(nodeInfoOption)
	  free(nodeInfoOption);
	if(mergeNodeInfo)
	  free(mergeNodeInfo);
	
	return optionFields;
}

DWORD SetupOverlay(ClusterConfig *clusterConfig)
/*++

Routine Description:
	
	Sets up the Overlay Network.

Arguments:
	
	clusterConfig - contains name of the network to be created.

Return Value:
	0 on success, error code otherwise.

--*/

{
	int retval=0;
	char subnetIp[32], gatewayIp[32];
	size_t conLength = 0, ipamLength = 0;
	char *postFields = NULL;
	char ipamConfig[2048];
	const char *driver="azure-overlay";
	int postFieldsLen=0;
	char *optionFields=NULL;
	
	GetIpRepository()->SetRepositoryPrimaryIp(clusterConfig->localNodeIp);
	if(networkNotExists(clusterConfig->networkName))
	{

		LOGINFO("Netowrk %s does not exist, going to create it\n", clusterConfig->networkName);

		if(!getSubnetGwIpfromConfig(clusterConfig, subnetIp, sizeof(subnetIp), gatewayIp, sizeof(gatewayIp)))
		{
			retval=1;
			LOGERR("Error in retrieving subnet and gw ip\n");
			goto cleanup_SetupOverlay;
		}
		LOGINFO("retrieved subnet and gateway %s %s\n", subnetIp, gatewayIp);

		snprintf(ipamConfig, sizeof(ipamConfig), "\"IPAM\":{\"Config\":[{\"Subnet\":\"%s\",\"Gateway\":\"%s\"}]}", subnetIp, gatewayIp);

		ipamLength = strlen(ipamConfig);
		conLength = strlen(clusterConfig->networkName) + NAME_CONST_LEN + DRIVER_CONST_LEN + 12 + ipamLength + strlen(driver) ;

		optionFields = constructOptionFields(clusterConfig);

		postFieldsLen = conLength + strlen(optionFields) + sizeof(char);
		postFields = (char *)malloc(postFieldsLen);

		snprintf(postFields, postFieldsLen, "\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s,%s}",clusterConfig->networkName, driver, ipamConfig,optionFields);

		LOGINFO("Now going to get create network driver... cmd:%s\n", postFields);

		if(!createDockerNetworkDriver(postFields))
		{
			LOGERR("Creating Overlay Network failed\n");
			retval=1;
		}
		else
			LOGINFO("Docker Overlay network create succeeded\n");
	}

	cleanup_SetupOverlay:

	if(postFields)
	  free(postFields);
	if(optionFields)
	  free(optionFields);
	return retval;

}

DWORD GetHostIp(unsigned int *ip, NclRepositoryManager *nclRepositoryManager)
/*++

Routine Description:

	For Overlay, it will be retrieved from ClusterConfig Structure
	For underlay, it returns the primary CA of primary nic
	
Arguments:

	ip - to be populated with the HostIp

Return Value:

	0 on success, error code otherwise

--*/
{
	*ip = nclRepositoryManager->GetRepositoryPrimaryIp();
	if(*ip!=0)
		return 0;
	return 1;
}
} } } } }
#endif
