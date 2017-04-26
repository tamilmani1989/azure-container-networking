

/* this ALWAYS GENERATED file contains the proxy stub code */


 /* File created by MIDL compiler version 8.01.0618 */
/* @@MIDL_FILE_HEADING(  ) */

#if defined(_M_AMD64)


#if _MSC_VER >= 1200
#pragma warning(push)
#endif

#pragma warning( disable: 4211 )  /* redefine extern to static */
#pragma warning( disable: 4232 )  /* dllimport identity*/
#pragma warning( disable: 4024 )  /* array to pointer mapping*/
#pragma warning( disable: 4152 )  /* function/data pointer conversion in expression */

#define USE_STUBLESS_PROXY


/* verify that the <rpcproxy.h> version is high enough to compile this file*/
#ifndef __REDQ_RPCPROXY_H_VERSION__
#define __REQUIRED_RPCPROXY_H_VERSION__ 475
#endif


#include "rpcproxy.h"
#include "ndr64types.h"
#ifndef __RPCPROXY_H_VERSION__
#error this stub requires an updated version of <rpcproxy.h>
#endif /* __RPCPROXY_H_VERSION__ */


#include "hnspluginapi.h"

#define TYPE_FORMAT_STRING_SIZE   111                               
#define PROC_FORMAT_STRING_SIZE   1671                              
#define EXPR_FORMAT_STRING_SIZE   1                                 
#define TRANSMIT_AS_TABLE_SIZE    0            
#define WIRE_MARSHAL_TABLE_SIZE   0            

typedef struct _hnspluginapi_MIDL_TYPE_FORMAT_STRING
    {
    short          Pad;
    unsigned char  Format[ TYPE_FORMAT_STRING_SIZE ];
    } hnspluginapi_MIDL_TYPE_FORMAT_STRING;

typedef struct _hnspluginapi_MIDL_PROC_FORMAT_STRING
    {
    short          Pad;
    unsigned char  Format[ PROC_FORMAT_STRING_SIZE ];
    } hnspluginapi_MIDL_PROC_FORMAT_STRING;

typedef struct _hnspluginapi_MIDL_EXPR_FORMAT_STRING
    {
    long          Pad;
    unsigned char  Format[ EXPR_FORMAT_STRING_SIZE ];
    } hnspluginapi_MIDL_EXPR_FORMAT_STRING;


static const RPC_SYNTAX_IDENTIFIER  _RpcTransferSyntax = 
{{0x8A885D04,0x1CEB,0x11C9,{0x9F,0xE8,0x08,0x00,0x2B,0x10,0x48,0x60}},{2,0}};

static const RPC_SYNTAX_IDENTIFIER  _NDR64_RpcTransferSyntax = 
{{0x71710533,0xbeba,0x4937,{0x83,0x19,0xb5,0xdb,0xef,0x9c,0xcc,0x36}},{1,0}};



extern const hnspluginapi_MIDL_TYPE_FORMAT_STRING hnspluginapi__MIDL_TypeFormatString;
extern const hnspluginapi_MIDL_PROC_FORMAT_STRING hnspluginapi__MIDL_ProcFormatString;
extern const hnspluginapi_MIDL_EXPR_FORMAT_STRING hnspluginapi__MIDL_ExprFormatString;


extern const MIDL_STUB_DESC Object_StubDesc;


extern const MIDL_SERVER_INFO IHNSObject_ServerInfo;
extern const MIDL_STUBLESS_PROXY_INFO IHNSObject_ProxyInfo;


extern const MIDL_STUB_DESC Object_StubDesc;


extern const MIDL_SERVER_INFO IHNSObjectList_ServerInfo;
extern const MIDL_STUBLESS_PROXY_INFO IHNSObjectList_ProxyInfo;


extern const MIDL_STUB_DESC Object_StubDesc;


extern const MIDL_SERVER_INFO IHNSProvider_ServerInfo;
extern const MIDL_STUBLESS_PROXY_INFO IHNSProvider_ProxyInfo;


extern const MIDL_STUB_DESC Object_StubDesc;


extern const MIDL_SERVER_INFO IHNSPlugin_ServerInfo;
extern const MIDL_STUBLESS_PROXY_INFO IHNSPlugin_ProxyInfo;



#if !defined(__RPC_WIN64__)
#error  Invalid build platform for this stub.
#endif

static const hnspluginapi_MIDL_PROC_FORMAT_STRING hnspluginapi__MIDL_ProcFormatString =
    {
        0,
        {

	/* Procedure IsEqualId */

			0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/*  2 */	NdrFcLong( 0x0 ),	/* 0 */
/*  6 */	NdrFcShort( 0x3 ),	/* 3 */
/*  8 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 10 */	NdrFcShort( 0x44 ),	/* 68 */
/* 12 */	NdrFcShort( 0x8 ),	/* 8 */
/* 14 */	0x44,		/* Oi2 Flags:  has return, has ext, */
			0x2,		/* 2 */
/* 16 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 18 */	NdrFcShort( 0x0 ),	/* 0 */
/* 20 */	NdrFcShort( 0x0 ),	/* 0 */
/* 22 */	NdrFcShort( 0x0 ),	/* 0 */
/* 24 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter CheckGuid */

/* 26 */	NdrFcShort( 0x10a ),	/* Flags:  must free, in, simple ref, */
/* 28 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 30 */	NdrFcShort( 0xc ),	/* Type Offset=12 */

	/* Return value */

/* 32 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 34 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 36 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure IsEqualType */

/* 38 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 40 */	NdrFcLong( 0x0 ),	/* 0 */
/* 44 */	NdrFcShort( 0x4 ),	/* 4 */
/* 46 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 48 */	NdrFcShort( 0x6 ),	/* 6 */
/* 50 */	NdrFcShort( 0x8 ),	/* 8 */
/* 52 */	0x44,		/* Oi2 Flags:  has return, has ext, */
			0x2,		/* 2 */
/* 54 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 56 */	NdrFcShort( 0x0 ),	/* 0 */
/* 58 */	NdrFcShort( 0x0 ),	/* 0 */
/* 60 */	NdrFcShort( 0x0 ),	/* 0 */
/* 62 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter ObjectType */

/* 64 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 66 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 68 */	0xd,		/* FC_ENUM16 */
			0x0,		/* 0 */

	/* Return value */

/* 70 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 72 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 74 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure PropertyExists */

/* 76 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 78 */	NdrFcLong( 0x0 ),	/* 0 */
/* 82 */	NdrFcShort( 0x5 ),	/* 5 */
/* 84 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 86 */	NdrFcShort( 0x0 ),	/* 0 */
/* 88 */	NdrFcShort( 0x8 ),	/* 8 */
/* 90 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x2,		/* 2 */
/* 92 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 94 */	NdrFcShort( 0x0 ),	/* 0 */
/* 96 */	NdrFcShort( 0x0 ),	/* 0 */
/* 98 */	NdrFcShort( 0x0 ),	/* 0 */
/* 100 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 102 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 104 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 106 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Return value */

/* 108 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 110 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 112 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure PropertyIsType */

/* 114 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 116 */	NdrFcLong( 0x0 ),	/* 0 */
/* 120 */	NdrFcShort( 0x6 ),	/* 6 */
/* 122 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 124 */	NdrFcShort( 0x6 ),	/* 6 */
/* 126 */	NdrFcShort( 0x8 ),	/* 8 */
/* 128 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 130 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 132 */	NdrFcShort( 0x0 ),	/* 0 */
/* 134 */	NdrFcShort( 0x0 ),	/* 0 */
/* 136 */	NdrFcShort( 0x0 ),	/* 0 */
/* 138 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 140 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 142 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 144 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter ExpectedType */

/* 146 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 148 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 150 */	0xd,		/* FC_ENUM16 */
			0x0,		/* 0 */

	/* Return value */

/* 152 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 154 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 156 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure IsClientProperty */

/* 158 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 160 */	NdrFcLong( 0x0 ),	/* 0 */
/* 164 */	NdrFcShort( 0x7 ),	/* 7 */
/* 166 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 168 */	NdrFcShort( 0x0 ),	/* 0 */
/* 170 */	NdrFcShort( 0x8 ),	/* 8 */
/* 172 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x2,		/* 2 */
/* 174 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 176 */	NdrFcShort( 0x0 ),	/* 0 */
/* 178 */	NdrFcShort( 0x0 ),	/* 0 */
/* 180 */	NdrFcShort( 0x0 ),	/* 0 */
/* 182 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 184 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 186 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 188 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Return value */

/* 190 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 192 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 194 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetObjectId */

/* 196 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 198 */	NdrFcLong( 0x0 ),	/* 0 */
/* 202 */	NdrFcShort( 0x8 ),	/* 8 */
/* 204 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 206 */	NdrFcShort( 0x44 ),	/* 68 */
/* 208 */	NdrFcShort( 0x8 ),	/* 8 */
/* 210 */	0x44,		/* Oi2 Flags:  has return, has ext, */
			0x2,		/* 2 */
/* 212 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 214 */	NdrFcShort( 0x0 ),	/* 0 */
/* 216 */	NdrFcShort( 0x0 ),	/* 0 */
/* 218 */	NdrFcShort( 0x0 ),	/* 0 */
/* 220 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter ObjectId */

/* 222 */	NdrFcShort( 0x10a ),	/* Flags:  must free, in, simple ref, */
/* 224 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 226 */	NdrFcShort( 0xc ),	/* Type Offset=12 */

	/* Return value */

/* 228 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 230 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 232 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetObjectType */

/* 234 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 236 */	NdrFcLong( 0x0 ),	/* 0 */
/* 240 */	NdrFcShort( 0x9 ),	/* 9 */
/* 242 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 244 */	NdrFcShort( 0x6 ),	/* 6 */
/* 246 */	NdrFcShort( 0x8 ),	/* 8 */
/* 248 */	0x44,		/* Oi2 Flags:  has return, has ext, */
			0x2,		/* 2 */
/* 250 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 252 */	NdrFcShort( 0x0 ),	/* 0 */
/* 254 */	NdrFcShort( 0x0 ),	/* 0 */
/* 256 */	NdrFcShort( 0x0 ),	/* 0 */
/* 258 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter ObjectType */

/* 260 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 262 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 264 */	0xd,		/* FC_ENUM16 */
			0x0,		/* 0 */

	/* Return value */

/* 266 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 268 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 270 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetPropertyVisible */

/* 272 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 274 */	NdrFcLong( 0x0 ),	/* 0 */
/* 278 */	NdrFcShort( 0xa ),	/* 10 */
/* 280 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 282 */	NdrFcShort( 0x5 ),	/* 5 */
/* 284 */	NdrFcShort( 0x8 ),	/* 8 */
/* 286 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 288 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 290 */	NdrFcShort( 0x0 ),	/* 0 */
/* 292 */	NdrFcShort( 0x0 ),	/* 0 */
/* 294 */	NdrFcShort( 0x0 ),	/* 0 */
/* 296 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 298 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 300 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 302 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter Visible */

/* 304 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 306 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 308 */	0x3,		/* FC_SMALL */
			0x0,		/* 0 */

	/* Return value */

/* 310 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 312 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 314 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetPropertyPersistent */

/* 316 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 318 */	NdrFcLong( 0x0 ),	/* 0 */
/* 322 */	NdrFcShort( 0xb ),	/* 11 */
/* 324 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 326 */	NdrFcShort( 0x5 ),	/* 5 */
/* 328 */	NdrFcShort( 0x8 ),	/* 8 */
/* 330 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 332 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 334 */	NdrFcShort( 0x0 ),	/* 0 */
/* 336 */	NdrFcShort( 0x0 ),	/* 0 */
/* 338 */	NdrFcShort( 0x0 ),	/* 0 */
/* 340 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 342 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 344 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 346 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter Persistent */

/* 348 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 350 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 352 */	0x3,		/* FC_SMALL */
			0x0,		/* 0 */

	/* Return value */

/* 354 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 356 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 358 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure DuplicateObject */

/* 360 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 362 */	NdrFcLong( 0x0 ),	/* 0 */
/* 366 */	NdrFcShort( 0xc ),	/* 12 */
/* 368 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 370 */	NdrFcShort( 0x0 ),	/* 0 */
/* 372 */	NdrFcShort( 0x8 ),	/* 8 */
/* 374 */	0x45,		/* Oi2 Flags:  srv must size, has return, has ext, */
			0x2,		/* 2 */
/* 376 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 378 */	NdrFcShort( 0x0 ),	/* 0 */
/* 380 */	NdrFcShort( 0x0 ),	/* 0 */
/* 382 */	NdrFcShort( 0x0 ),	/* 0 */
/* 384 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter OutputObject */

/* 386 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 388 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 390 */	NdrFcShort( 0x1c ),	/* Type Offset=28 */

	/* Return value */

/* 392 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 394 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 396 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure PropertyCount */

/* 398 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 400 */	NdrFcLong( 0x0 ),	/* 0 */
/* 404 */	NdrFcShort( 0xd ),	/* 13 */
/* 406 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 408 */	NdrFcShort( 0x0 ),	/* 0 */
/* 410 */	NdrFcShort( 0x24 ),	/* 36 */
/* 412 */	0x44,		/* Oi2 Flags:  has return, has ext, */
			0x2,		/* 2 */
/* 414 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 416 */	NdrFcShort( 0x0 ),	/* 0 */
/* 418 */	NdrFcShort( 0x0 ),	/* 0 */
/* 420 */	NdrFcShort( 0x0 ),	/* 0 */
/* 422 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter Count */

/* 424 */	NdrFcShort( 0x2150 ),	/* Flags:  out, base type, simple ref, srv alloc size=8 */
/* 426 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 428 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Return value */

/* 430 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 432 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 434 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure GetPropertyName */

/* 436 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 438 */	NdrFcLong( 0x0 ),	/* 0 */
/* 442 */	NdrFcShort( 0xe ),	/* 14 */
/* 444 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 446 */	NdrFcShort( 0x8 ),	/* 8 */
/* 448 */	NdrFcShort( 0x8 ),	/* 8 */
/* 450 */	0x45,		/* Oi2 Flags:  srv must size, has return, has ext, */
			0x3,		/* 3 */
/* 452 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 454 */	NdrFcShort( 0x0 ),	/* 0 */
/* 456 */	NdrFcShort( 0x0 ),	/* 0 */
/* 458 */	NdrFcShort( 0x0 ),	/* 0 */
/* 460 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter Index */

/* 462 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 464 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 466 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Parameter PropertyName */

/* 468 */	NdrFcShort( 0x2013 ),	/* Flags:  must size, must free, out, srv alloc size=8 */
/* 470 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 472 */	NdrFcShort( 0x36 ),	/* Type Offset=54 */

	/* Return value */

/* 474 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 476 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 478 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetGuid */

/* 480 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 482 */	NdrFcLong( 0x0 ),	/* 0 */
/* 486 */	NdrFcShort( 0xf ),	/* 15 */
/* 488 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 490 */	NdrFcShort( 0x44 ),	/* 68 */
/* 492 */	NdrFcShort( 0x8 ),	/* 8 */
/* 494 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 496 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 498 */	NdrFcShort( 0x0 ),	/* 0 */
/* 500 */	NdrFcShort( 0x0 ),	/* 0 */
/* 502 */	NdrFcShort( 0x0 ),	/* 0 */
/* 504 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 506 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 508 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 510 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter GuidValue */

/* 512 */	NdrFcShort( 0x10a ),	/* Flags:  must free, in, simple ref, */
/* 514 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 516 */	NdrFcShort( 0xc ),	/* Type Offset=12 */

	/* Return value */

/* 518 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 520 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 522 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetInteger */

/* 524 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 526 */	NdrFcLong( 0x0 ),	/* 0 */
/* 530 */	NdrFcShort( 0x10 ),	/* 16 */
/* 532 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 534 */	NdrFcShort( 0x8 ),	/* 8 */
/* 536 */	NdrFcShort( 0x8 ),	/* 8 */
/* 538 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 540 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 542 */	NdrFcShort( 0x0 ),	/* 0 */
/* 544 */	NdrFcShort( 0x0 ),	/* 0 */
/* 546 */	NdrFcShort( 0x0 ),	/* 0 */
/* 548 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 550 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 552 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 554 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter IntegerValue */

/* 556 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 558 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 560 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Return value */

/* 562 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 564 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 566 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetBoolean */

/* 568 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 570 */	NdrFcLong( 0x0 ),	/* 0 */
/* 574 */	NdrFcShort( 0x11 ),	/* 17 */
/* 576 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 578 */	NdrFcShort( 0x5 ),	/* 5 */
/* 580 */	NdrFcShort( 0x8 ),	/* 8 */
/* 582 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 584 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 586 */	NdrFcShort( 0x0 ),	/* 0 */
/* 588 */	NdrFcShort( 0x0 ),	/* 0 */
/* 590 */	NdrFcShort( 0x0 ),	/* 0 */
/* 592 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 594 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 596 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 598 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter BoolValue */

/* 600 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 602 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 604 */	0x3,		/* FC_SMALL */
			0x0,		/* 0 */

	/* Return value */

/* 606 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 608 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 610 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetString */

/* 612 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 614 */	NdrFcLong( 0x0 ),	/* 0 */
/* 618 */	NdrFcShort( 0x12 ),	/* 18 */
/* 620 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 622 */	NdrFcShort( 0x0 ),	/* 0 */
/* 624 */	NdrFcShort( 0x8 ),	/* 8 */
/* 626 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 628 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 630 */	NdrFcShort( 0x0 ),	/* 0 */
/* 632 */	NdrFcShort( 0x0 ),	/* 0 */
/* 634 */	NdrFcShort( 0x0 ),	/* 0 */
/* 636 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 638 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 640 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 642 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter StringValue */

/* 644 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 646 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 648 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Return value */

/* 650 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 652 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 654 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetObject */

/* 656 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 658 */	NdrFcLong( 0x0 ),	/* 0 */
/* 662 */	NdrFcShort( 0x13 ),	/* 19 */
/* 664 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 666 */	NdrFcShort( 0x0 ),	/* 0 */
/* 668 */	NdrFcShort( 0x8 ),	/* 8 */
/* 670 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 672 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 674 */	NdrFcShort( 0x0 ),	/* 0 */
/* 676 */	NdrFcShort( 0x0 ),	/* 0 */
/* 678 */	NdrFcShort( 0x0 ),	/* 0 */
/* 680 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 682 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 684 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 686 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter Object */

/* 688 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 690 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 692 */	NdrFcShort( 0x20 ),	/* Type Offset=32 */

	/* Return value */

/* 694 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 696 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 698 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure SetObjectList */

/* 700 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 702 */	NdrFcLong( 0x0 ),	/* 0 */
/* 706 */	NdrFcShort( 0x14 ),	/* 20 */
/* 708 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 710 */	NdrFcShort( 0x0 ),	/* 0 */
/* 712 */	NdrFcShort( 0x8 ),	/* 8 */
/* 714 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x2,		/* 2 */
/* 716 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 718 */	NdrFcShort( 0x0 ),	/* 0 */
/* 720 */	NdrFcShort( 0x0 ),	/* 0 */
/* 722 */	NdrFcShort( 0x0 ),	/* 0 */
/* 724 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 726 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 728 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 730 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Return value */

/* 732 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 734 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 736 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure AddListObject */

/* 738 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 740 */	NdrFcLong( 0x0 ),	/* 0 */
/* 744 */	NdrFcShort( 0x15 ),	/* 21 */
/* 746 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 748 */	NdrFcShort( 0x0 ),	/* 0 */
/* 750 */	NdrFcShort( 0x8 ),	/* 8 */
/* 752 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 754 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 756 */	NdrFcShort( 0x0 ),	/* 0 */
/* 758 */	NdrFcShort( 0x0 ),	/* 0 */
/* 760 */	NdrFcShort( 0x0 ),	/* 0 */
/* 762 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 764 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 766 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 768 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter Object */

/* 770 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 772 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 774 */	NdrFcShort( 0x20 ),	/* Type Offset=32 */

	/* Return value */

/* 776 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 778 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 780 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure RemoveListObject */

/* 782 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 784 */	NdrFcLong( 0x0 ),	/* 0 */
/* 788 */	NdrFcShort( 0x16 ),	/* 22 */
/* 790 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 792 */	NdrFcShort( 0x0 ),	/* 0 */
/* 794 */	NdrFcShort( 0x8 ),	/* 8 */
/* 796 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 798 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 800 */	NdrFcShort( 0x0 ),	/* 0 */
/* 802 */	NdrFcShort( 0x0 ),	/* 0 */
/* 804 */	NdrFcShort( 0x0 ),	/* 0 */
/* 806 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 808 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 810 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 812 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter Object */

/* 814 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 816 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 818 */	NdrFcShort( 0x20 ),	/* Type Offset=32 */

	/* Return value */

/* 820 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 822 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 824 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure GetGuid */

/* 826 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 828 */	NdrFcLong( 0x0 ),	/* 0 */
/* 832 */	NdrFcShort( 0x17 ),	/* 23 */
/* 834 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 836 */	NdrFcShort( 0x0 ),	/* 0 */
/* 838 */	NdrFcShort( 0x4c ),	/* 76 */
/* 840 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 842 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 844 */	NdrFcShort( 0x0 ),	/* 0 */
/* 846 */	NdrFcShort( 0x0 ),	/* 0 */
/* 848 */	NdrFcShort( 0x0 ),	/* 0 */
/* 850 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 852 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 854 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 856 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter GuidValue */

/* 858 */	NdrFcShort( 0x4112 ),	/* Flags:  must free, out, simple ref, srv alloc size=16 */
/* 860 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 862 */	NdrFcShort( 0xc ),	/* Type Offset=12 */

	/* Return value */

/* 864 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 866 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 868 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure GetInteger */

/* 870 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 872 */	NdrFcLong( 0x0 ),	/* 0 */
/* 876 */	NdrFcShort( 0x18 ),	/* 24 */
/* 878 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 880 */	NdrFcShort( 0x0 ),	/* 0 */
/* 882 */	NdrFcShort( 0x24 ),	/* 36 */
/* 884 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 886 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 888 */	NdrFcShort( 0x0 ),	/* 0 */
/* 890 */	NdrFcShort( 0x0 ),	/* 0 */
/* 892 */	NdrFcShort( 0x0 ),	/* 0 */
/* 894 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 896 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 898 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 900 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter IntegerValue */

/* 902 */	NdrFcShort( 0x2150 ),	/* Flags:  out, base type, simple ref, srv alloc size=8 */
/* 904 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 906 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Return value */

/* 908 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 910 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 912 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure GetBoolean */

/* 914 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 916 */	NdrFcLong( 0x0 ),	/* 0 */
/* 920 */	NdrFcShort( 0x19 ),	/* 25 */
/* 922 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 924 */	NdrFcShort( 0x0 ),	/* 0 */
/* 926 */	NdrFcShort( 0x21 ),	/* 33 */
/* 928 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 930 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 932 */	NdrFcShort( 0x0 ),	/* 0 */
/* 934 */	NdrFcShort( 0x0 ),	/* 0 */
/* 936 */	NdrFcShort( 0x0 ),	/* 0 */
/* 938 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 940 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 942 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 944 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter BooleanValue */

/* 946 */	NdrFcShort( 0x2150 ),	/* Flags:  out, base type, simple ref, srv alloc size=8 */
/* 948 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 950 */	0x3,		/* FC_SMALL */
			0x0,		/* 0 */

	/* Return value */

/* 952 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 954 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 956 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure GetString */

/* 958 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 960 */	NdrFcLong( 0x0 ),	/* 0 */
/* 964 */	NdrFcShort( 0x1a ),	/* 26 */
/* 966 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 968 */	NdrFcShort( 0x0 ),	/* 0 */
/* 970 */	NdrFcShort( 0x8 ),	/* 8 */
/* 972 */	0x47,		/* Oi2 Flags:  srv must size, clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 974 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 976 */	NdrFcShort( 0x0 ),	/* 0 */
/* 978 */	NdrFcShort( 0x0 ),	/* 0 */
/* 980 */	NdrFcShort( 0x0 ),	/* 0 */
/* 982 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 984 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 986 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 988 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter StringValue */

/* 990 */	NdrFcShort( 0x2013 ),	/* Flags:  must size, must free, out, srv alloc size=8 */
/* 992 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 994 */	NdrFcShort( 0x36 ),	/* Type Offset=54 */

	/* Return value */

/* 996 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 998 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1000 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure GetObject */

/* 1002 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1004 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1008 */	NdrFcShort( 0x1b ),	/* 27 */
/* 1010 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1012 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1014 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1016 */	0x47,		/* Oi2 Flags:  srv must size, clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 1018 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1020 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1022 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1024 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1026 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 1028 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 1030 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1032 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter OutObject */

/* 1034 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1036 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1038 */	NdrFcShort( 0x1c ),	/* Type Offset=28 */

	/* Return value */

/* 1040 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1042 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1044 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure GetObjectList */

/* 1046 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1048 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1052 */	NdrFcShort( 0x1c ),	/* 28 */
/* 1054 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1056 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1058 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1060 */	0x47,		/* Oi2 Flags:  srv must size, clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 1062 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1064 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1066 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1068 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1070 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 1072 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 1074 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1076 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter OutObjectList */

/* 1078 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1080 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1082 */	NdrFcShort( 0x46 ),	/* Type Offset=70 */

	/* Return value */

/* 1084 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1086 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1088 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure DeleteProperty */

/* 1090 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1092 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1096 */	NdrFcShort( 0x1d ),	/* 29 */
/* 1098 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1100 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1102 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1104 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x2,		/* 2 */
/* 1106 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1108 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1110 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1112 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1114 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PropertyName */

/* 1116 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 1118 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1120 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Return value */

/* 1122 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1124 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1126 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure ObjectCount */

/* 1128 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1130 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1134 */	NdrFcShort( 0x3 ),	/* 3 */
/* 1136 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1138 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1140 */	NdrFcShort( 0x24 ),	/* 36 */
/* 1142 */	0x44,		/* Oi2 Flags:  has return, has ext, */
			0x2,		/* 2 */
/* 1144 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1146 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1148 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1150 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1152 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter Count */

/* 1154 */	NdrFcShort( 0x2150 ),	/* Flags:  out, base type, simple ref, srv alloc size=8 */
/* 1156 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1158 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Return value */

/* 1160 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1162 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1164 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure GetObject */

/* 1166 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1168 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1172 */	NdrFcShort( 0x4 ),	/* 4 */
/* 1174 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1176 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1178 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1180 */	0x45,		/* Oi2 Flags:  srv must size, has return, has ext, */
			0x3,		/* 3 */
/* 1182 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1184 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1186 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1188 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1190 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter ObjectIndex */

/* 1192 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 1194 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1196 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Parameter OutObject */

/* 1198 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1200 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1202 */	NdrFcShort( 0x1c ),	/* Type Offset=28 */

	/* Return value */

/* 1204 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1206 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1208 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure EnumerateNetworks */

/* 1210 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1212 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1216 */	NdrFcShort( 0x3 ),	/* 3 */
/* 1218 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1220 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1222 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1224 */	0x45,		/* Oi2 Flags:  srv must size, has return, has ext, */
			0x2,		/* 2 */
/* 1226 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1228 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1230 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1232 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1234 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter OutNetworks */

/* 1236 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1238 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1240 */	NdrFcShort( 0x46 ),	/* Type Offset=70 */

	/* Return value */

/* 1242 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1244 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1246 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure EnumerateEndpoints */

/* 1248 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1250 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1254 */	NdrFcShort( 0x4 ),	/* 4 */
/* 1256 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1258 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1260 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1262 */	0x45,		/* Oi2 Flags:  srv must size, has return, has ext, */
			0x2,		/* 2 */
/* 1264 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1266 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1268 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1270 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1272 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter OutEndpoints */

/* 1274 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1276 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1278 */	NdrFcShort( 0x46 ),	/* Type Offset=70 */

	/* Return value */

/* 1280 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1282 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1284 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure CreateObject */

/* 1286 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1288 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1292 */	NdrFcShort( 0x5 ),	/* 5 */
/* 1294 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1296 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1298 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1300 */	0x45,		/* Oi2 Flags:  srv must size, has return, has ext, */
			0x2,		/* 2 */
/* 1302 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1304 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1306 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1308 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1310 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter OutNewObject */

/* 1312 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1314 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1316 */	NdrFcShort( 0x1c ),	/* Type Offset=28 */

	/* Return value */

/* 1318 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1320 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1322 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure CreateNetwork */

/* 1324 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1326 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1330 */	NdrFcShort( 0x6 ),	/* 6 */
/* 1332 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1334 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1336 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1338 */	0x47,		/* Oi2 Flags:  srv must size, clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 1340 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1342 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1344 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1346 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1348 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter InputObject */

/* 1350 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 1352 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1354 */	NdrFcShort( 0x20 ),	/* Type Offset=32 */

	/* Parameter CreatedNetwork */

/* 1356 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1358 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1360 */	NdrFcShort( 0x1c ),	/* Type Offset=28 */

	/* Return value */

/* 1362 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1364 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1366 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure CreateEndpoint */

/* 1368 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1370 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1374 */	NdrFcShort( 0x7 ),	/* 7 */
/* 1376 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1378 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1380 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1382 */	0x47,		/* Oi2 Flags:  srv must size, clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 1384 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1386 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1388 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1390 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1392 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter InputObject */

/* 1394 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 1396 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1398 */	NdrFcShort( 0x20 ),	/* Type Offset=32 */

	/* Parameter CreatedEndpoint */

/* 1400 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1402 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1404 */	NdrFcShort( 0x1c ),	/* Type Offset=28 */

	/* Return value */

/* 1406 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1408 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1410 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure FindNetworkByName */

/* 1412 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1414 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1418 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1420 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1422 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1424 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1426 */	0x47,		/* Oi2 Flags:  srv must size, clt must size, has return, has ext, */
			0x3,		/* 3 */
/* 1428 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1430 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1432 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1434 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1436 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter NetworkName */

/* 1438 */	NdrFcShort( 0x10b ),	/* Flags:  must size, must free, in, simple ref, */
/* 1440 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1442 */	NdrFcShort( 0x1a ),	/* Type Offset=26 */

	/* Parameter FoundObject */

/* 1444 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1446 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1448 */	NdrFcShort( 0x1c ),	/* Type Offset=28 */

	/* Return value */

/* 1450 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1452 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1454 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure FindObject */

/* 1456 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1458 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1462 */	NdrFcShort( 0x9 ),	/* 9 */
/* 1464 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1466 */	NdrFcShort( 0x44 ),	/* 68 */
/* 1468 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1470 */	0x45,		/* Oi2 Flags:  srv must size, has return, has ext, */
			0x3,		/* 3 */
/* 1472 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1474 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1476 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1478 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1480 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter ObjectGuid */

/* 1482 */	NdrFcShort( 0x10a ),	/* Flags:  must free, in, simple ref, */
/* 1484 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1486 */	NdrFcShort( 0xc ),	/* Type Offset=12 */

	/* Parameter FoundObject */

/* 1488 */	NdrFcShort( 0x13 ),	/* Flags:  must size, must free, out, */
/* 1490 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1492 */	NdrFcShort( 0x1c ),	/* Type Offset=28 */

	/* Return value */

/* 1494 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1496 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1498 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure DeleteRegisteredObject */

/* 1500 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1502 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1506 */	NdrFcShort( 0xa ),	/* 10 */
/* 1508 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1510 */	NdrFcShort( 0x4a ),	/* 74 */
/* 1512 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1514 */	0x44,		/* Oi2 Flags:  has return, has ext, */
			0x3,		/* 3 */
/* 1516 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1518 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1520 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1522 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1524 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter ObjectGuid */

/* 1526 */	NdrFcShort( 0x10a ),	/* Flags:  must free, in, simple ref, */
/* 1528 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1530 */	NdrFcShort( 0xc ),	/* Type Offset=12 */

	/* Parameter ObjectType */

/* 1532 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 1534 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1536 */	0xd,		/* FC_ENUM16 */
			0x0,		/* 0 */

	/* Return value */

/* 1538 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1540 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1542 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure Initialize */

/* 1544 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1546 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1550 */	NdrFcShort( 0x3 ),	/* 3 */
/* 1552 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1554 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1556 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1558 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x2,		/* 2 */
/* 1560 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1562 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1564 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1566 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1568 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PluginProvider */

/* 1570 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 1572 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1574 */	NdrFcShort( 0x5c ),	/* Type Offset=92 */

	/* Return value */

/* 1576 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1578 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1580 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure Teardown */

/* 1582 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1584 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1588 */	NdrFcShort( 0x4 ),	/* 4 */
/* 1590 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1592 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1594 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1596 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x2,		/* 2 */
/* 1598 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1600 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1602 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1604 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1606 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter PluginProvider */

/* 1608 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 1610 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1612 */	NdrFcShort( 0x5c ),	/* Type Offset=92 */

	/* Return value */

/* 1614 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1616 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1618 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Procedure HandleRequest */

/* 1620 */	0x33,		/* FC_AUTO_HANDLE */
			0x6c,		/* Old Flags:  object, Oi2 */
/* 1622 */	NdrFcLong( 0x0 ),	/* 0 */
/* 1626 */	NdrFcShort( 0x5 ),	/* 5 */
/* 1628 */	NdrFcShort( 0x28 ),	/* X64 Stack size/offset = 40 */
/* 1630 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1632 */	NdrFcShort( 0x8 ),	/* 8 */
/* 1634 */	0x46,		/* Oi2 Flags:  clt must size, has return, has ext, */
			0x4,		/* 4 */
/* 1636 */	0xa,		/* 10 */
			0x1,		/* Ext Flags:  new corr desc, */
/* 1638 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1640 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1642 */	NdrFcShort( 0x0 ),	/* 0 */
/* 1644 */	NdrFcShort( 0x0 ),	/* 0 */

	/* Parameter RequestType */

/* 1646 */	NdrFcShort( 0x48 ),	/* Flags:  in, base type, */
/* 1648 */	NdrFcShort( 0x8 ),	/* X64 Stack size/offset = 8 */
/* 1650 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

	/* Parameter PluginProvider */

/* 1652 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 1654 */	NdrFcShort( 0x10 ),	/* X64 Stack size/offset = 16 */
/* 1656 */	NdrFcShort( 0x5c ),	/* Type Offset=92 */

	/* Parameter PluginObject */

/* 1658 */	NdrFcShort( 0xb ),	/* Flags:  must size, must free, in, */
/* 1660 */	NdrFcShort( 0x18 ),	/* X64 Stack size/offset = 24 */
/* 1662 */	NdrFcShort( 0x20 ),	/* Type Offset=32 */

	/* Return value */

/* 1664 */	NdrFcShort( 0x70 ),	/* Flags:  out, return, base type, */
/* 1666 */	NdrFcShort( 0x20 ),	/* X64 Stack size/offset = 32 */
/* 1668 */	0x8,		/* FC_LONG */
			0x0,		/* 0 */

			0x0
        }
    };

static const hnspluginapi_MIDL_TYPE_FORMAT_STRING hnspluginapi__MIDL_TypeFormatString =
    {
        0,
        {
			NdrFcShort( 0x0 ),	/* 0 */
/*  2 */	
			0x11, 0x0,	/* FC_RP */
/*  4 */	NdrFcShort( 0x8 ),	/* Offset= 8 (12) */
/*  6 */	
			0x1d,		/* FC_SMFARRAY */
			0x0,		/* 0 */
/*  8 */	NdrFcShort( 0x8 ),	/* 8 */
/* 10 */	0x1,		/* FC_BYTE */
			0x5b,		/* FC_END */
/* 12 */	
			0x15,		/* FC_STRUCT */
			0x3,		/* 3 */
/* 14 */	NdrFcShort( 0x10 ),	/* 16 */
/* 16 */	0x8,		/* FC_LONG */
			0x6,		/* FC_SHORT */
/* 18 */	0x6,		/* FC_SHORT */
			0x4c,		/* FC_EMBEDDED_COMPLEX */
/* 20 */	0x0,		/* 0 */
			NdrFcShort( 0xfff1 ),	/* Offset= -15 (6) */
			0x5b,		/* FC_END */
/* 24 */	
			0x11, 0x8,	/* FC_RP [simple_pointer] */
/* 26 */	
			0x25,		/* FC_C_WSTRING */
			0x5c,		/* FC_PAD */
/* 28 */	
			0x11, 0x10,	/* FC_RP [pointer_deref] */
/* 30 */	NdrFcShort( 0x2 ),	/* Offset= 2 (32) */
/* 32 */	
			0x2f,		/* FC_IP */
			0x5a,		/* FC_CONSTANT_IID */
/* 34 */	NdrFcLong( 0xdbf70581 ),	/* -604568191 */
/* 38 */	NdrFcShort( 0xefeb ),	/* -4117 */
/* 40 */	NdrFcShort( 0x4936 ),	/* 18742 */
/* 42 */	0xb5,		/* 181 */
			0x7b,		/* 123 */
/* 44 */	0x34,		/* 52 */
			0x7b,		/* 123 */
/* 46 */	0x74,		/* 116 */
			0x36,		/* 54 */
/* 48 */	0x3b,		/* 59 */
			0xa8,		/* 168 */
/* 50 */	
			0x11, 0xc,	/* FC_RP [alloced_on_stack] [simple_pointer] */
/* 52 */	0x8,		/* FC_LONG */
			0x5c,		/* FC_PAD */
/* 54 */	
			0x11, 0x14,	/* FC_RP [alloced_on_stack] [pointer_deref] */
/* 56 */	NdrFcShort( 0x2 ),	/* Offset= 2 (58) */
/* 58 */	
			0x13, 0x8,	/* FC_OP [simple_pointer] */
/* 60 */	
			0x25,		/* FC_C_WSTRING */
			0x5c,		/* FC_PAD */
/* 62 */	
			0x11, 0x4,	/* FC_RP [alloced_on_stack] */
/* 64 */	NdrFcShort( 0xffcc ),	/* Offset= -52 (12) */
/* 66 */	
			0x11, 0xc,	/* FC_RP [alloced_on_stack] [simple_pointer] */
/* 68 */	0x3,		/* FC_SMALL */
			0x5c,		/* FC_PAD */
/* 70 */	
			0x11, 0x10,	/* FC_RP [pointer_deref] */
/* 72 */	NdrFcShort( 0x2 ),	/* Offset= 2 (74) */
/* 74 */	
			0x2f,		/* FC_IP */
			0x5a,		/* FC_CONSTANT_IID */
/* 76 */	NdrFcLong( 0xdbf70581 ),	/* -604568191 */
/* 80 */	NdrFcShort( 0xefeb ),	/* -4117 */
/* 82 */	NdrFcShort( 0x4936 ),	/* 18742 */
/* 84 */	0xb5,		/* 181 */
			0x7b,		/* 123 */
/* 86 */	0x34,		/* 52 */
			0x7b,		/* 123 */
/* 88 */	0x74,		/* 116 */
			0x36,		/* 54 */
/* 90 */	0x3b,		/* 59 */
			0xa9,		/* 169 */
/* 92 */	
			0x2f,		/* FC_IP */
			0x5a,		/* FC_CONSTANT_IID */
/* 94 */	NdrFcLong( 0x9292e69b ),	/* -1835866469 */
/* 98 */	NdrFcShort( 0x21ad ),	/* 8621 */
/* 100 */	NdrFcShort( 0x400e ),	/* 16398 */
/* 102 */	0x93,		/* 147 */
			0x57,		/* 87 */
/* 104 */	0x9e,		/* 158 */
			0x36,		/* 54 */
/* 106 */	0x6d,		/* 109 */
			0x35,		/* 53 */
/* 108 */	0xd2,		/* 210 */
			0xa0,		/* 160 */

			0x0
        }
    };


/* Standard interface: __MIDL_itf_hnspluginapi_0000_0000, ver. 0.0,
   GUID={0x00000000,0x0000,0x0000,{0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00}} */


/* Object interface: IUnknown, ver. 0.0,
   GUID={0x00000000,0x0000,0x0000,{0xC0,0x00,0x00,0x00,0x00,0x00,0x00,0x46}} */


/* Object interface: IHNSObject, ver. 0.0,
   GUID={0xdbf70581,0xefeb,0x4936,{0xb5,0x7b,0x34,0x7b,0x74,0x36,0x3b,0xa8}} */

#pragma code_seg(".orpc")
static const unsigned short IHNSObject_FormatStringOffsetTable[] =
    {
    0,
    38,
    76,
    114,
    158,
    196,
    234,
    272,
    316,
    360,
    398,
    436,
    480,
    524,
    568,
    612,
    656,
    700,
    738,
    782,
    826,
    870,
    914,
    958,
    1002,
    1046,
    1090
    };



/* Object interface: IHNSObjectList, ver. 0.0,
   GUID={0xdbf70581,0xefeb,0x4936,{0xb5,0x7b,0x34,0x7b,0x74,0x36,0x3b,0xa9}} */

#pragma code_seg(".orpc")
static const unsigned short IHNSObjectList_FormatStringOffsetTable[] =
    {
    1128,
    1166
    };



/* Object interface: IHNSProvider, ver. 0.0,
   GUID={0x9292e69b,0x21ad,0x400e,{0x93,0x57,0x9e,0x36,0x6d,0x35,0xd2,0xa0}} */

#pragma code_seg(".orpc")
static const unsigned short IHNSProvider_FormatStringOffsetTable[] =
    {
    1210,
    1248,
    1286,
    1324,
    1368,
    1412,
    1456,
    1500
    };



/* Object interface: IHNSPlugin, ver. 0.0,
   GUID={0x398a46e6,0x572e,0x4d19,{0x84,0x76,0x1c,0xab,0xa8,0x46,0x28,0x94}} */

#pragma code_seg(".orpc")
static const unsigned short IHNSPlugin_FormatStringOffsetTable[] =
    {
    1544,
    1582,
    1620
    };



#endif /* defined(_M_AMD64)*/



/* this ALWAYS GENERATED file contains the proxy stub code */


 /* File created by MIDL compiler version 8.01.0618 */
/* @@MIDL_FILE_HEADING(  ) */

#if defined(_M_AMD64)




#if !defined(__RPC_WIN64__)
#error  Invalid build platform for this stub.
#endif


#include "ndr64types.h"
#include "pshpack8.h"


typedef 
NDR64_FORMAT_CHAR
__midl_frag208_t;
extern const __midl_frag208_t __midl_frag208;

typedef 
struct _NDR64_CONSTANT_IID_FORMAT
__midl_frag207_t;
extern const __midl_frag207_t __midl_frag207;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag206_t;
extern const __midl_frag206_t __midl_frag206;

typedef 
struct _NDR64_CONSTANT_IID_FORMAT
__midl_frag205_t;
extern const __midl_frag205_t __midl_frag205;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag204_t;
extern const __midl_frag204_t __midl_frag204;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
    struct _NDR64_PARAM_FORMAT frag5;
}
__midl_frag202_t;
extern const __midl_frag202_t __midl_frag202;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
}
__midl_frag198_t;
extern const __midl_frag198_t __midl_frag198;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag191_t;
extern const __midl_frag191_t __midl_frag191;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag190_t;
extern const __midl_frag190_t __midl_frag190;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag186_t;
extern const __midl_frag186_t __midl_frag186;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag184_t;
extern const __midl_frag184_t __midl_frag184;

typedef 
struct _NDR64_CONFORMANT_STRING_FORMAT
__midl_frag179_t;
extern const __midl_frag179_t __midl_frag179;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag178_t;
extern const __midl_frag178_t __midl_frag178;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag177_t;
extern const __midl_frag177_t __midl_frag177;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag170_t;
extern const __midl_frag170_t __midl_frag170;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
}
__midl_frag158_t;
extern const __midl_frag158_t __midl_frag158;

typedef 
struct _NDR64_CONSTANT_IID_FORMAT
__midl_frag156_t;
extern const __midl_frag156_t __midl_frag156;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag155_t;
extern const __midl_frag155_t __midl_frag155;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag154_t;
extern const __midl_frag154_t __midl_frag154;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
}
__midl_frag153_t;
extern const __midl_frag153_t __midl_frag153;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag142_t;
extern const __midl_frag142_t __midl_frag142;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag139_t;
extern const __midl_frag139_t __midl_frag139;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
}
__midl_frag138_t;
extern const __midl_frag138_t __midl_frag138;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
}
__midl_frag134_t;
extern const __midl_frag134_t __midl_frag134;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag127_t;
extern const __midl_frag127_t __midl_frag127;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag117_t;
extern const __midl_frag117_t __midl_frag117;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag116_t;
extern const __midl_frag116_t __midl_frag116;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag113_t;
extern const __midl_frag113_t __midl_frag113;

typedef 
NDR64_FORMAT_CHAR
__midl_frag111_t;
extern const __midl_frag111_t __midl_frag111;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag110_t;
extern const __midl_frag110_t __midl_frag110;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag107_t;
extern const __midl_frag107_t __midl_frag107;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag101_t;
extern const __midl_frag101_t __midl_frag101;

typedef 
struct _NDR64_POINTER_FORMAT
__midl_frag99_t;
extern const __midl_frag99_t __midl_frag99;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag96_t;
extern const __midl_frag96_t __midl_frag96;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag90_t;
extern const __midl_frag90_t __midl_frag90;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag68_t;
extern const __midl_frag68_t __midl_frag68;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag63_t;
extern const __midl_frag63_t __midl_frag63;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag58_t;
extern const __midl_frag58_t __midl_frag58;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag53_t;
extern const __midl_frag53_t __midl_frag53;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
    struct _NDR64_PARAM_FORMAT frag4;
}
__midl_frag47_t;
extern const __midl_frag47_t __midl_frag47;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
}
__midl_frag25_t;
extern const __midl_frag25_t __midl_frag25;

typedef 
struct 
{
    struct _NDR64_PROC_FORMAT frag1;
    struct _NDR64_PARAM_FORMAT frag2;
    struct _NDR64_PARAM_FORMAT frag3;
}
__midl_frag22_t;
extern const __midl_frag22_t __midl_frag22;

typedef 
struct 
{
    struct _NDR64_STRUCTURE_HEADER_FORMAT frag1;
}
__midl_frag4_t;
extern const __midl_frag4_t __midl_frag4;

typedef 
NDR64_FORMAT_UINT32
__midl_frag1_t;
extern const __midl_frag1_t __midl_frag1;

static const __midl_frag208_t __midl_frag208 =
0x5    /* FC64_INT32 */;

static const __midl_frag207_t __midl_frag207 =
{ 
/* struct _NDR64_CONSTANT_IID_FORMAT */
    0x24,    /* FC64_IP */
    (NDR64_UINT8) 1 /* 0x1 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    {
        0xdbf70581,
        0xefeb,
        0x4936,
        {0xb5, 0x7b, 0x34, 0x7b, 0x74, 0x36, 0x3b, 0xa8}
    }
};

static const __midl_frag206_t __midl_frag206 =
{ 
/* *struct _NDR64_POINTER_FORMAT */
    0x24,    /* FC64_IP */
    (NDR64_UINT8) 0 /* 0x0 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag207
};

static const __midl_frag205_t __midl_frag205 =
{ 
/* struct _NDR64_CONSTANT_IID_FORMAT */
    0x24,    /* FC64_IP */
    (NDR64_UINT8) 1 /* 0x1 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    {
        0x9292e69b,
        0x21ad,
        0x400e,
        {0x93, 0x57, 0x9e, 0x36, 0x6d, 0x35, 0xd2, 0xa0}
    }
};

static const __midl_frag204_t __midl_frag204 =
{ 
/* *struct _NDR64_POINTER_FORMAT */
    0x24,    /* FC64_IP */
    (NDR64_UINT8) 0 /* 0x0 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag205
};

static const __midl_frag202_t __midl_frag202 =
{ 
/* HandleRequest */
    { 
    /* HandleRequest */      /* procedure HandleRequest */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 40 /* 0x28 */ ,  /* Stack size */
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 4 /* 0x4 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* RequestType */      /* parameter RequestType */
        &__midl_frag208,
        { 
        /* RequestType */
            0,
            0,
            0,
            1,
            0,
            0,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [in], Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* PluginProvider */      /* parameter PluginProvider */
        &__midl_frag204,
        { 
        /* PluginProvider */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in] */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* PluginObject */      /* parameter PluginObject */
        &__midl_frag206,
        { 
        /* PluginObject */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in] */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        32 /* 0x20 */,   /* Stack offset */
    }
};

static const __midl_frag198_t __midl_frag198 =
{ 
/* Teardown */
    { 
    /* Teardown */      /* procedure Teardown */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 24 /* 0x18 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 2 /* 0x2 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PluginProvider */      /* parameter PluginProvider */
        &__midl_frag204,
        { 
        /* PluginProvider */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in] */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    }
};

static const __midl_frag191_t __midl_frag191 =
{ 
/* *GUID */
    0x20,    /* FC64_RP */
    (NDR64_UINT8) 0 /* 0x0 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag4
};

static const __midl_frag190_t __midl_frag190 =
{ 
/* DeleteRegisteredObject */
    { 
    /* DeleteRegisteredObject */      /* procedure DeleteRegisteredObject */
        (NDR64_UINT32) 524611 /* 0x80143 */,    /* auto handle */ /* IsIntrepreted, [object], HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 80 /* 0x50 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* ObjectGuid */      /* parameter ObjectGuid */
        &__midl_frag4,
        { 
        /* ObjectGuid */
            0,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* ObjectType */      /* parameter ObjectType */
        &__midl_frag208,
        { 
        /* ObjectType */
            0,
            0,
            0,
            1,
            0,
            0,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [in], Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag186_t __midl_frag186 =
{ 
/* **struct _NDR64_POINTER_FORMAT */
    0x20,    /* FC64_RP */
    (NDR64_UINT8) 16 /* 0x10 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag206
};

static const __midl_frag184_t __midl_frag184 =
{ 
/* FindObject */
    { 
    /* FindObject */      /* procedure FindObject */
        (NDR64_UINT32) 655683 /* 0xa0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 72 /* 0x48 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* ObjectGuid */      /* parameter ObjectGuid */
        &__midl_frag4,
        { 
        /* ObjectGuid */
            0,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* FoundObject */      /* parameter FoundObject */
        &__midl_frag186,
        { 
        /* FoundObject */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [out] */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag179_t __midl_frag179 =
{ 
/* *WCHAR */
    { 
    /* *WCHAR */
        0x64,    /* FC64_CONF_WCHAR_STRING */
        { 
        /* *WCHAR */
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
        },
        (NDR64_UINT16) 2 /* 0x2 */
    }
};

static const __midl_frag178_t __midl_frag178 =
{ 
/* *WCHAR */
    0x20,    /* FC64_RP */
    (NDR64_UINT8) 0 /* 0x0 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag179
};

static const __midl_frag177_t __midl_frag177 =
{ 
/* FindNetworkByName */
    { 
    /* FindNetworkByName */      /* procedure FindNetworkByName */
        (NDR64_UINT32) 917827 /* 0xe0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* NetworkName */      /* parameter NetworkName */
        &__midl_frag179,
        { 
        /* NetworkName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* FoundObject */      /* parameter FoundObject */
        &__midl_frag186,
        { 
        /* FoundObject */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [out] */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag170_t __midl_frag170 =
{ 
/* CreateEndpoint */
    { 
    /* CreateEndpoint */      /* procedure CreateEndpoint */
        (NDR64_UINT32) 917827 /* 0xe0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* InputObject */      /* parameter InputObject */
        &__midl_frag206,
        { 
        /* InputObject */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in] */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* CreatedEndpoint */      /* parameter CreatedEndpoint */
        &__midl_frag186,
        { 
        /* CreatedEndpoint */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [out] */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag158_t __midl_frag158 =
{ 
/* CreateObject */
    { 
    /* CreateObject */      /* procedure CreateObject */
        (NDR64_UINT32) 655683 /* 0xa0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, HasReturn */
        (NDR64_UINT32) 24 /* 0x18 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 2 /* 0x2 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* OutNewObject */      /* parameter OutNewObject */
        &__midl_frag186,
        { 
        /* OutNewObject */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [out] */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    }
};

static const __midl_frag156_t __midl_frag156 =
{ 
/* struct _NDR64_CONSTANT_IID_FORMAT */
    0x24,    /* FC64_IP */
    (NDR64_UINT8) 1 /* 0x1 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    {
        0xdbf70581,
        0xefeb,
        0x4936,
        {0xb5, 0x7b, 0x34, 0x7b, 0x74, 0x36, 0x3b, 0xa9}
    }
};

static const __midl_frag155_t __midl_frag155 =
{ 
/* *struct _NDR64_POINTER_FORMAT */
    0x24,    /* FC64_IP */
    (NDR64_UINT8) 0 /* 0x0 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag156
};

static const __midl_frag154_t __midl_frag154 =
{ 
/* **struct _NDR64_POINTER_FORMAT */
    0x20,    /* FC64_RP */
    (NDR64_UINT8) 16 /* 0x10 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag155
};

static const __midl_frag153_t __midl_frag153 =
{ 
/* EnumerateEndpoints */
    { 
    /* EnumerateEndpoints */      /* procedure EnumerateEndpoints */
        (NDR64_UINT32) 655683 /* 0xa0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, HasReturn */
        (NDR64_UINT32) 24 /* 0x18 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 2 /* 0x2 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* OutEndpoints */      /* parameter OutEndpoints */
        &__midl_frag154,
        { 
        /* OutEndpoints */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [out] */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    }
};

static const __midl_frag142_t __midl_frag142 =
{ 
/* GetObject */
    { 
    /* GetObject */      /* procedure GetObject */
        (NDR64_UINT32) 655683 /* 0xa0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* ObjectIndex */      /* parameter ObjectIndex */
        &__midl_frag208,
        { 
        /* ObjectIndex */
            0,
            0,
            0,
            1,
            0,
            0,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [in], Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* OutObject */      /* parameter OutObject */
        &__midl_frag186,
        { 
        /* OutObject */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [out] */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag139_t __midl_frag139 =
{ 
/* *DWORD */
    0x20,    /* FC64_RP */
    (NDR64_UINT8) 12 /* 0xc */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag208
};

static const __midl_frag138_t __midl_frag138 =
{ 
/* ObjectCount */
    { 
    /* ObjectCount */      /* procedure ObjectCount */
        (NDR64_UINT32) 524611 /* 0x80143 */,    /* auto handle */ /* IsIntrepreted, [object], HasReturn */
        (NDR64_UINT32) 24 /* 0x18 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 40 /* 0x28 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 2 /* 0x2 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* Count */      /* parameter Count */
        &__midl_frag208,
        { 
        /* Count */
            0,
            0,
            0,
            0,
            1,
            0,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            1
        },    /* [out], Basetype, SimpleRef, UseCache */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    }
};

static const __midl_frag134_t __midl_frag134 =
{ 
/* DeleteProperty */
    { 
    /* DeleteProperty */      /* procedure DeleteProperty */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 24 /* 0x18 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 2 /* 0x2 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    }
};

static const __midl_frag127_t __midl_frag127 =
{ 
/* GetObjectList */
    { 
    /* GetObjectList */      /* procedure GetObjectList */
        (NDR64_UINT32) 917827 /* 0xe0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* OutObjectList */      /* parameter OutObjectList */
        &__midl_frag154,
        { 
        /* OutObjectList */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [out] */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag117_t __midl_frag117 =
{ 
/* *WCHAR */
    0x22,    /* FC64_OP */
    (NDR64_UINT8) 0 /* 0x0 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag179
};

static const __midl_frag116_t __midl_frag116 =
{ 
/* **WCHAR */
    0x20,    /* FC64_RP */
    (NDR64_UINT8) 20 /* 0x14 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag117
};

static const __midl_frag113_t __midl_frag113 =
{ 
/* GetString */
    { 
    /* GetString */      /* procedure GetString */
        (NDR64_UINT32) 917827 /* 0xe0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* StringValue */      /* parameter StringValue */
        &__midl_frag116,
        { 
        /* StringValue */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            1
        },    /* MustSize, MustFree, [out], UseCache */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag111_t __midl_frag111 =
0x2    /* FC64_INT8 */;

static const __midl_frag110_t __midl_frag110 =
{ 
/* *BOOLEAN */
    0x20,    /* FC64_RP */
    (NDR64_UINT8) 12 /* 0xc */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag111
};

static const __midl_frag107_t __midl_frag107 =
{ 
/* GetBoolean */
    { 
    /* GetBoolean */      /* procedure GetBoolean */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 37 /* 0x25 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* BooleanValue */      /* parameter BooleanValue */
        &__midl_frag111,
        { 
        /* BooleanValue */
            0,
            0,
            0,
            0,
            1,
            0,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            1
        },    /* [out], Basetype, SimpleRef, UseCache */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag101_t __midl_frag101 =
{ 
/* GetInteger */
    { 
    /* GetInteger */      /* procedure GetInteger */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 40 /* 0x28 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* IntegerValue */      /* parameter IntegerValue */
        &__midl_frag208,
        { 
        /* IntegerValue */
            0,
            0,
            0,
            0,
            1,
            0,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            1
        },    /* [out], Basetype, SimpleRef, UseCache */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag99_t __midl_frag99 =
{ 
/* *GUID */
    0x20,    /* FC64_RP */
    (NDR64_UINT8) 4 /* 0x4 */,
    (NDR64_UINT16) 0 /* 0x0 */,
    &__midl_frag4
};

static const __midl_frag96_t __midl_frag96 =
{ 
/* GetGuid */
    { 
    /* GetGuid */      /* procedure GetGuid */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 80 /* 0x50 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* GuidValue */      /* parameter GuidValue */
        &__midl_frag4,
        { 
        /* GuidValue */
            0,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            1
        },    /* MustFree, [out], SimpleRef, UseCache */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag90_t __midl_frag90 =
{ 
/* RemoveListObject */
    { 
    /* RemoveListObject */      /* procedure RemoveListObject */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* Object */      /* parameter Object */
        &__midl_frag206,
        { 
        /* Object */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in] */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag68_t __midl_frag68 =
{ 
/* SetString */
    { 
    /* SetString */      /* procedure SetString */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 0 /* 0x0 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* StringValue */      /* parameter StringValue */
        &__midl_frag179,
        { 
        /* StringValue */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag63_t __midl_frag63 =
{ 
/* SetBoolean */
    { 
    /* SetBoolean */      /* procedure SetBoolean */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 5 /* 0x5 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* BoolValue */      /* parameter BoolValue */
        &__midl_frag111,
        { 
        /* BoolValue */
            0,
            0,
            0,
            1,
            0,
            0,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [in], Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag58_t __midl_frag58 =
{ 
/* SetInteger */
    { 
    /* SetInteger */      /* procedure SetInteger */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* IntegerValue */      /* parameter IntegerValue */
        &__midl_frag208,
        { 
        /* IntegerValue */
            0,
            0,
            0,
            1,
            0,
            0,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [in], Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag53_t __midl_frag53 =
{ 
/* SetGuid */
    { 
    /* SetGuid */      /* procedure SetGuid */
        (NDR64_UINT32) 786755 /* 0xc0143 */,    /* auto handle */ /* IsIntrepreted, [object], ClientMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 72 /* 0x48 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag179,
        { 
        /* PropertyName */
            1,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustSize, MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* GuidValue */      /* parameter GuidValue */
        &__midl_frag4,
        { 
        /* GuidValue */
            0,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag47_t __midl_frag47 =
{ 
/* GetPropertyName */
    { 
    /* GetPropertyName */      /* procedure GetPropertyName */
        (NDR64_UINT32) 655683 /* 0xa0143 */,    /* auto handle */ /* IsIntrepreted, [object], ServerMustSize, HasReturn */
        (NDR64_UINT32) 32 /* 0x20 */ ,  /* Stack size */
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 3 /* 0x3 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* Index */      /* parameter Index */
        &__midl_frag208,
        { 
        /* Index */
            0,
            0,
            0,
            1,
            0,
            0,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [in], Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* PropertyName */      /* parameter PropertyName */
        &__midl_frag116,
        { 
        /* PropertyName */
            1,
            1,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            1
        },    /* MustSize, MustFree, [out], UseCache */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        24 /* 0x18 */,   /* Stack offset */
    }
};

static const __midl_frag25_t __midl_frag25 =
{ 
/* SetObjectType */
    { 
    /* SetObjectType */      /* procedure SetObjectType */
        (NDR64_UINT32) 524611 /* 0x80143 */,    /* auto handle */ /* IsIntrepreted, [object], HasReturn */
        (NDR64_UINT32) 24 /* 0x18 */ ,  /* Stack size */
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 2 /* 0x2 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* ObjectType */      /* parameter ObjectType */
        &__midl_frag208,
        { 
        /* ObjectType */
            0,
            0,
            0,
            1,
            0,
            0,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [in], Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    }
};

static const __midl_frag22_t __midl_frag22 =
{ 
/* SetObjectId */
    { 
    /* SetObjectId */      /* procedure SetObjectId */
        (NDR64_UINT32) 524611 /* 0x80143 */,    /* auto handle */ /* IsIntrepreted, [object], HasReturn */
        (NDR64_UINT32) 24 /* 0x18 */ ,  /* Stack size */
        (NDR64_UINT32) 72 /* 0x48 */,
        (NDR64_UINT32) 8 /* 0x8 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 0 /* 0x0 */,
        (NDR64_UINT16) 2 /* 0x2 */,
        (NDR64_UINT16) 0 /* 0x0 */
    },
    { 
    /* ObjectId */      /* parameter ObjectId */
        &__midl_frag4,
        { 
        /* ObjectId */
            0,
            1,
            0,
            1,
            0,
            0,
            0,
            0,
            1,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* MustFree, [in], SimpleRef */
        (NDR64_UINT16) 0 /* 0x0 */,
        8 /* 0x8 */,   /* Stack offset */
    },
    { 
    /* HRESULT */      /* parameter HRESULT */
        &__midl_frag208,
        { 
        /* HRESULT */
            0,
            0,
            0,
            0,
            1,
            1,
            1,
            1,
            0,
            0,
            0,
            0,
            0,
            (NDR64_UINT16) 0 /* 0x0 */,
            0
        },    /* [out], IsReturn, Basetype, ByValue */
        (NDR64_UINT16) 0 /* 0x0 */,
        16 /* 0x10 */,   /* Stack offset */
    }
};

static const __midl_frag4_t __midl_frag4 =
{ 
/* GUID */
    { 
    /* GUID */
        0x30,    /* FC64_STRUCT */
        (NDR64_UINT8) 3 /* 0x3 */,
        { 
        /* GUID */
            0,
            0,
            0,
            0,
            0,
            0,
            0,
            0
        },
        (NDR64_UINT8) 0 /* 0x0 */,
        (NDR64_UINT32) 16 /* 0x10 */
    }
};

static const __midl_frag1_t __midl_frag1 =
(NDR64_UINT32) 0 /* 0x0 */;


#include "poppack.h"



/* Standard interface: __MIDL_itf_hnspluginapi_0000_0000, ver. 0.0,
   GUID={0x00000000,0x0000,0x0000,{0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00}} */


/* Object interface: IUnknown, ver. 0.0,
   GUID={0x00000000,0x0000,0x0000,{0xC0,0x00,0x00,0x00,0x00,0x00,0x00,0x46}} */


/* Object interface: IHNSObject, ver. 0.0,
   GUID={0xdbf70581,0xefeb,0x4936,{0xb5,0x7b,0x34,0x7b,0x74,0x36,0x3b,0xa8}} */

#pragma code_seg(".orpc")
static const FormatInfoRef IHNSObject_Ndr64ProcTable[] =
    {
    &__midl_frag22,
    &__midl_frag25,
    &__midl_frag134,
    &__midl_frag58,
    &__midl_frag134,
    &__midl_frag22,
    &__midl_frag25,
    &__midl_frag63,
    &__midl_frag63,
    &__midl_frag158,
    &__midl_frag138,
    &__midl_frag47,
    &__midl_frag53,
    &__midl_frag58,
    &__midl_frag63,
    &__midl_frag68,
    &__midl_frag90,
    &__midl_frag134,
    &__midl_frag90,
    &__midl_frag90,
    &__midl_frag96,
    &__midl_frag101,
    &__midl_frag107,
    &__midl_frag113,
    &__midl_frag177,
    &__midl_frag127,
    &__midl_frag134
    };


static const MIDL_SYNTAX_INFO IHNSObject_SyntaxInfo [  2 ] = 
    {
    {
    {{0x8A885D04,0x1CEB,0x11C9,{0x9F,0xE8,0x08,0x00,0x2B,0x10,0x48,0x60}},{2,0}},
    0,
    hnspluginapi__MIDL_ProcFormatString.Format,
    &IHNSObject_FormatStringOffsetTable[-3],
    hnspluginapi__MIDL_TypeFormatString.Format,
    0,
    0,
    0
    }
    ,{
    {{0x71710533,0xbeba,0x4937,{0x83,0x19,0xb5,0xdb,0xef,0x9c,0xcc,0x36}},{1,0}},
    0,
    0 ,
    (unsigned short *) &IHNSObject_Ndr64ProcTable[-3],
    0,
    0,
    0,
    0
    }
    };

static const MIDL_STUBLESS_PROXY_INFO IHNSObject_ProxyInfo =
    {
    &Object_StubDesc,
    hnspluginapi__MIDL_ProcFormatString.Format,
    &IHNSObject_FormatStringOffsetTable[-3],
    (RPC_SYNTAX_IDENTIFIER*)&_RpcTransferSyntax,
    2,
    (MIDL_SYNTAX_INFO*)IHNSObject_SyntaxInfo
    
    };


static const MIDL_SERVER_INFO IHNSObject_ServerInfo = 
    {
    &Object_StubDesc,
    0,
    hnspluginapi__MIDL_ProcFormatString.Format,
    (unsigned short *) &IHNSObject_FormatStringOffsetTable[-3],
    0,
    (RPC_SYNTAX_IDENTIFIER*)&_NDR64_RpcTransferSyntax,
    2,
    (MIDL_SYNTAX_INFO*)IHNSObject_SyntaxInfo
    };
const CINTERFACE_PROXY_VTABLE(30) _IHNSObjectProxyVtbl = 
{
    &IHNSObject_ProxyInfo,
    &IID_IHNSObject,
    IUnknown_QueryInterface_Proxy,
    IUnknown_AddRef_Proxy,
    IUnknown_Release_Proxy ,
    ObjectStublessClient3 /* IHNSObject::IsEqualId */ ,
    ObjectStublessClient4 /* IHNSObject::IsEqualType */ ,
    ObjectStublessClient5 /* IHNSObject::PropertyExists */ ,
    ObjectStublessClient6 /* IHNSObject::PropertyIsType */ ,
    ObjectStublessClient7 /* IHNSObject::IsClientProperty */ ,
    ObjectStublessClient8 /* IHNSObject::SetObjectId */ ,
    ObjectStublessClient9 /* IHNSObject::SetObjectType */ ,
    ObjectStublessClient10 /* IHNSObject::SetPropertyVisible */ ,
    ObjectStublessClient11 /* IHNSObject::SetPropertyPersistent */ ,
    ObjectStublessClient12 /* IHNSObject::DuplicateObject */ ,
    ObjectStublessClient13 /* IHNSObject::PropertyCount */ ,
    ObjectStublessClient14 /* IHNSObject::GetPropertyName */ ,
    ObjectStublessClient15 /* IHNSObject::SetGuid */ ,
    ObjectStublessClient16 /* IHNSObject::SetInteger */ ,
    ObjectStublessClient17 /* IHNSObject::SetBoolean */ ,
    ObjectStublessClient18 /* IHNSObject::SetString */ ,
    ObjectStublessClient19 /* IHNSObject::SetObject */ ,
    ObjectStublessClient20 /* IHNSObject::SetObjectList */ ,
    ObjectStublessClient21 /* IHNSObject::AddListObject */ ,
    ObjectStublessClient22 /* IHNSObject::RemoveListObject */ ,
    ObjectStublessClient23 /* IHNSObject::GetGuid */ ,
    ObjectStublessClient24 /* IHNSObject::GetInteger */ ,
    ObjectStublessClient25 /* IHNSObject::GetBoolean */ ,
    ObjectStublessClient26 /* IHNSObject::GetString */ ,
    ObjectStublessClient27 /* IHNSObject::GetObject */ ,
    ObjectStublessClient28 /* IHNSObject::GetObjectList */ ,
    ObjectStublessClient29 /* IHNSObject::DeleteProperty */
};

const CInterfaceStubVtbl _IHNSObjectStubVtbl =
{
    &IID_IHNSObject,
    &IHNSObject_ServerInfo,
    30,
    0, /* pure interpreted */
    CStdStubBuffer_METHODS_OPT
};


/* Object interface: IHNSObjectList, ver. 0.0,
   GUID={0xdbf70581,0xefeb,0x4936,{0xb5,0x7b,0x34,0x7b,0x74,0x36,0x3b,0xa9}} */

#pragma code_seg(".orpc")
static const FormatInfoRef IHNSObjectList_Ndr64ProcTable[] =
    {
    &__midl_frag138,
    &__midl_frag142
    };


static const MIDL_SYNTAX_INFO IHNSObjectList_SyntaxInfo [  2 ] = 
    {
    {
    {{0x8A885D04,0x1CEB,0x11C9,{0x9F,0xE8,0x08,0x00,0x2B,0x10,0x48,0x60}},{2,0}},
    0,
    hnspluginapi__MIDL_ProcFormatString.Format,
    &IHNSObjectList_FormatStringOffsetTable[-3],
    hnspluginapi__MIDL_TypeFormatString.Format,
    0,
    0,
    0
    }
    ,{
    {{0x71710533,0xbeba,0x4937,{0x83,0x19,0xb5,0xdb,0xef,0x9c,0xcc,0x36}},{1,0}},
    0,
    0 ,
    (unsigned short *) &IHNSObjectList_Ndr64ProcTable[-3],
    0,
    0,
    0,
    0
    }
    };

static const MIDL_STUBLESS_PROXY_INFO IHNSObjectList_ProxyInfo =
    {
    &Object_StubDesc,
    hnspluginapi__MIDL_ProcFormatString.Format,
    &IHNSObjectList_FormatStringOffsetTable[-3],
    (RPC_SYNTAX_IDENTIFIER*)&_RpcTransferSyntax,
    2,
    (MIDL_SYNTAX_INFO*)IHNSObjectList_SyntaxInfo
    
    };


static const MIDL_SERVER_INFO IHNSObjectList_ServerInfo = 
    {
    &Object_StubDesc,
    0,
    hnspluginapi__MIDL_ProcFormatString.Format,
    (unsigned short *) &IHNSObjectList_FormatStringOffsetTable[-3],
    0,
    (RPC_SYNTAX_IDENTIFIER*)&_NDR64_RpcTransferSyntax,
    2,
    (MIDL_SYNTAX_INFO*)IHNSObjectList_SyntaxInfo
    };
const CINTERFACE_PROXY_VTABLE(5) _IHNSObjectListProxyVtbl = 
{
    &IHNSObjectList_ProxyInfo,
    &IID_IHNSObjectList,
    IUnknown_QueryInterface_Proxy,
    IUnknown_AddRef_Proxy,
    IUnknown_Release_Proxy ,
    ObjectStublessClient3 /* IHNSObjectList::ObjectCount */ ,
    ObjectStublessClient4 /* IHNSObjectList::GetObject */
};

const CInterfaceStubVtbl _IHNSObjectListStubVtbl =
{
    &IID_IHNSObjectList,
    &IHNSObjectList_ServerInfo,
    5,
    0, /* pure interpreted */
    CStdStubBuffer_METHODS_OPT
};


/* Object interface: IHNSProvider, ver. 0.0,
   GUID={0x9292e69b,0x21ad,0x400e,{0x93,0x57,0x9e,0x36,0x6d,0x35,0xd2,0xa0}} */

#pragma code_seg(".orpc")
static const FormatInfoRef IHNSProvider_Ndr64ProcTable[] =
    {
    &__midl_frag153,
    &__midl_frag153,
    &__midl_frag158,
    &__midl_frag170,
    &__midl_frag170,
    &__midl_frag177,
    &__midl_frag184,
    &__midl_frag190
    };


static const MIDL_SYNTAX_INFO IHNSProvider_SyntaxInfo [  2 ] = 
    {
    {
    {{0x8A885D04,0x1CEB,0x11C9,{0x9F,0xE8,0x08,0x00,0x2B,0x10,0x48,0x60}},{2,0}},
    0,
    hnspluginapi__MIDL_ProcFormatString.Format,
    &IHNSProvider_FormatStringOffsetTable[-3],
    hnspluginapi__MIDL_TypeFormatString.Format,
    0,
    0,
    0
    }
    ,{
    {{0x71710533,0xbeba,0x4937,{0x83,0x19,0xb5,0xdb,0xef,0x9c,0xcc,0x36}},{1,0}},
    0,
    0 ,
    (unsigned short *) &IHNSProvider_Ndr64ProcTable[-3],
    0,
    0,
    0,
    0
    }
    };

static const MIDL_STUBLESS_PROXY_INFO IHNSProvider_ProxyInfo =
    {
    &Object_StubDesc,
    hnspluginapi__MIDL_ProcFormatString.Format,
    &IHNSProvider_FormatStringOffsetTable[-3],
    (RPC_SYNTAX_IDENTIFIER*)&_RpcTransferSyntax,
    2,
    (MIDL_SYNTAX_INFO*)IHNSProvider_SyntaxInfo
    
    };


static const MIDL_SERVER_INFO IHNSProvider_ServerInfo = 
    {
    &Object_StubDesc,
    0,
    hnspluginapi__MIDL_ProcFormatString.Format,
    (unsigned short *) &IHNSProvider_FormatStringOffsetTable[-3],
    0,
    (RPC_SYNTAX_IDENTIFIER*)&_NDR64_RpcTransferSyntax,
    2,
    (MIDL_SYNTAX_INFO*)IHNSProvider_SyntaxInfo
    };
const CINTERFACE_PROXY_VTABLE(11) _IHNSProviderProxyVtbl = 
{
    &IHNSProvider_ProxyInfo,
    &IID_IHNSProvider,
    IUnknown_QueryInterface_Proxy,
    IUnknown_AddRef_Proxy,
    IUnknown_Release_Proxy ,
    ObjectStublessClient3 /* IHNSProvider::EnumerateNetworks */ ,
    ObjectStublessClient4 /* IHNSProvider::EnumerateEndpoints */ ,
    ObjectStublessClient5 /* IHNSProvider::CreateObject */ ,
    ObjectStublessClient6 /* IHNSProvider::CreateNetwork */ ,
    ObjectStublessClient7 /* IHNSProvider::CreateEndpoint */ ,
    ObjectStublessClient8 /* IHNSProvider::FindNetworkByName */ ,
    ObjectStublessClient9 /* IHNSProvider::FindObject */ ,
    ObjectStublessClient10 /* IHNSProvider::DeleteRegisteredObject */
};

const CInterfaceStubVtbl _IHNSProviderStubVtbl =
{
    &IID_IHNSProvider,
    &IHNSProvider_ServerInfo,
    11,
    0, /* pure interpreted */
    CStdStubBuffer_METHODS_OPT
};


/* Object interface: IHNSPlugin, ver. 0.0,
   GUID={0x398a46e6,0x572e,0x4d19,{0x84,0x76,0x1c,0xab,0xa8,0x46,0x28,0x94}} */

#pragma code_seg(".orpc")
static const FormatInfoRef IHNSPlugin_Ndr64ProcTable[] =
    {
    &__midl_frag198,
    &__midl_frag198,
    &__midl_frag202
    };


static const MIDL_SYNTAX_INFO IHNSPlugin_SyntaxInfo [  2 ] = 
    {
    {
    {{0x8A885D04,0x1CEB,0x11C9,{0x9F,0xE8,0x08,0x00,0x2B,0x10,0x48,0x60}},{2,0}},
    0,
    hnspluginapi__MIDL_ProcFormatString.Format,
    &IHNSPlugin_FormatStringOffsetTable[-3],
    hnspluginapi__MIDL_TypeFormatString.Format,
    0,
    0,
    0
    }
    ,{
    {{0x71710533,0xbeba,0x4937,{0x83,0x19,0xb5,0xdb,0xef,0x9c,0xcc,0x36}},{1,0}},
    0,
    0 ,
    (unsigned short *) &IHNSPlugin_Ndr64ProcTable[-3],
    0,
    0,
    0,
    0
    }
    };

static const MIDL_STUBLESS_PROXY_INFO IHNSPlugin_ProxyInfo =
    {
    &Object_StubDesc,
    hnspluginapi__MIDL_ProcFormatString.Format,
    &IHNSPlugin_FormatStringOffsetTable[-3],
    (RPC_SYNTAX_IDENTIFIER*)&_RpcTransferSyntax,
    2,
    (MIDL_SYNTAX_INFO*)IHNSPlugin_SyntaxInfo
    
    };


static const MIDL_SERVER_INFO IHNSPlugin_ServerInfo = 
    {
    &Object_StubDesc,
    0,
    hnspluginapi__MIDL_ProcFormatString.Format,
    (unsigned short *) &IHNSPlugin_FormatStringOffsetTable[-3],
    0,
    (RPC_SYNTAX_IDENTIFIER*)&_NDR64_RpcTransferSyntax,
    2,
    (MIDL_SYNTAX_INFO*)IHNSPlugin_SyntaxInfo
    };
const CINTERFACE_PROXY_VTABLE(6) _IHNSPluginProxyVtbl = 
{
    &IHNSPlugin_ProxyInfo,
    &IID_IHNSPlugin,
    IUnknown_QueryInterface_Proxy,
    IUnknown_AddRef_Proxy,
    IUnknown_Release_Proxy ,
    ObjectStublessClient3 /* IHNSPlugin::Initialize */ ,
    ObjectStublessClient4 /* IHNSPlugin::Teardown */ ,
    ObjectStublessClient5 /* IHNSPlugin::HandleRequest */
};

const CInterfaceStubVtbl _IHNSPluginStubVtbl =
{
    &IID_IHNSPlugin,
    &IHNSPlugin_ServerInfo,
    6,
    0, /* pure interpreted */
    CStdStubBuffer_METHODS_OPT
};

static const MIDL_STUB_DESC Object_StubDesc = 
    {
    0,
    NdrOleAllocate,
    NdrOleFree,
    0,
    0,
    0,
    0,
    0,
    hnspluginapi__MIDL_TypeFormatString.Format,
    1, /* -error bounds_check flag */
    0xa0000, /* Ndr library version */
    0,
    0x801026a, /* MIDL Version 8.1.618 */
    0,
    0,
    0,  /* notify & notify_flag routine table */
    0x2000001, /* MIDL flag */
    0, /* cs routines */
    0,   /* proxy/server info */
    0
    };

const CInterfaceProxyVtbl * const _hnspluginapi_ProxyVtblList[] = 
{
    ( CInterfaceProxyVtbl *) &_IHNSObjectProxyVtbl,
    ( CInterfaceProxyVtbl *) &_IHNSObjectListProxyVtbl,
    ( CInterfaceProxyVtbl *) &_IHNSProviderProxyVtbl,
    ( CInterfaceProxyVtbl *) &_IHNSPluginProxyVtbl,
    0
};

const CInterfaceStubVtbl * const _hnspluginapi_StubVtblList[] = 
{
    ( CInterfaceStubVtbl *) &_IHNSObjectStubVtbl,
    ( CInterfaceStubVtbl *) &_IHNSObjectListStubVtbl,
    ( CInterfaceStubVtbl *) &_IHNSProviderStubVtbl,
    ( CInterfaceStubVtbl *) &_IHNSPluginStubVtbl,
    0
};

PCInterfaceName const _hnspluginapi_InterfaceNamesList[] = 
{
    "IHNSObject",
    "IHNSObjectList",
    "IHNSProvider",
    "IHNSPlugin",
    0
};


#define _hnspluginapi_CHECK_IID(n)	IID_GENERIC_CHECK_IID( _hnspluginapi, pIID, n)

int __stdcall _hnspluginapi_IID_Lookup( const IID * pIID, int * pIndex )
{
    IID_BS_LOOKUP_SETUP

    IID_BS_LOOKUP_INITIAL_TEST( _hnspluginapi, 4, 2 )
    IID_BS_LOOKUP_NEXT_TEST( _hnspluginapi, 1 )
    IID_BS_LOOKUP_RETURN_RESULT( _hnspluginapi, 4, *pIndex )
    
}

const ExtendedProxyFileInfo hnspluginapi_ProxyFileInfo = 
{
    (PCInterfaceProxyVtblList *) & _hnspluginapi_ProxyVtblList,
    (PCInterfaceStubVtblList *) & _hnspluginapi_StubVtblList,
    (const PCInterfaceName * ) & _hnspluginapi_InterfaceNamesList,
    0, /* no delegation */
    & _hnspluginapi_IID_Lookup, 
    4,
    2,
    0, /* table of [async_uuid] interfaces */
    0, /* Filler1 */
    0, /* Filler2 */
    0  /* Filler3 */
};
#if _MSC_VER >= 1200
#pragma warning(pop)
#endif


#endif /* defined(_M_AMD64)*/

