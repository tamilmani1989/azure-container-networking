#pragma once
#include "..\Common\logger.h"

class WinLogger : public Logger
{
public:
	static void LogInformation(wstring logMessage);
};