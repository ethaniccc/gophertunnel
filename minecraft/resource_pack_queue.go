package minecraft

import (
	"bytes"
	"fmt"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/resource"
)

// resourcePackQueue is used to aid in the handling of resource pack queueing and downloading. Only one
// resource pack is downloaded at a time.
type resourcePackQueue struct {
	packs           []*resource.Pack
	packsToDownload map[string]*resource.Pack
	currentPack     *resource.Pack
	currentOffset   int64

	packAmount       int
	downloadingPacks map[string]downloadingPack
	awaitingPacks    map[string]*downloadingPack
}

// downloadingPack is a resource pack that is being downloaded by a client connection.
type downloadingPack struct {
	buf           *bytes.Buffer
	chunkSize     int32
	size          int64
	expectedIndex int32
	newFrag       chan []byte
}

// Request 'requests' all resource packs passed, provided they all exist in the resourcePackQueue. If not,
// an error is returned.
func (queue *resourcePackQueue) Request(packs []string) error {
	queue.packsToDownload = make(map[string]*resource.Pack)
	for _, packUUID := range packs {
		found := false
		for _, pack := range queue.packs {
			// Mojang made some hack that merges the UUID with the version, so we need to combine that here
			// too in order to find the proper pack.
			if pack.UUID()+"_"+pack.Version() == packUUID {
				queue.packsToDownload[pack.UUID()] = pack
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("could not find resource pack %v", packUUID)
		}
	}
	return nil
}

// NextPack assigns the next resource pack to the current pack and returns true if successful. If there were
// no more packs to assign, false is returned. If ok is true, a packet with data info is returned.
func (queue *resourcePackQueue) NextPack() (pk *packet.ResourcePackDataInfo, ok bool) {
	for index, pack := range queue.packsToDownload {
		delete(queue.packsToDownload, index)

		queue.currentPack = pack
		queue.currentOffset = 0
		checksum := pack.Checksum()
		return &packet.ResourcePackDataInfo{
			UUID:          pack.UUID(),
			DataChunkSize: packChunkSize,
			ChunkCount:    int32(pack.DataChunkCount(packChunkSize)),
			Size:          int64(pack.Len()),
			Hash:          string(checksum[:]),
		}, true
	}
	return nil, false
}

// AllDownloaded checks if all resource packs in the queue are downloaded.
func (queue *resourcePackQueue) AllDownloaded() bool {
	return len(queue.packsToDownload) == 0
}
