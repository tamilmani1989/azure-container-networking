#ifdef __linux__
#include "../Common/aquanetworkclient.h"
#include <stdlib.h>
#include "../Common/nclcore.h"
#include <curl/curl.h>
#include <string.h>
#define UNREFERENCED_PARAMETER(param) do { (void)(param); } while (0)
#define DockerHostNetwork "http://127.0.0.1:2375/networks/"
#define NAME_CONST_LEN 9
#define DRIVER_CONST_LEN 11

extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);
extern int ParseSubnetAndGatewayIp( char *xmlResponse, char **subnetIp, char **gatewayIp);
typedef struct string
{
	char *ptr;
	size_t len;
}string;


void initString(string *s)
{
	s->len = 0;
	s->ptr = (char *)malloc(s->len + 1);
	if (s->ptr == NULL)
	{
		fprintf(stderr, "malloc() failed\n");
		exit(EXIT_FAILURE);
	}
	s->ptr[0] = '\0';
}

void freeStringMemory(string *s)
{
	if (s->len>0)
	{
		free(s->ptr);
	}
}

size_t saveData(void *source, size_t size, size_t nmemb, string *target)
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

int getIpsFromNmAgent(unsigned int *ips)
{
	CURL *handle;
	CURLcode res;
	handle = curl_easy_init();
	int retval = 1;
	int ipCount = 0;

	printf("Going to get ips from nm agent\n");

	if (handle)
	{
		curl_easy_setopt(handle, CURLOPT_URL, "http://169.254.169.254/machine/plugins?comp=nmagent&type=getinterfaceinfov1");
		curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, saveData);
		string s;
		initString(&s);
		curl_easy_setopt(handle, CURLOPT_WRITEDATA, &s);
		res = curl_easy_perform(handle);
		if (res != CURLE_OK)
		{
			fprintf(stderr, "curl_easy_perform() failed %d\n", res);
			retval=0;
		}

		if (retval && ips == NULL)
		{
			ipCount = ParseSecondaryIpAddresses(s.ptr, 1, ips);
			printf("Received only ip count as %d\n", ipCount);
		}
		else if(retval)
		{
			ipCount = ParseSecondaryIpAddresses(s.ptr, 0, ips);
			printf("Received ips with ip count as %d\n", ipCount);
		}
		freeStringMemory(&s);
		curl_easy_cleanup(handle);
	}

	return ipCount;
}

int getSubnetGwIpfromNMAgent(char **subnetIp, char **gatewayIp)
{
	CURL *handle;
	CURLcode res;
	handle = curl_easy_init();
	int retval = 1;

	printf("Going to get ips from nm agent\n");

	if (handle)
	{
		curl_easy_setopt(handle, CURLOPT_URL, "http://169.254.169.254/machine/plugins?comp=nmagent&type=getinterfaceinfov1");
		curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, saveData);
		string s;
		long httpCode=0;
		initString(&s);
		curl_easy_setopt(handle, CURLOPT_WRITEDATA, &s);
		res = curl_easy_perform(handle);
		if (res != CURLE_OK)
		{
			fprintf(stderr, "curl_easy_perform() failed %d\n", res);
			retval=0;
		}
		
		if(retval)
		{
			curl_easy_getinfo (handle, CURLINFO_RESPONSE_CODE, &httpCode);
			
			if(httpCode!=200)
			{
				printf("Http Error:%ld\n",httpCode);
				printf("Http Response:%s\n",s.ptr);
				retval=0;
			}
		}
		
		if(retval)
			ParseSubnetAndGatewayIp(s.ptr, subnetIp, gatewayIp);

		freeStringMemory(&s);
		curl_easy_cleanup(handle);
	}
	return retval;
}


int networkNotExists(char *networkName)
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
		printf("geturl:%s\n", getUrl);
		
		curl_easy_setopt(handle, CURLOPT_URL, getUrl);
		curl_easy_setopt(handle, CURLOPT_WRITEFUNCTION, saveData);
	
		initString(&s);
		curl_easy_setopt(handle, CURLOPT_WRITEDATA, &s);
		res = curl_easy_perform(handle);
		if (res != CURLE_OK)
		{
			fprintf(stderr, "curl_easy_perform() failed %d\n", res);
			retval = 0;
		}
		
		if(retval)
		{
			curl_easy_getinfo (handle, CURLINFO_RESPONSE_CODE, &httpCode);
			if(httpCode!=404)
			{
				if(httpCode == 200)
					printf("Network Initialized already\n");
				else
					printf("Http Error:%ld\n",httpCode);
				
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

int createUnderlayNetworkDriver(char *postFields)
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
		
		printf("geturl:%s postfields:%s\n", url,postFields);
		
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
			fprintf(stderr, "curl_easy_perform() failed %d\n", res);
			retval=0;
		}
		if(retval)
		{
			curl_easy_getinfo (handle, CURLINFO_RESPONSE_CODE, &httpCode);
			
			if(httpCode!=201)
			{
				printf("Http Error:%ld\n",httpCode);
				printf("create network response:%s\n",s.ptr);
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

DWORD setupUnderlayInAzure(ClusterConfig *clusterConfig)
{
  int retval=0;
  char *subnetIp=NULL, *gatewayIp=NULL;
  size_t conLength = 0, ipamLength = 0;
  char *postFields = NULL;
  char ipamConfig[2048];
  const char *driver="azure-vnet";
		
  if(networkNotExists(clusterConfig->networkName))
  {

	  	printf("Netowrk %s does not exist, going to create it\n", clusterConfig->networkName);

		if(!getSubnetGwIpfromNMAgent(&subnetIp, &gatewayIp))
		{
			retval=1;
			printf("Error in retrieving subnet and gw ip\n");
			goto cleanup_setupunderlay;
		}
		printf("retrieved subnet and gateway %s %s\n", subnetIp, gatewayIp);

		snprintf(ipamConfig, sizeof(ipamConfig), "\"IPAM\":{\"Config\":[{\"Subnet\":\"%s\",\"Gateway\":\"%s\"}]}", subnetIp, gatewayIp);

		ipamLength = strlen(ipamConfig);
		conLength = strlen(clusterConfig->networkName) + NAME_CONST_LEN + DRIVER_CONST_LEN + 8 + ipamLength + strlen(driver) ;
		postFields = (char *)malloc(conLength + 1);

		snprintf(postFields, conLength + 1, "\r\n\r\n{\"Name\":\"%s\",\"Driver\":\"%s\",%s}",clusterConfig->networkName, driver, ipamConfig);

		printf("Now going to get create network driver... cmd:%s\n", postFields);
		
		if(!createUnderlayNetworkDriver(postFields))
		{
			printf("Creating Underlay Network in Azure failed\n");
			retval=1;
		}
		else
			printf("Docker underlay network create succeeded\n");
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

DWORD setupOverlay(ClusterConfig *clusterConfig)
{
	return UNSUPPORTED_ENVIRONMENT;
}
#endif
