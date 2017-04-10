#pragma once


#ifdef _WIN32

#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif

#include <windows.h>
#define DllExport   __declspec( dllexport )  
#endif

#ifdef __linux__
#include <pthread.h>
#define DllExport   
typedef unsigned long DWORD
typedef unsigned short USHORT
typedef unsigned char UCHAR
#endif

#include "aquaError.h"
#include <string>
#include <iostream>

using namespace std;
using namespace aqua;