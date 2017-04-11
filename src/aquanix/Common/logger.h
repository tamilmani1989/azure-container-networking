#pragma once
#include "aquabase.h"

class Logger
{
public:
	virtual void LogInformation(wstring logMessage) = 0;
};