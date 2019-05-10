package worldDataFormat

import (
	//"bytes"

	"encoding/binary"
	"errors"
	"io"
	"log"
)

const WorldSimulationVersion = 2

/* There are several modes of reading and writing
 * A standard write writes only the information already given to the WorldSimulation object
 * A streaming write will write all new frame sets added, but will not retain them in memory once written
 * A streaming read will read a set number of framesets ahead of the consumption of the data, reading is always streamed, ie. files not read in full
 * When streaming for realtime consumption, the ReadToWriter method should be used
 *   Whenever a new frame should be written, call WriteNext()
 */

type WorldSimulation struct {
	frameSets       []FrameSet
	subdivisions    int
	subdivisionsSet bool

	frameSetStream      chan FrameSet
	writeFinishedSignal chan bool
	isStreamingWrite    bool

	isCompressed bool
	isRendered   bool
	typesToWrite uint64

	typesRead uint64

	source io.ReadSeeker
	target io.Writer
}

func (sim *WorldSimulation) AddFrameSet(set FrameSet) {
	if sim.isStreamingWrite {
		sim.frameSetStream <- set
	} else {
		sim.frameSets = append(sim.frameSets, set)
	}
}

// blocks until all writes are finished
func (sim *WorldSimulation) FlushAndCloseWriteStream() {
	// only close if is streaming write
	if sim.isStreamingWrite {
		close(sim.frameSetStream)
		<-sim.writeFinishedSignal
	}
}

func (sim *WorldSimulation) FrameSets() []FrameSet {
	return sim.frameSets
}

func (sim *WorldSimulation) SetSubdivisions(subdivisions int) {
	sim.subdivisions = subdivisions
	sim.subdivisionsSet = true
}

func (sim *WorldSimulation) Subdivisions() int {
	return sim.subdivisions
}

func (sim *WorldSimulation) WriteFull(target io.Writer, isCompressed bool, typesToWrite uint64) error {
	return sim.internalWrite(target, isCompressed, false, typesToWrite)
}

// func (sim *WorldSimulation) WriteNext() error {
// 	return sim.internalWriteNext()
// }

func (sim *WorldSimulation) WriteRendered(target io.Writer, isCompressed bool, typesToWrite uint64) error {
	return sim.internalWrite(target, isCompressed, true, typesToWrite)
}

func (sim *WorldSimulation) ReadToWriter(source io.ReadSeeker, target io.Writer, isCompressed, isRendered bool, typesToWrite uint64) error {
	return sim.internalReadToWriter(source, target, 30, isCompressed, isRendered, typesToWrite)
}

func (sim *WorldSimulation) StreamWriteRendered(target io.WriteSeeker, isCompressed bool, typesToWrite uint64) chan error {
	sim.frameSetStream = make(chan FrameSet, 1)
	sim.isStreamingWrite = true

	var errChan chan error = make(chan error, 1)

	// probably should put this elsewhere, but for now
	sim.writeFinishedSignal = make(chan bool, 1)

	go func() {
		errChan <- sim.internalStreamWrite(target, isCompressed, true, typesToWrite)
		close(errChan)
	}()
	return errChan
}

func (sim *WorldSimulation) writeHeader(target io.Writer, typesToWrite uint64) error {
	var err error
	err = binary.Write(target, binary.LittleEndian, uint64(WorldSimulationVersion))
	if err != nil {
		return err
	}
	err = binary.Write(target, binary.LittleEndian, uint64(24))
	if err != nil {
		return err
	}
	err = binary.Write(target, binary.LittleEndian, uint64(sim.subdivisions))
	if err != nil {
		return err
	}
	err = binary.Write(target, binary.LittleEndian, uint64(len(sim.frameSets)))
	if err != nil {
		return err
	}
	err = binary.Write(target, binary.LittleEndian, typesToWrite)
	if err != nil {
		return err
	}
	return nil
}

func (sim *WorldSimulation) internalWrite(target io.Writer, isCompressed bool, isRendered bool, typesToWrite uint64) error {
	var err error

	if sim.subdivisionsSet == false {
		return MissingGridDefinition
	}

	err = sim.writeHeader(target, typesToWrite)
	if err != nil {
		return err
	}

	// write each frameset
	for _, set := range sim.frameSets {
		err = set.internalWrite(target, isCompressed, isRendered, typesToWrite)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sim *WorldSimulation) internalWriteNext() error {
	var err error

	set, err := internalReadFrameSet(sim.source, sim.typesRead)
	if err != nil {
		log.Printf("Error reading set: %s", err)
		return NoData
	}

	//log.Print("Writing next set from stream")

	err = set.internalWrite(sim.target, sim.isCompressed, sim.isRendered, sim.typesToWrite)
	if err != nil {
		return err
	}

	return nil
}

func (sim *WorldSimulation) internalStreamWrite(target io.WriteSeeker, isCompressed bool, isRendered bool, typesToWrite uint64) error {
	if sim.frameSetStream == nil || !sim.isStreamingWrite {
		return errors.New("Not set up to stream sets")
	}
	// write any previously added frames
	var err error
	err = sim.internalWrite(target, isCompressed, isRendered, typesToWrite)
	if err != nil {
		return err
	}

	for set := range sim.frameSetStream {
		err = set.internalWrite(target, isCompressed, isRendered, typesToWrite)
		if err != nil {
			sim.writeFinishedSignal <- true // indicate that all sets are finished writing
			return err
		}
	}

	// need to update frame count

	sim.writeFinishedSignal <- true // indicate that all sets are finished writing

	return nil
}

func (sim *WorldSimulation) readHeader(source io.ReadSeeker) error {
	var err error
	// read version
	var version uint64
	err = binary.Read(source, binary.LittleEndian, &version)
	if err != nil {
		return err
	} else if version != WorldSimulationVersion {
		return IncompatibleVersion
	}
	// read header length
	var length uint64
	err = binary.Read(source, binary.LittleEndian, &length)
	if err != nil {
		return err
	}
	// read subdivisions
	var subdivCount uint64
	err = binary.Read(source, binary.LittleEndian, &subdivCount)
	if err != nil {
		return err
	}
	sim.subdivisions = int(subdivCount)
	sim.subdivisionsSet = true
	// read frame count
	var frameCount uint64
	err = binary.Read(source, binary.LittleEndian, &frameCount)
	if err != nil {
		return err
	}
	// read type field
	var typesWritten uint64
	err = binary.Read(source, binary.LittleEndian, &typesWritten)
	if err != nil {
		return err
	} else {
		//log.Printf("Types written read as: %d", typesWritten)
	}
	sim.typesRead = typesWritten

	return nil
}

func (sim *WorldSimulation) internalReadToWriter(source io.ReadSeeker, target io.Writer, setCount int, isCompressed, isRendered bool, typesToWrite uint64) error {
	sim.source = source
	sim.target = target
	sim.isCompressed = isCompressed
	sim.isRendered = isRendered
	sim.typesToWrite = typesToWrite

	// start read
	err := sim.readHeader(source)
	if err != nil {
		return err
	}
	// write our header
	sim.writeHeader(target, typesToWrite)

	// read sets and collect in requested size
	var readFrames []Frame
	for {
		// read set
		set, err := internalReadFrameSet(sim.source, sim.typesRead)
		if err == io.EOF {
			// write last frame?
			break
		} else if err != nil {
			log.Printf("Error reading next frame set: %s", err)
			return err
		}

		readFrames = append(readFrames, set.Frames()...)

		// write as many as possible
		for {
			if len(readFrames) < setCount {
				break
			}

			var writeSet FrameSet
			for i := 0; i < setCount; i++ {
				writeSet.AddFrame(readFrames[i])
			}

			err = writeSet.internalWrite(sim.target, sim.isCompressed, sim.isRendered, sim.typesToWrite)
			if err != nil {
				log.Printf("Error writing frame set")
			}

			readFrames = readFrames[setCount:] // TODO: will this discard frames during GC?
		}
	}

	return nil
}
