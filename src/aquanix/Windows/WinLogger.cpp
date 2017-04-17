#include "WinLogger.h"
#include "..\..\Shared\Utilities.h"


void WinLogger::LogInformation(wstring logMsg)
{
	AquariusUtilities::WriteEventLogEntry((WCHAR *)logMsg.c_str());
}