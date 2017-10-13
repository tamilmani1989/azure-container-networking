
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
	const DWORD E_InsufficientMemory = 8;
	const DWORD E_CouldNotCreateRawMessage = 9;
	const DWORD E_GetLocalMappingsFailed = 10;
}

#define ENTER(method) \
	wchar_t *methodName = method; \
	wchar_t logMessage[2048]; \
	size_t msgSize = 2048; \
	DWORD result = E_SUCCESS; \
	LogXA(L"Enter %s", methodName) \

#define EXIT() \
	LogXAB(L"Exit %s. Result:%d", methodName, result) \
	return result; \

#define EXITNORETURN() \
	LogXAB(L"Exit %s. Result:%d", methodName, result) \

#define LogX(X) \
	WinLogger::LogInformation(X); \

#define LogXA(X, A) \
	swprintf_s(logMessage,msgSize,X,A); \
	WinLogger::LogInformation(logMessage); \

#define LogXAB(X, A, B) \
	swprintf_s(logMessage,msgSize,X,A,B); \
	WinLogger::LogInformation(logMessage); \

#define LogXABC(X, A, B, C) \
	swprintf_s(logMessage,msgSize,X,A,B,C); \
	LogX(logMessage); \

#define LogXABCD(X, A, B, C, D) \
	swprintf_s(logMessage,msgSize,X,A,B,C,D); \
	LogX(logMessage); \

#define LogXABCDE(X, A, B, C, D, E) \
	swprintf_s(logMessage,msgSize,X,A,B,C,D,E); \
	LogX(logMessage); \

#define LogXABCDEF(X, A, B, C, D, E, F) \
	swprintf_s(logMessage,msgSize,X,A,B,C,D,E,F); \
	LogX(logMessage); \

#define TraceLog(message) \
	cout << message; \

#define E_AQUA(X, message) \
	result = X; \
	if(result != E_SUCCESS) E_AQ_FAIL(result, message) \

#define E_NULLPTR(X, error,message) \
	if(X == nullptr) E_AQ_FAIL(error, message)

#define E_NULL(X, error,message) \
	if(X == NULL) E_AQ_FAIL(error, message)

#define E_AQ_FAIL(error,message) \
	{ \
		LogXAB(L"%s . Error:%d", message, error); \
		result = error; \
		goto AquaError; \
	} \

#define E_ZERO(X,Y,Z) \
	result = X; \
	if(result != 0) E_AQ_FAIL(Z,Y) \

#define E_WIN32(X,Y,Z) E_ZERO(X,Y,Z)

#define E_WIN32_EX2(A,X,Y,Z) \
	result = X; \
	if(result != A) E_AQ_FAIL(Z,Y) \

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

