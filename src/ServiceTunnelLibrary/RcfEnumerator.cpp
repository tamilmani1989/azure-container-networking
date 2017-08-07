/*++

Copyright (c) 2008  Microsoft Corporation

Module Name:

RcfEnumerator.cpp

Abstract:

Implements methods related to enumeation of RCF files.

--*/

//
// ------------------------------------------------------------------- Includes
//

#include "RcfEnumerator.h"

int SortWin32FindDataProcByLastWriteTime(
	const void *elem1,
	const void *elem2
)
/*++

Routine Description:

Helper to compare WIN32_FIND_DATA objects by LastWriteTime

Arguments:

elem1 -To be compared object1.

elem2 - To be compared object2.

Return Value:

return number < 0 if elem1 < elem2 and > 0 if elem1 > elem2 and 0 if eleme1 is equal to elem2.

--*/
{
	WIN32_FIND_DATA *fd1 = (WIN32_FIND_DATA*)elem1;
	WIN32_FIND_DATA *fd2 = (WIN32_FIND_DATA*)elem2;
	return (int)CompareFileTime(&fd1->ftLastWriteTime, &fd2->ftLastWriteTime);
}

HRESULT
RcfEnumerator::GetRcfLocation(
	__out LPWSTR* RcfLocation
)
/*++

Routine Description:

Helper to retrieve the location of the RCF file.
Paas the Rcf location is C:\Config
Iaas the Rcf location is  %SystemDrive%\WindowsAzure\Config


Arguments:

RcfLocation - Out parameter to hold the location fo the RCF file.

Return Value:

S_OK on success. On failure appropriate HRESULT is returned.

--*/
{
	CRegKey vmTypeKey;
	DWORD error = ERROR_SUCCESS;
	HRESULT hr = S_OK;
	WCHAR vmType[MAX_PATH];
	WCHAR configPath[MAX_PATH];
	bool isPaas = true;

	*RcfLocation = NULL;

	error = vmTypeKey.Open(HKEY_LOCAL_MACHINE,
		WINDOWS_AZURE_VM_TYPE_KEY,
		KEY_READ);

	if (error != ERROR_SUCCESS)
	{
		RD_TRACE_LOG((RD_TRACE_ERROR,
			L"Possible Registry Corruption. Cannot open [%s]. Assuming VMType as PaaS. Error:[%0x]",
			WINDOWS_AZURE_VM_TYPE_KEY,
			error));
	}
	else
	{
		error = vmTypeKey.GetValueSz(VM_TYPE_VALUE,
			vmType,
			MAX_PATH);
		if (error != ERROR_SUCCESS)
		{
			RD_TRACE_LOG((RD_TRACE_ERROR,
				L"Registry value not found Key:[%s], Value:[%s]. Assuming VMType as PaaS. Error:[%0x]",
				WINDOWS_AZURE_VM_TYPE_KEY,
				VM_TYPE_VALUE,
				error));
			wprintf(L"hello");
		}
		else
		{
			wprintf(L"vmtype:%s\n", vmType);
			if (wcsncmp(IAAS_STRING, vmType, wcslen(IAAS_STRING)) == 0)
			{
				isPaas = false;
			}
		}
	}

	if (!isPaas)
	{
		if (0 == ExpandEnvironmentStrings(IAAS_RCF_LOCATION, &configPath[0], MAX_PATH))
		{
			error = GetLastError();
			RD_TRACE_E_WIN32((RD_TRACE_ERROR,
				L"ExpandEnvironmentStrings failed for %s. Error:[%0x]", IAAS_RCF_LOCATION, error),
				error,
				Cleanup);
		}

		hr = WcsDupHr(configPath, RcfLocation);
	}
	else
	{
		hr = WcsDupHr(PAAS_RCF_LOCATION, RcfLocation);
	}

	RD_E_HR(hr, Cleanup);

	RD_TRACE_LOG((RD_TRACE_ERROR, L"Using RCF Location:[%s]", *RcfLocation));

Cleanup:

	return hr;
}

HRESULT
RcfEnumerator::FindLatestRcf(
	__out LPWSTR* LatestRcfFile,
	__in LPWSTR rcfLocation)
	/*++

	Routine Description:

	Helper to retrieve the last updated RCF file from a given location.

	Arguments:

	RcfLocation - Out parameter to hold the full path to the latest RCF file.

	Return Value:

	S_OK on success. On failure appropriate HRESULT is returned.

	--*/
{
	HRESULT hr = S_OK;
	//  wregex pattern(L"\\w{32}.\\w{32}.\\w+.\\d+.xml");
	wregex pattern(L"\\w+.\\w+.\\w+.xml");
	std::vector<WIN32_FIND_DATA> fdata;
	WIN32_FIND_DATA fd;
	DWORD error;
	std::wsmatch match_result;
	std::wstring fileName;
	LPWSTR fileFilterString = NULL;

	*LatestRcfFile = NULL;

	error = AppendStrings(&fileFilterString,
		2,
		rcfLocation,
		L"\\*.xml");
	RD_E_WIN32(error, Cleanup);

	HANDLE hFind = ::FindFirstFile(fileFilterString, &fd);
	while (INVALID_HANDLE_VALUE != hFind) {
		fileName.assign(fd.cFileName, wcslen(fd.cFileName));
		if (regex_search(fileName, match_result, pattern))
		{
			fdata.push_back(fd);
		}
		if (!::FindNextFile(hFind, &fd)) {
			::FindClose(hFind);
			break;
		}
	}

	qsort(
		&fdata[0],
		fdata.size(),
		sizeof(WIN32_FIND_DATA),
		SortWin32FindDataProcByLastWriteTime);
	for (int ctr = 0; ctr < fdata.size(); ctr++)
	{
		FILETIME ft;
		SYSTEMTIME st;
		::FileTimeToLocalFileTime(&fdata[ctr].ftLastWriteTime, &ft);
		::FileTimeToSystemTime(&ft, &st);
		RD_TRACE_LOG((RD_TRACE_INFO,
			L"DateTime:%.4d/%.2d/%.2d %.2d:%.2d:%.2d - File:%s",
			st.wYear, st.wMonth, st.wDay, st.wHour, st.wMinute, st.wSecond,
			fdata[ctr].cFileName));
	}

	if (!fdata.empty())
	{
		error = AppendStrings(LatestRcfFile,
			3,
			rcfLocation,
			L"\\",
			fdata.back().cFileName);
		if (error != ERROR_SUCCESS)
		{
			RD_TRACE_LOG((RD_TRACE_ERROR, L"Failed to append string [%s],[%s], Error:[%0x]",
				rcfLocation,
				fdata.back().cFileName,
				error));
			RD_E_WIN32(error, Cleanup);
		}
	}
	else
	{
		error = ERROR_FILE_NOT_FOUND;
		RD_TRACE_LOG((RD_TRACE_ERROR, L"Failed to find RCF files at location %s", fileFilterString));
		RD_E_WIN32(error, Cleanup);
	}

	RD_TRACE_LOG((RD_TRACE_INFO, L"Using RCF File:[%s]", *LatestRcfFile));

Cleanup:

	STR_FREE(rcfLocation);
	STR_FREE(fileFilterString);
	return hr;
}


