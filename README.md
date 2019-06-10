# gophertunnel
A Minecraft library containing packages to create clients, servers, proxies and other tools.

## Overview
gophertunnel is composed of several packages that may be of use for creating Minecraft related tools.

package [minecraft](https://godoc.org/github.com/Sandertv/gophertunnel/minecraft): A package implementing connecting
to Minecraft Bedrock Edition servers and listening for Minecraft Bedrock Edition clients using a TCP style interface.

package [minecraft/auth](https://godoc.org/github.com/Sandertv/gophertunnel/minecraft/auth): A package implementing
Microsoft, XBOX Live and Minecraft account authentication.

package [minecraft/nbt](https://godoc.org/github.com/Sandertv/gophertunnel/minecraft/nbt): A package implementing the
Minecraft NBT format. Three variants of the format are implemented: The Java Edition variant (Big Endian) and
the Bedrock Edition variants (Little Endian, both with and without varints)

package [minecraft/protocol](https://godoc.org/github.com/Sandertv/gophertunnel/minecraft/protocol): A package
implementing the reading, writing and handling of packets found in the Minecraft Bedrock Edition protocol.

## Examples
Creating a Minecraft client that authenticates using an XBOX Live account and connects to a server:
```go
package main

import (
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func main() {
    conn, err := minecraft.Dialer{
        Email: "some@email.address",
        Password: "password",
    }.Dial("raknet", "mco.mineplex.com:19132")
    if err != nil {
        panic(err)
    }
    defer conn.Close()
    for {
    	pk, err := conn.ReadPacket()
    	if err != nil {
    		break
    	}
    	// Handle the incoming packet.
    	_ = pk
    	
    	// Send a packet to the server.
    	if err := conn.WritePacket(&packet.RequestChunkRadius{ChunkRadius: 32}); err != nil {
    		break
    	}
    }
}
```

Creating a Minecraft listener that can accept incoming clients and adapts the MOTD from another server:
```go
package main

import (
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft"
)

func main() {
	listener, err := minecraft.Listen("raknet", "0.0.0.0:19132")
	if err != nil {
		panic(err)
	}
	_ = listener.HijackPong("mco.mineplex.com:19132")

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go func() {
			// Process the connection on another goroutine as you would with TCP connections.
			defer conn.Close()
			for {
				// Read a packet from the client.
				if _, err := conn.(*minecraft.Conn).ReadPacket(); err != nil {
					return
				}
			}
		}()
	}
}

```