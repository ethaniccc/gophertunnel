package minecraft

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/google/uuid"
	"github.com/sandertv/go-raknet"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/device"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"log"
	rand2 "math/rand"
	"net"
	"os"
	"time"
)

// Dialer allows specifying specific settings for connection to a Minecraft server.
// The zero value of Dialer is used for the package level Dial function.
type Dialer struct {
	// ErrorLog is a log.Logger that errors that occur during packet handling of servers are written to. By
	// default, ErrorLog is set to one equal to the global logger.
	ErrorLog *log.Logger

	// ClientData is the client data used to login to the server with. It includes fields such as the skin,
	// locale and UUIDs unique to the client. If empty, a default is sent produced using defaultClientData().
	ClientData login.ClientData

	// Email is the email used to login to the XBOX Live account. If empty, no attempt will be made to login,
	// and an unauthenticated login request will be sent.
	Email string
	// Password is the password used to login to the XBOX Live account. If Email is non-empty, a login attempt
	// will be made using this password.
	Password string

	// PacketFunc is called whenever a packet is read from or written to the connection returned when using
	// Dialer.Dial(). It includes packets that are otherwise covered in the connection sequence, such as the
	// Login packet. The function is called with the header of the packet and its raw payload, the address
	// from which the packet originated, and the destination address.
	PacketFunc func(header packet.Header, payload []byte, src, dst net.Addr)
}

// Dial dials a Minecraft connection to the address passed over the network passed. The network must be "tcp",
// "tcp4", "tcp6", "unix", "unixpacket" or "raknet". A Conn is returned which may be used to receive packets
// from and send packets to.
//
// A zero value of a Dialer struct is used to initiate the connection. A custom Dialer may be used to specify
// additional behaviour.
func Dial(network string, address string) (conn *Conn, err error) {
	return Dialer{}.Dial(network, address)
}

// Dial dials a Minecraft connection to the address passed over the network passed. The network must be "tcp",
// "tcp4", "tcp6", "unix", "unixpacket" or "raknet". A Conn is returned which may be used to receive packets
// from and send packets to.
// Specific fields in the Dialer specify additional behaviour during the connection, such as authenticating
// to XBOX Live and custom client data.
func (dialer Dialer) Dial(network string, address string) (conn *Conn, err error) {
	var netConn net.Conn

	switch network {
	case "raknet":
		// If the network is specifically 'raknet', we use the raknet library to dial a RakNet connection.
		netConn, err = raknet.Dial(address)
	default:
		// If not set to 'raknet', we fall back to the default net.Dial method to find a proper connection for
		// the network passed.
		netConn, err = net.Dial(network, address)
	}
	if err != nil {
		return nil, err
	}
	key, _ := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)

	var chainData string
	if dialer.Email != "" {
		chainData, err = authChain(dialer.Email, dialer.Password, key)
		if err != nil {
			return nil, err
		}
	}
	if dialer.ErrorLog == nil {
		dialer.ErrorLog = log.New(os.Stderr, "", log.LstdFlags)
	}
	conn = newConn(netConn, key, dialer.ErrorLog)
	conn.clientData = defaultClientData(address)
	conn.packetFunc = dialer.PacketFunc

	var emptyClientData login.ClientData
	if dialer.ClientData != emptyClientData {
		// If a custom client data struct was set, we change the default.
		conn.clientData = dialer.ClientData
	}
	conn.expect(packet.IDServerToClientHandshake, packet.IDPlayStatus)

	go listenConn(conn, dialer.ErrorLog)

	request := login.Encode(chainData, conn.clientData, key)
	if err := conn.WritePacket(&packet.Login{ConnectionRequest: request, ClientProtocol: protocol.CurrentProtocol}); err != nil {
		return nil, err
	}
	select {
	case <-conn.connected:
		// We've connected successfully. We return the connection and no error.
		return conn, nil
	case <-conn.close:
		// The connection was closed before we even were fully 'connected', so we return an error.
		conn.close <- true
		return nil, fmt.Errorf("connection timeout")
	}
}

// listenConn listens on the connection until it is closed on another goroutine.
func listenConn(conn *Conn, logger *log.Logger) {
	defer func() {
		_ = conn.Close()
	}()
	for {
		// We finally arrived at the packet decoding loop. We constantly decode packets that arrive
		// and push them to the Conn so that they may be processed.
		packets, err := conn.decoder.Decode()
		if err != nil {
			if !raknet.ErrConnectionClosed(err) {
				logger.Printf("error reading from client connection: %v\n", err)
			}
			return
		}
		for _, data := range packets {
			if err := conn.handleIncoming(data); err != nil {
				logger.Printf("error: %v", err)
				return
			}
		}
	}
}

// authChain requests the Minecraft auth JWT chain using the credentials passed. If successful, an encoded
// chain ready to be put in a login request is returned.
func authChain(email, password string, key *ecdsa.PrivateKey) (string, error) {
	// Obtain the Live token, and using that the XSTS token.
	liveToken, err := auth.RequestLiveToken(email, password)
	if err != nil {
		return "", fmt.Errorf("error obtaining Live token: %v", err)
	}
	xsts, err := auth.RequestXSTSToken(liveToken)
	if err != nil {
		return "", fmt.Errorf("error obtaining XSTS token: %v", err)
	}

	// Obtain the raw chain data using the
	chain, err := auth.RequestMinecraftChain(xsts, key)
	if err != nil {
		return "", fmt.Errorf("error obtaining Minecraft auth chain: %v", err)
	}
	return chain, nil
}

// defaultClientData returns a valid, mostly filled out ClientData struct using the connection address
// passed, which is sent by default, if no other client data is set.
func defaultClientData(address string) login.ClientData {
	rand2.Seed(time.Now().Unix())
	return login.ClientData{
		ClientRandomID:   rand2.Int63(),
		DeviceOS:         device.Win10,
		GameVersion:      protocol.CurrentVersion,
		DeviceID:         uuid.Must(uuid.NewRandom()).String(),
		LanguageCode:     "en_UK",
		ThirdPartyName:   "Steve",
		SelfSignedID:     uuid.Must(uuid.NewRandom()).String(),
		SkinGeometryName: "geometry.humanoid",
		ServerAddress:    address,
		SkinID:           uuid.Must(uuid.NewRandom()).String(),
		SkinData:         base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{0}, 32*64*4)),
	}
}
