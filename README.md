# Globally Unique ID Generator

[![godoc](http://img.shields.io/badge/godoc-reference-blue.svg?style=flat)](https://godoc.org/github.com/jxskiss/xxid) [![license](http://img.shields.io/badge/license-MIT-red.svg?style=flat)](https://raw.githubusercontent.com/jxskiss/xxid/master/LICENSE)

Package xxid is a globally unique id generator library derived from the awesome
[xid](https://github.com/rs/xid) library by Olivier Poitrey <rs@dailymotion.com>.

For more details about xid, please follow [xid's doc](https://github.com/rs/xid/master/README.md).
The copy of xid's README below may be outdated.

----

This xxid package is different from xid in following ways:

1. xxid use 15 bytes for an ID object (xid use 12 bytes):

   - 4-byte value representing the seconds since epoch 15e8
   - 1-byte random or user specified flag (0-127)
   - 4-byte machine id or user specified IPv4 address
   - 2-byte pid or user specified port number
   - 4-byte counter (low 31 bits), starting with a random value

2. xxid use base62 for string representation, generates 20 chars which is same with xid,
   while keeping both the ID object and string representation K-ordered and sortable as xid.

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


----

Package xid is a globally unique id generator library, ready to be used safely directly in your server code.

Xid is using Mongo Object ID algorithm to generate globally unique ids with a different serialization (base64) to make it shorter when transported as a string:
https://docs.mongodb.org/manual/reference/object-id/

- 4-byte value representing the seconds since the Unix epoch,
- 3-byte machine identifier,
- 2-byte process id, and
- 3-byte counter, starting with a random value.

The binary representation of the id is compatible with Mongo 12 bytes Object IDs.
The string representation is using base32 hex (w/o padding) for better space efficiency
when stored in that form (20 bytes). The hex variant of base32 is used to retain the
sortable property of the id.

Xid doesn't use base64 because case sensitivity and the 2 non alphanum chars may be an
issue when transported as a string between various systems. Base36 wasn't retained either
because 1/ it's not standard 2/ the resulting size is not predictable (not bit aligned)
and 3/ it would not remain sortable. To validate a base32 `xid`, expect a 20 chars long,
all lowercase sequence of `a` to `v` letters and `0` to `9` numbers (`[0-9a-v]{20}`).

UUIDs are 16 bytes (128 bits) and 36 chars as string representation. Twitter Snowflake
ids are 8 bytes (64 bits) but require machine/data-center configuration and/or central
generator servers. xid stands in between with 12 bytes (96 bits) and a more compact
URL-safe string representation (20 chars). No configuration or central generator server
is required so it can be used directly in server's code.

| Name        | Binary Size | String Size    | Features
|-------------|-------------|----------------|----------------
| [UUID]      | 16 bytes    | 36 chars       | configuration free, not sortable
| [shortuuid] | 16 bytes    | 22 chars       | configuration free, not sortable
| [Snowflake] | 8 bytes     | up to 20 chars | needs machin/DC configuration, needs central server, sortable
| [MongoID]   | 12 bytes    | 24 chars       | configuration free, sortable
| xid         | 12 bytes    | 20 chars       | configuration free, sortable

[UUID]: https://en.wikipedia.org/wiki/Universally_unique_identifier
[shortuuid]: https://github.com/stochastic-technologies/shortuuid
[Snowflake]: https://blog.twitter.com/2010/announcing-snowflake
[MongoID]: https://docs.mongodb.org/manual/reference/object-id/

Features:

- Size: 12 bytes (96 bits), smaller than UUID, larger than snowflake
- Base32 hex encoded by default (20 chars when transported as printable string, still sortable)
- Non configured, you don't need set a unique machine and/or data center id
- K-ordered
- Embedded time with 1 second precision
- Unicity guaranteed for 16,777,216 (24 bits) unique ids per second and per host/process
- Lock-free (i.e.: unlike UUIDv1 and v2)

Best used with [zerolog](https://github.com/rs/zerolog)'s
[RequestIDHandler](https://godoc.org/github.com/rs/zerolog/hlog#RequestIDHandler).

Notes:

- Xid is dependent on the system time, a monotonic counter and so is not cryptographically secure. If unpredictability of IDs is important, you should not use Xids. It is worth noting that most of the other UUID like implementations are also not cryptographically secure. You shoud use libraries that rely on cryptographically secure sources (like /dev/urandom on unix, crypto/rand in golang), if you want a truly random ID generator.

References:

- http://www.slideshare.net/davegardnerisme/unique-id-generation-in-distributed-systems
- https://en.wikipedia.org/wiki/Universally_unique_identifier
- https://blog.twitter.com/2010/announcing-snowflake
- Python port by [Graham Abbott](https://github.com/graham): https://github.com/graham/python_xid
- Scala port by [Egor Kolotaev](https://github.com/kolotaev): https://github.com/kolotaev/ride

## Install

    go get github.com/rs/xid

## Usage

```go
guid := xid.New()

println(guid.String())
// Output: 9m4e2mr0ui3e8a215n4g
```

Get `xid` embedded info:

```go
guid.Machine()
guid.Pid()
guid.Time()
guid.Counter()
```

## Benchmark

Benchmark against Go [Maxim Bublis](https://github.com/satori)'s [UUID](https://github.com/satori/go.uuid).

```
BenchmarkXID        	20000000	        91.1 ns/op	      32 B/op	       1 allocs/op
BenchmarkXID-2      	20000000	        55.9 ns/op	      32 B/op	       1 allocs/op
BenchmarkXID-4      	50000000	        32.3 ns/op	      32 B/op	       1 allocs/op
BenchmarkUUIDv1     	10000000	       204 ns/op	      48 B/op	       1 allocs/op
BenchmarkUUIDv1-2   	10000000	       160 ns/op	      48 B/op	       1 allocs/op
BenchmarkUUIDv1-4   	10000000	       195 ns/op	      48 B/op	       1 allocs/op
BenchmarkUUIDv4     	 1000000	      1503 ns/op	      64 B/op	       2 allocs/op
BenchmarkUUIDv4-2   	 1000000	      1427 ns/op	      64 B/op	       2 allocs/op
BenchmarkUUIDv4-4   	 1000000	      1452 ns/op	      64 B/op	       2 allocs/op
```

Note: UUIDv1 requires a global lock, hence the performence degrading as we add more CPUs.

## Licenses

All source code is licensed under the [MIT License](https://raw.github.com/rs/xid/master/LICENSE).
