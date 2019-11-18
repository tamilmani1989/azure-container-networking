#include <stdlib.h>
#include <stdio.h>
#include "log.h"

bool LogCreated = false;

#include <stdio.h>
#include <stdarg.h>
#include <time.h>
#include <string.h>
#include <stdlib.h>
#include "log.h"

FILE *fp;
static bool SESSION_TRACKER = false; //Keeps track of session

void log_print(const char *fmt, ...)
{
    char *buffer;
    errno_t err;
    va_list args;
    va_start(args, fmt);

    // +1 is being used for storing string termination \0 character.
    int len = _vscprintf( fmt, args ) + 1;

    if(len == 1)
    {
        return;
    }

    buffer = (char *) malloc( len * sizeof(char) );

    if ( NULL != buffer )
    {
        vsprintf_s(buffer, len, fmt, args);
    }
    else
    {
        return;
    }
    
    va_end(args);

    if (SESSION_TRACKER)
    {
        err = fopen_s(&fp, LOGFILE, "a+");
    }
    else
    {
        err = fopen_s(&fp, LOGFILE, "w");
    }

    if(err != 0)
    {
        return;
    }

    SESSION_TRACKER = true;

    time_t result = time(NULL);
    char str[26];
    ctime_s(str, sizeof str, &result);
    fprintf(fp, "%s %s\n", str, buffer);
    fclose(fp);
    free(buffer);
}