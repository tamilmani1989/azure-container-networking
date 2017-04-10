
#pragma once

namespace aqua
{
	const DWORD E_SUCCESS = 0;
	const DWORD E_WSAStartupFailed = 1;
	const DWORD E_GetAddrFailed = 2;
	const DWORD E_SocketFailed = 3;
	const DWORD E_BindFailed = 4;
	const DWORD E_SendFailed = 5;
	const DWORD E_InvalidArguments = 6;
	const DWORD E_CreateAquaNixFailed = 7;
}

#define TraceLog(message) \
	cout << message; \

#define E_AQUA(X, message) \
	result = X; \
	if(result != E_SUCCESS) E_AQ_FAIL(result, message)

#define E_NULLPTR(X, error,message) \
	if(X == nullptr) E_AQ_FAIL(error, message)

#define E_NULL(X, error,message) \
	if(X == NULL) E_AQ_FAIL(error, message)

#define E_AQ_FAIL(error,message) \
	{ \
		cout << message << '\n'; \
		result = error; \
		goto AquaError; \
	} \

#define E_WIN32(X,Y) \
	result = X; \
	if(result != 0) E_FAIL(result, Y) \

#define E_WIN32_EX2(A,X,Y,Z) \
	result = X; \
	if(result != A) E_FAIL(Z,Y) \

#define E_WIN32_EX(X,Y,Z) E_WIN32_EX2(0,X,Y,Z)

#define E_WSA_EX(A,X,Y,Z) \
	iResult = (int)X; \
	if(iResult != A) E_WSA_FAIL(Y,Z)

#define E_WSA_SOCKET_ERROR(X,Y,Z) \
	iResult = (int)X; \
	if(iResult == SOCKET_ERROR) E_WSA_FAIL(Y,Z)

#define E_WSA_FAILFOR(A,X,Y,Z) \
	iResult = (int)X; \
	if(iResult == A) E_WSA_FAIL(Y,Z)

#define E_WSA(X,Y,Z) E_WSA_EX(0,X,Y,Z)

#define E_WSA_FAIL(Y,Z) \
	{\
		cout << Y << iResult << " WSAGetLastError=" << WSAGetLastError() << '\n'; \
		result = Z; \
		goto AquaError; \
	} \

