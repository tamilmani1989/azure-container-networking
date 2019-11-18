#pragma once
#define LOGFILE	"parser.log"     // all Log(); messages will be appended to this file

extern bool LogCreated;      // keeps track whether the log file is created or not

void log_print(const char *fmt, ...);
#define Log(...) log_print(__VA_ARGS__ )