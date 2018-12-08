package worldDataFormat

import (
	//"log"
	"io"
	"bytes"
	"encoding/binary"
)

const FrameSetVersion = 1

type Frame struct {
	Elevations *ElevationFrame
	Age *AgeFrame
	Satallite *SatalliteFrame
}

type FrameSet struct {
	frames []Frame

	typesRead uint64
	typeOffsets []uint64
}

func (set *FrameSet)AddFrame(frame Frame) {
	set.frames = append(set.frames, frame)
}

func (set *FrameSet)Frames() []Frame {
	return set.frames
}

func (set *FrameSet)WriteFull(target io.Writer, isCompressed bool, typesToWrite uint64) error {
	return set.internalWrite(target, isCompressed, false, typesToWrite)
}

func (set *FrameSet)WriteRendered(target io.Writer, isCompressed bool, typesToWrite uint64) error {
	return set.internalWrite(target, isCompressed, true, typesToWrite)
}

func (set *FrameSet)writeHeader(target io.Writer, typeLengths []uint64) error {
	var err error
	if len(set.frames) == 0 {
		//log.Print("No frames to write")
		return NoData
	}

	var headerSize uint64
	headerSize = 8
	for _, val := range typeLengths {
		if val > 0 {
			headerSize += 8
		}
	}
	// caculate total size, includes the 8 bytes to store to total size
	var totalSize uint64
	totalSize = 24 + headerSize  // up to begining of data
	for _, size := range typeLengths {
		totalSize += size
	}

	// write the stuff
	err = binary.Write(target, binary.LittleEndian, totalSize)
	if err != nil {
		return err
	}
	err = binary.Write(target, binary.LittleEndian, uint64(FrameSetVersion))
	if err != nil {
		return err
	}
	err = binary.Write(target, binary.LittleEndian, headerSize)
	if err != nil {
		return err
	}
	err = binary.Write(target, binary.LittleEndian, uint64(len(set.frames)))
	if err != nil {
		return err
	}
	var offset uint64 = 0
	for _, length := range typeLengths {
		if length > 0 {
			err = binary.Write(target, binary.LittleEndian, offset)
			if err != nil {
				return err
			}
			offset += length
		}
	}


	return nil
}

func (set *FrameSet)internalWrite(target io.Writer, isCompressed bool, isRendered bool, typesToWrite uint64) error {
	var err error

	// verify can write
	for _, theFrame := range set.frames {
		if (AgeFrameFlag & typesToWrite) > 0 && theFrame.Age == nil {
			return MissingData
		}
		if (ElevationFrameFlag & typesToWrite) > 0 && theFrame.Elevations == nil {
			return MissingData
		}
		if (SatalliteFrameFlag & typesToWrite) > 0 && theFrame.Satallite == nil {
			return MissingData
		}
	}

	// write all frames to temporary buffers
	var ageBuffer, elevationsBuffer, satalliteBuffer bytes.Buffer
	for index, theFrame := range set.frames {
		if (AgeFrameFlag & typesToWrite) > 0 {
			err = theFrame.Age.internalWrite(&ageBuffer)
			if err != nil {
				return err
			}
		}
		if (ElevationFrameFlag & typesToWrite) > 0 {
			var prevFrame *ElevationFrame
			if index == 0 {
				prevFrame = nil
			} else {
				prevFrame = set.frames[index - 1].Elevations
			}

			err = theFrame.Elevations.internalWrite(&elevationsBuffer, isCompressed, isRendered, prevFrame)
			if err != nil {
				return err
			}
		}
		if (SatalliteFrameFlag & typesToWrite) > 0 {
			err = theFrame.Satallite.internalWrite(&satalliteBuffer, isCompressed, nil)
			if err != nil {
				return err
			}
		}
	}

	var typeLengths []uint64
	typeLengths = append(typeLengths, uint64(ageBuffer.Len()))
	typeLengths = append(typeLengths, uint64(elevationsBuffer.Len()))
	typeLengths = append(typeLengths, uint64(satalliteBuffer.Len()))


	err = set.writeHeader(target, typeLengths)
	if err != nil {
		return err
	}

	// write the data
	var n int64
	n, err = ageBuffer.WriteTo(target)
	if err != nil {
		return err
	}
	if uint64(n) != typeLengths[0] {
		// do something!
	}
	n, err = elevationsBuffer.WriteTo(target)
	if err != nil {
		return err
	}
	if uint64(n) != typeLengths[1] {
		// do something
	}
	n, err = satalliteBuffer.WriteTo(target)
	if err != nil {
		return err
	}
	if uint64(n) != typeLengths[2] {
		// do something
	}

	return nil
}

func (set *FrameSet)internalReadHeader(source io.Reader) error {
	var err error

	// check total size
	var totalSize uint64
	err = binary.Read(source, binary.LittleEndian, &totalSize)
	if err != nil {
		return err
	} else {
		//log.Printf("Total frame set size: %d", totalSize)
	}

	// check version
	var version uint64
	err = binary.Read(source, binary.LittleEndian, &version)
	if err != nil {
		return err
	}

	// check Header Length (with only one offset)
	var headerLen uint64
	err = binary.Read(source, binary.LittleEndian, &headerLen)
	if err != nil {
		return err
	}

	// check frame count
	var frameCount uint64
	err = binary.Read(source, binary.LittleEndian, &frameCount)
	if err != nil {
		return err
	} else {
		//log.Printf("Frames in frame set: %d", frameCount)
	}
	set.frames = make([]Frame, frameCount)

	var offsetCount uint64 = (headerLen - 8)/8

	set.typeOffsets = make([]uint64, offsetCount)

	// read offsets
	var i uint64
	for i = 0; i < offsetCount; i++ {
		err = binary.Read(source, binary.LittleEndian, &set.typeOffsets[i])
		if err != nil {
			return err
		} else {
			//log.Printf("Type offset of: %d", set.typeOffsets[i])
		}
	}
	
	return nil
}

func internalReadFrameSet(source io.Reader, typesWritten uint64) (FrameSet, error) {
	var readSet FrameSet

	err := readSet.internalReadHeader(source)
	if err != nil {
		return readSet, err
	}

	// read age frames
	if typesWritten & AgeFrameFlag > 0 {
		//log.Print("reading ages")
		for index, _ := range readSet.frames {
			ageFrame, err := internalReadAgeFrame(source)
			if err != nil {
				return readSet, err
			}
			readSet.frames[index].Age = &ageFrame
		}
	}

	// read elevation frames
	if typesWritten & ElevationFrameFlag > 0 {
		//log.Print("reading elevations")
		for index, _ := range readSet.frames {
			elevationFrame, err := internalReadElevationFrame(source)
			if err != nil {
				return readSet, err
			}
			readSet.frames[index].Elevations = &elevationFrame
		}
	}

	return readSet, nil
}