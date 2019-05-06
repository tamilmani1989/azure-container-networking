/*++

Copyright (c) Microsoft Corporation

Module Name:

StatusHandler.h

Abstract:

This contains class and methods related to writing extension status

Author:

Tamilmani Manoharan (tamanoha) 06-May-2019

--*/

#pragma once
#include <string>
#include <windows.h>
#include<vector>
#include<cpprest/json.h>

#define VersionStr L"version"
#define TimestampUTCStr L"timestampUTC"
#define StatusStr L"status"
#define NameStr L"name"
#define OperationStr L"operation"
#define ConfigurationAppliedTimeStr L"configurationAppliedTime"
#define CodeStr L"code"
#define IdStr L"id"
#define ParamsStr L"params"
#define MessageStr L"message"
#define FormattedMessageStr L"formattedMessage"
#define LangStr L"lang"
#define SubstatusStr L"substatus"

using namespace std;
using namespace web;

class Message
{
public:
    wstring id;
    vector<wstring> params;
};

class FormattedMessage
{
public:
    wstring lang;
    wstring message;
};

class Substatus
{
public:
    wstring name;
    wstring status;
    int code;
    Message message;
    FormattedMessage formattedMessage;
};

class Status
{
public:
    wstring name;
    wstring operation;
    wstring configurationAppliedTime;
    wstring status;
    int code;
    Message message;
    FormattedMessage formattedMessage;
    vector<Substatus> substatus;
};

class StatusHandler
{
public:
    wstring version;
    wstring timestampUTC;
    Status status;

    json::value ToJson();
    static void WriteExtensionStatus(char *fileName, vector<json::value> statusHandlers);
};