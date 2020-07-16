package MadariClient

/*
#include <stdlib.h>
#include <iconv.h>
#include <string.h>
#include <wchar.h>

void resetWcharAt(char *buffer, int at, int size) {
    memset(buffer + at, 0, size);
}
*/
import "C"

import (
    "fmt"
    "unsafe"
)

func newWcharBuffer(length int) *C.char {
    return (*C.char)(C.malloc(C.size_t(length)))
}

func DeleteWcharBuffer(buffer *C.wchar_t) {
    if nil != buffer {
        C.free(unsafe.Pointer(buffer))
    }
}

func WcharStringPtrToGoString(first unsafe.Pointer) string {
    if 0x0 == uintptr(first) {
        return ""
    }

    length := C.wcslen((*C.wchar_t)(first)) * bytesPerWchar
    if 0 == length {
        return ""
    }

    iconv, errno := C.iconv_open(iconvCharsetUtf8, iconvCharsetWchar)
    if iconv == nil || errno != nil {
        panic(fmt.Errorf("Could not open iconv instance: %s", errno.Error()))
    }
    defer C.iconv_close(iconv)

    inputChars := (*C.char)(first)
    bytesLeftInCSize := length
    outputChars := (*C.char)(C.malloc(length))
    defer C.free(unsafe.Pointer(outputChars))
    bytesLeftOutCSize := length

    saveOutputChars := outputChars
    _, errno = C.iconv(iconv, &inputChars, &bytesLeftInCSize, &outputChars, &bytesLeftOutCSize)
    if errno != nil {
        panic(errno)
    }
    outputChars = saveOutputChars

    return C.GoStringN(saveOutputChars, (C.int)(length - bytesLeftOutCSize))
}

func GoStringToWcharStringPtr(input string) *C.wchar_t {
    iconv, errno := C.iconv_open(iconvCharsetWchar, iconvCharsetUtf8)
    if iconv == nil || errno != nil {
        panic(fmt.Errorf("Could not open iconv instance: %s", errno.Error()))
    }
    defer C.iconv_close(iconv)

    inputCString := C.CString(input)
    defer C.free(unsafe.Pointer(inputCString))
    bytesLeftInCSize := C.strlen(inputCString)

    inputLength := len(input) * bytesPerWchar
    outputCString := newWcharBuffer(inputLength + bytesPerWchar)
    bytesLeftOutCSize := C.size_t(inputLength)

    saveInputCString, saveOutputCString := inputCString, outputCString
    _, errno = C.iconv(iconv, &inputCString, &bytesLeftInCSize, &outputCString, &bytesLeftOutCSize)
    if errno != nil {
        panic(errno)
    }
    inputCString, outputCString = saveInputCString, saveOutputCString

    C.resetWcharAt(outputCString, C.int(inputLength - int(bytesLeftOutCSize)), bytesPerWchar)

    return (*C.wchar_t)(unsafe.Pointer(outputCString))
}

var (
    iconvCharsetWchar = C.CString("wchar_t//TRANSLIT")
    iconvCharsetChar  = C.CString("//TRANSLIT")
    iconvCharsetAscii = C.CString("ascii//TRANSLIT")
    iconvCharsetUtf8  = C.CString("utf-8//TRANSLIT")
)
