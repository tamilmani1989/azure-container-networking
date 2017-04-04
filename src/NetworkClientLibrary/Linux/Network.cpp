#ifdef __linux__
#include "../Common/aquanetworkclient.h"
#include <stdlib.h>
#include "../Common/nclcore.h"
#include <curl/curl.h>
#include <string.h>
#define UNREFERENCED_PARAMETER(param) do { (void)(param); } while (0)


extern int ParseSecondaryIpAddresses(char *xmlResponse, int countOnly, unsigned int *ips);

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
			return ipCount;
		}
		curl_easy_cleanup(handle);
		if (ips == NULL)
		{
			ipCount = ParseSecondaryIpAddresses(s.ptr, 1, ips);
			printf("Received only ip count as %d\n", ipCount);
		}
		else
		{
			ipCount = ParseSecondaryIpAddresses(s.ptr, 0, ips);
			printf("Received ips with ip count as %d\n", ipCount);
		}
		freeStringMemory(&s);
	}

	return ipCount;
}

DWORD setupUnderlayInAzure(ClusterConfig *clusterConfig)
{
	return UNSUPPORTED_ENVIRONMENT;
}

DWORD setupOverlay(ClusterConfig *clusterConfig)
{
	return UNSUPPORTED_ENVIRONMENT;
}
#endif
