GC-friendly radix tree-based IP set
===

This is a Go implementation of an IP set, i.e. a data structure that holds a set of IP addresses and allows efficient
checks whether an IP address is contained within it.

Features
---

- GC-friendly. The internal representation is a slice of uint32.
- Zero-effort serialization/deserialization (on Little-Endian architectures).
- Support for both IPv4 and IPv6.
- Zero value is ready to use.

Basic Example
---

```go
package main

import "net/netip"

var s IPSet
s.Add(netip.MustParsePrefix("127.0.0.0/8"))

if s.Contains(netip.MustParseAddr("127.0.0.1")) {
	// ...
}

```