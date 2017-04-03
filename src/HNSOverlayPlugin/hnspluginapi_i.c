

/* this ALWAYS GENERATED file contains the IIDs and CLSIDs */

/* link this file in with the server and any clients */


 /* File created by MIDL compiler version 8.01.0618 */
/* @@MIDL_FILE_HEADING(  ) */



#ifdef __cplusplus
extern "C"{
#endif 


#include <rpc.h>
#include <rpcndr.h>

#ifdef _MIDL_USE_GUIDDEF_

#ifndef INITGUID
#define INITGUID
#include <guiddef.h>
#undef INITGUID
#else
#include <guiddef.h>
#endif

#define MIDL_DEFINE_GUID(type,name,l,w1,w2,b1,b2,b3,b4,b5,b6,b7,b8) \
        DEFINE_GUID(name,l,w1,w2,b1,b2,b3,b4,b5,b6,b7,b8)

#else // !_MIDL_USE_GUIDDEF_

#ifndef __IID_DEFINED__
#define __IID_DEFINED__

typedef struct _IID
{
    unsigned long x;
    unsigned short s1;
    unsigned short s2;
    unsigned char  c[8];
} IID;

#endif // __IID_DEFINED__

#ifndef CLSID_DEFINED
#define CLSID_DEFINED
typedef IID CLSID;
#endif // CLSID_DEFINED

#define MIDL_DEFINE_GUID(type,name,l,w1,w2,b1,b2,b3,b4,b5,b6,b7,b8) \
        const type name = {l,w1,w2,{b1,b2,b3,b4,b5,b6,b7,b8}}

#endif !_MIDL_USE_GUIDDEF_

MIDL_DEFINE_GUID(IID, IID_IHNSObject,0xdbf70581,0xefeb,0x4936,0xb5,0x7b,0x34,0x7b,0x74,0x36,0x3b,0xa8);


MIDL_DEFINE_GUID(IID, IID_IHNSObjectList,0xdbf70581,0xefeb,0x4936,0xb5,0x7b,0x34,0x7b,0x74,0x36,0x3b,0xa9);


MIDL_DEFINE_GUID(IID, IID_IHNSProvider,0x9292e69b,0x21ad,0x400e,0x93,0x57,0x9e,0x36,0x6d,0x35,0xd2,0xa0);


MIDL_DEFINE_GUID(IID, IID_IHNSPlugin,0x398a46e6,0x572e,0x4d19,0x84,0x76,0x1c,0xab,0xa8,0x46,0x28,0x94);

#undef MIDL_DEFINE_GUID

#ifdef __cplusplus
}
#endif



