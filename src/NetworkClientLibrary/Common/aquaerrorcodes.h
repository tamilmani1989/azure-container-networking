/****************************** Module Header ******************************\
* Module Name:  aquaerrorcodes.h
* Project:      Aquarius
* Copyright (c) Microsoft Corporation.
*
* Header to be consumed by partners.
* Contains external contract.
*
\***************************************************************************/

#pragma once

// 0 indicates success
unsigned long UNSUPPORTED_NETWORK = 1;
unsigned long INVALID_PARAMETER = 2;
unsigned long UNSUPPORTED_ENVIRONMENT = 3;
unsigned long UNREACHABLE_HOST = 4;
unsigned long RESERVATION_NOT_FOUND = 5;
unsigned long OVERLAYPLUGIN_INSTALL_FAILED = 6;
unsigned long OVERLAY_SUBNET_NOT_SPECIFIED = 7;
unsigned long MALFORMED_SUBNET_IP = 8;
unsigned long UNREACHABLE_DOCKER_DAEMON = 9;
unsigned long UNSPECIFIED_NETWORK_NAME = 10;
unsigned long SUBNET_SIZE_TOO_SMALL = 11;
unsigned long UNEXPECTED_ERROR = 99;
