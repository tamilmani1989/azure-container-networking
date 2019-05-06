
/*++

Copyright (c) Microsoft Corporation

Module Name:

StatusHandler.h

Abstract:

This contains implementation of methods related to writing extension status as json

Author:

Tamilmani Manoharan (tamanoha) 06-May-2019

--*/

#include "StatusHandler.h"
#include "../Logging/Logging.h"
#include <iostream>
#include <fstream>

using namespace Microsoft::Azure::Networking::Aqua::ClientLibrary;
using namespace std;

#define BUFFER_SIZE 1024

void writeToFile(char *fileName, wstring buffer)
{
    wofstream myfile;
    myfile.open (fileName);
    myfile << buffer;
    myfile.close();
}

json::value processFormattedMessage(FormattedMessage formattedMessage)
{
    json::value value;
    value[LangStr] = json::value::string(formattedMessage.lang);
    value[MessageStr] = json::value::string(formattedMessage.message);

    return value;
}

json::value processMessage(Message message)
{
    json::value value;
    vector<json::value> elements;
    vector<wstring>::iterator it;

    value[IdStr] = json::value::string(message.id);

    for(it = message.params.begin(); it !=  message.params.end(); it++)
    {
        json::value param = json::value::string(*it);
        elements.push_back(param);
    }

    value[ParamsStr] = json::value::array(elements);

    return value;
}

json::value processSubstatus(Substatus substatus)
{
    json::value value;

    value[NameStr] = json::value::string(substatus.name);
    value[StatusStr] = json::value::string(substatus.status);
    value[CodeStr] = substatus.code;
    value[MessageStr] = processMessage(substatus.message);
    value[FormattedMessageStr] = processFormattedMessage(substatus.formattedMessage);

    return value;
}

json::value processStatus(Status status)
{
    json::value value;
    vector<json::value> elements;
    vector<Substatus>::iterator it;

    value[NameStr] = json::value::string(status.name);
    value[OperationStr] = json::value::string(status.operation);
    value[ConfigurationAppliedTimeStr] = json::value::string(status.configurationAppliedTime);
    value[StatusStr] = json::value::string(status.status);
    value[CodeStr] = status.code;
    value[MessageStr] = processMessage(status.message);
    value[FormattedMessageStr] = processFormattedMessage(status.formattedMessage);

    for(it = status.substatus.begin(); it !=  status.substatus.end(); it++)
    {
        json::value substatusValue = processSubstatus(*it);
        elements.push_back(substatusValue);
    }

    value[SubstatusStr] = json::value::array(elements);

    return value;
}

json::value StatusHandler::ToJson()
{
    json::value value;

    value[VersionStr] = json::value::string(this->version);
    value[TimestampUTCStr] = json::value::string(this->timestampUTC);
    value[StatusStr] = processStatus(this->status);

    return value;
}

void StatusHandler::WriteExtensionStatus(char *fileName, vector<json::value> statusHandlers)
{
    json::value value = json::value::array(statusHandlers);
    wstring buffer = value.serialize();
    LOGINFO("Status filename %s", fileName);
    LOGINFOW(L"status value: %s", buffer.c_str());

    writeToFile(fileName, buffer);
}
