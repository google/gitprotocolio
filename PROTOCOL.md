# Unofficial Git protocol syntax spec

Git protocol is a line-based text protocol except git-receive-pack's packfile
part. To define the protocol, we define its low-level token structure and its
higher-level syntax structure.

## Token

There are four Git protocol tokens. FlushPacket, DelimPacket, and BytesPacket.

### FlushPacket

FlushPacket is 4-byte array `[0x30, 0x30, 0x30, 0x30]`. This is `"0000"` in
ASCII encoding.

### DelimPacket

DelimPacket is 4-byte array `[0x30, 0x30, 0x30, 0x31]`. This is `"0001"` in
ASCII encoding.

### BytesPacket

BytesPacket is a byte array prefixed by length. This is similar to a Pascal
string. The length prefix is 4-byte, hexadecimal, zero-filled, ASCII-encoding
string. The length includes the length prefix itself. For example,
`"hello, world"` (12 characters) is encoded to `"0010hello, world"`. As the
encoding suggests, the payload size cannot exceed 65531 bytes.

### ErrorPacket

ErrorPacket is a special BytesPacket that the payload starts with "ERR ".

## Syntax

The syntax defines the valid token sequence of the Git protcol.

### Basic constructs

```
NUL ::= 0x00
SP  ::= 0x20
LF  ::= 0x0A
DECIMAL_DIGIT ::= c where isdigit(c)
DECIMAL_NUMBER ::= '0' | c DECIMAL_DIGIT* where '1' <= c && c <= '9'
HEX_DIGIT ::= c where isxdigit(c)
NAME ::= c+ where isgraph(c)
ANY_STR ::= c+ where isprint(c)
ANY_BYTES ::= c+ where 0x00 <= c && c <= 0xFF

OID_STR  ::= HEX_DIGIT{40}
REF_NAME ::= NAME
```

### Sideband

Sideband encoding is a packetization method of the Git protocol. For example,
the input `000eunpack ok\n` is sideband encoded to `0013\x01000eunpack ok\n`. As
you can see, the BytesPacket is nested.

Some parts of the Git protocol optionally need the sideband encoding. If we want
to define a formal grammar of the protocol, we should define two languages.

We define a special form `MaybeSidebandEncoding` for the syntax language. When
this appears in the syntax, it represents the language it takes as-is, or a
language that is sideband encoded. The sideband encoded languages is defined as
follows.

```
SIDEBAND_STREAM ::= BytePacket((0x01 | 0x02 | 0x03) ANY_BYTES)*
```

The byte stream consists of BytePackets starting from 0x01 is in the input
language. The byte stream of 0x02 is a progress message. This progress
message is usually just shown to the end-user and does not have a meaning in
Git. The byte stream of 0x03 is an error message. This error message is treated
same as ErrorPacket.

### Protocol V2 constructs

```
PROTOCOL_V2_HEADER ::= BytesPacket("version 2" LF)
                       CAPABILITY_LINE*
                       FlushPacket()

CAPABILITY_LINE  ::= BytesPacket(CAPABILITY_KEY ("=" CAPABILITY_VALUE) LF)
CAPABILITY_KEY   ::= c+ where isalpha(c) || isdigit(c) || c == '-' || c == '_'
CAPABILITY_VALUE ::= c+ where isalpha(c) || isdigit(c) || strchr(" -_.,?\/{}[]()<>!@#$%^&*+=:;", c) != NULL

PROTOCOL_V2_REQ ::= BytesPacket("command=" CAPABILITY_KEY)
                    CAPABILITY_LINE*
                    (DelimPacket() BytesPacket(ANY_BYTES)*)?
                    FlushPacket()

PROTOCOL_V2_RESP ::= BytesPacket(ANY_BYTES)*
                     FlushPacket()
```

### HTTP transport /info/refs

```
HTTP_INFO_REFS ::= INFO_REFS_V0_V1 | PROTOCOL_V2_HEADER

INFO_REFS_V0_V1 ::= BytesPacket("# service=" SERVICE_NAME LF)
                    FlushPacket()
                    (REFS)?
                    FlushPacket()
REFS            ::= BytesPacket(OID_STR SP REF_NAME NUL CAPABILITY_LIST? LF)
                    REF*
REF             ::= BytesPacket(OID_STR SP REF_NAME LF)
SERVICE_NAME    ::= NAME
CAPABILITY_LIST ::= CAPABILITY
                  | CAPABILITY (SP CAPABILITY_LIST)?
CAPABILITY      ::= NAME
```

### HTTP transport /git-upload-pack

```
HTTP_UPLOAD_PACK_REQ ::= UPLOAD_PACK_V0_V1_REQ | PROTOCOL_V2_REQ

UPLOAD_PACK_V0_V1_REQ ::= BytesPacket("want" SP OID_STR (SP CAPABILITY_LIST)? LF)
                          CLIENT_WANT*
                          SHALLOW_REQUEST*
                          (DEPTH_REQUEST)?
                          (FILTER_REQUEST)?
                          FlushPacket()
                          NEXT_NEGOTIATION
NEXT_NEGOTIATION      ::= CLIENT_HAVE*
                          FlushPacket() NEXT_NEGOTIATION | BytesPacket("done"))
CLIENT_WANT           ::= BytesPacket("want" SP OID_STR LF)
SHALLOW_REQUEST       ::= BytesPacket("shallow" SP OID_STR LF)
DEPTH_REQUEST         ::= BytesPacket("deepen" SP DECIMAL_NUMBER LF)
                        | BytesPacket("deepen-since" SP DECIMAL_NUMBER LF)
                        | BytesPacket("deepen-not" SP REF_NAME LF)
FILTER_REQUEST        ::= BytesPacket("filter" SP ANY_STR LF)
CLIENT_HAVE           ::= BytesPacket("have" SP OID_STR LF)

HTTP_UPLOAD_PACK_RESP ::= UPLOAD_PACK_V0_V1_RESP | PROTOCOL_V2_RESP

UPLOAD_PACK_V0_V1_RESP ::= SHALLOW_UPDATE*
                           FlushPacket?
                           ACKNOWLEDGEMENT+
                           (MaybeSidebandEncoding(PACK_FILE))?
                           FlushPacket()
SHALLOW_UPDATE         ::= BytesPacket("shallow" SP OID_STR LF)*
                           BytesPacket("unshallow" SP OID_STR LF)*
ACKNOWLEDGEMENT        ::= BytesPacket("ACK" SP OID_STR SP ("continue" | "common" | "ready") LF)*
                         | BytesPacket("ACK" SP OID_STR LF)
                         | BytesPacket("NAK" LF)
```

### HTTP transport /git-receive-pack

```
HTTP_RECEIVE_PACK_REQ ::= RECEIVE_PACK_V0_V1_REQ | PROTOCOL_V2_REQ

RECEIVE_PACK_V0_V1_REQ ::= CLIENT_SHALLOW*
                           (COMMAND_LIST | PUSH_CERT)
                           (PUSH_OPTION* FlushPacket())?
                           (PACK_FILE)?
CLIENT_SHALLOW         ::= BytesPacket("shallow" SP OID_STR LF)
COMMAND_LIST           ::= BytesPacket(COMMAND NUL SP? CAPABILITY_LIST? LF)
                           BytesPacket(COMMAND LF)*
                           FlushPacket()
COMMAND                ::= OID_STR SP OID_STR SP REF_NAME
PUSH_CERT              ::= BytesPacket("push-cert" NUL SP? CAPABILITIY_LIST? LF)
                           BytesPacket("certificate version 0.1" LF)
                           BytesPacket("pusher" SP ANY_STR LF)
                           (BytesPacket("pushee" SP ANY_STR LF))?
                           BytesPacket("nonce" SP ANY_STR LF)
                           BytesPacket("push-option" SP ANY_STR LF)*
                           BytesPacket(LF)
                           BytesPacket(COMMAND LF)*
                           BytesPacket(GPG_SIGNATURE_LINES LF)*
                           BytesPacket("push-cert-end" LF)
GPG_SIGNATURE_LINES    ::= ANY_BYTES
PUSH_OPTION            ::= BytesPacket(ANY_STR LF)
PACK_FILE              ::= "PACK" ANY_BYTES

HTTP_RECEIVE_PACK_RESP ::=
                         | MaybeSidebandEncoding(RECEIVE_PACK_V0_V1_RESP)
                           FlushPacket()
                         | PROTOCOL_V2_RESP

RECEIVE_PACK_V0_V1_RESP ::=
                          | BytesPacket("unpack" SP ("ok" | ANY_STR) LF)
                            REF_UPDATE_RESULT+
REF_UPDATE_RESULT       ::= BytesPacket("ok" SP REF_NAME LF)
                          | BytesPacket("ng" SP REF_NAME SP ANY_STR LF)
```

## Questions

### What's wrong with the capability list

In case you haven't noticed this, let us extract the rules around capabilities:

```
INFO_REFS_V0_V1 ::= ...
                    BytesPacket(OID_STR SP REF_NAME NUL CAPABILITY_LIST? LF)
UPLOAD_PACK_V0_V1_REQ ::= BytesPacket("want" SP OID_STR (SP CAPABILITY_LIST)? LF)
COMMAND_LIST           ::= BytesPacket(COMMAND NUL SP? CAPABILITY_LIST? LF)
PUSH_CERT              ::= BytesPacket("push-cert" NUL CAPABILITIY_LIST? LF)
```

The way the capabilities are sent is inconsistent. They're even not in the first
line in some cases. We have no idea what's going on, but anyway the protocol v2
will get rid of this.

### What's wrong with the push-options

When a client pushes a push certificate, it sends the push options twice. We
have no idea what's going on.

### Is push cert's pushee optional?

This can be a bug. Looking at builtin/send-pack.c, args.url is not set for this
path always.
