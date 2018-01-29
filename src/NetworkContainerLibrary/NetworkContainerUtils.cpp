#include "NetworkContainerUtils.h"

using namespace std;

template <class T>
void NetworkContainerUtils::splitWstr(const wstring& Wstr, T& container, wchar_t delim)
{
	wstringstream ss(Wstr);
	wstring token;
	while (getline(ss, token, delim)) { container.push_back(token); }
}