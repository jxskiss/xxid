# Globally Unique ID Generator

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/jxskiss/xxid) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/jxskiss/xxid/master/LICENSE)

Package xxid is a globally unique id generator library derived from the awesome
[xid](https://github.com/rs/xid/) library by [Olivier Poitrey](https://github.com/rs/).

----

Package xid is a globally unique id generator library, ready to safely be used directly in your server code.

Xid uses the Mongo Object ID algorithm to generate globally unique ids with a different serialization (base64) to make it shorter when transported as a string: https://docs.mongodb.org/manual/reference/object-id/

- 4-byte value representing the seconds since the Unix epoch,
- 3-byte machine identifier,
- 2-byte process id, and
- 3-byte counter, starting with a random value.

The binary representation of the id is compatible with Mongo 12 bytes Object IDs. The string representation is using base32 hex (w/o padding) for better space efficiency when stored in that form (20 bytes). The hex variant of base32 is used to retain the sortable property of the id.

For more information and updated docs abort xid, please see [xid's README](https://github.com/rs/xid/master/README.md).

----

This xxid package is different from xid in following ways:

1. xxid use 15 bytes for an ID object instead of xid's 12 bytes:

   - 4-byte value representing the seconds since epoch 15e8
   - 1-byte random or user specified flag (0-127)
   - 4-byte machine id or user specified IPv4 address
   - 2-byte pid or user specified port number
   - 4-byte counter (low 31 bits), starting with a random value
   
   Thus it's not compatible with either Mongo Object ID nor xid.

2. xxid use base62 for string representation, generates 20 chars which is same length with xid,
   while keeping both the ID object and string representation K-ordered and sortable like xid.

3. xxid provides a `Generator` type for user to specify flag, IP address and port number,
   this is useful to mark where the id is generated, for example, use flag for IDC, and
   IP address, port number for service instance.

   The flag/IP/port design enables the ability to develop coordinator-free distributed service.
   For example, if you have a stateful websocket server cluster which holds many client connections
   on different service instance, you can use xxid as an identifier for each connection.
   When downstream service want to talk to a client, they don't need to do an extra query to see
   which server the connection is on, they can get it by just checking the xxid identifier, good!

4. xxid provides an int64 representation of an ID consisting of timestamp and counter,
   this is more efficient than string representation to be used inside process.

5. xxid provides an UUID representation of an ID, which may help to work with other systems.

6. xxid use 1 more byte for machine id, and 1 more byte for counter, thus it has smaller chance
   to encounter an ID collision, though you should never encounter one using the Mongo Object ID algorithm ~

Hope you enjoy this package, and any issues is welcome 😃

## Install

    go get github.com/jxskiss/xxid

## Usage

```go
guid := xxid.New()
fmt.Println(guid.String(), guid.UUID(), guid.Short())
// 0lZttCu0sDkrIgnwnolX 035bc246-d1f7-d050-dc15-00c8367a8bc3 121000306362977219

ip := net.ParseIP("10.9.8.7")
gen := xxid.NewGenerator().UseFlag(123).UseIP(ip).UsePort(8888)
guid = gen.New()
fmt.Println(guid.Time(), guid.Flag(), guid.MachineIP(), guid.Port())
// 2019-04-27 14:05:58 +0800 CST 123 10.9.8.7 8888
```

## FAQ

Q: Why not using the standard Unix epoch?

A: Using a more recently epoch gives a much higher useful lifetime of around 136 years from July 2017,
   15e8 was picked to be easy to remember.
 