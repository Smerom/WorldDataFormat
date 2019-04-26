package worldDataFormat

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"io" //"log"
	"io/ioutil"
	"math"
)

type ElevationFrame struct {
	sealevel           float64
	elevations         []float64 // external getter and setter provided
	renderedElevations []int16   // currently not externally accesible

	data []byte // data stored here after read as we might not need to decompress it

	// frame attributes used in header
	dataReadSize     uint64
	isFromCompressed bool
	isFromRendered   bool
}

func (frame *ElevationFrame) SetSealevel(value float64) {
	frame.sealevel = value
}

// need to update rendered, vs unrendered state when setting elevation values
func (frame *ElevationFrame) SetElevations(values []float64) {
	frame.elevations = values
	frame.renderedElevations = nil
	frame.isFromRendered = false
}

func (frame *ElevationFrame) Elevations() []float64 {
	if frame.isFromRendered {
		return nil
	}
	return frame.elevations
}

// writes frame as loss-less float64s
func (frame *ElevationFrame) WriteFull(target io.Writer, isCompressed bool) error {
	return frame.internalWrite(target, isCompressed, false, nil)
}

// renders frame to a color scheme, information lost in data written
func (frame *ElevationFrame) WriteRendered(target io.Writer, isCompressed bool) error {
	return frame.internalWrite(target, isCompressed, true, nil)
}

// reads in frame header and data from source
func ReadElevationFrame(source io.Reader) (ElevationFrame, error) {
	return internalReadElevationFrame(source)
}

// writes header values describing the frame data
func (frame *ElevationFrame) writeHeader(target io.Writer, dataSize uint64, flags uint64) error {
	//log.Printf("Writing size %d", dataSize)
	err := binary.Write(target, binary.LittleEndian, dataSize)
	if err != nil {
		return err
	}
	//log.Printf("Writing flags: %064b", flags)
	err = binary.Write(target, binary.LittleEndian, flags)
	if err != nil {
		return err
	}
	return nil
}

// writes header followed by elevation frame data in the specified format (compressed or not, rendered or not)
// prevFrame used for time series compression
func (frame *ElevationFrame) internalWrite(target io.Writer, isCompressed, isRendered bool, prevFrame *ElevationFrame) error {
	var err error
	// must have data somewhere
	if len(frame.elevations) == 0 && len(frame.renderedElevations) == 0 && frame.data == nil {
		return NoData
	}
	var flags uint64
	if isCompressed {
		flags = flags | IsCompressedFlag
	}
	if isRendered {
		flags = flags | IsRenderedFlag
	} else if frame.isFromRendered {
		return InvalidData // can't unrender our data
	}

	// check if we have valid stored data
	if frame.isFromRendered {
		//log.Print("writing from rendered")

		// compress or decompress if needed
		var dataToWrite []byte
		if isCompressed && !frame.isFromCompressed {
			//log.Print("rendered needs compression")
			// compress it
			var finalizedBuffer bytes.Buffer
			zipWriter, err := gzip.NewWriterLevel(&finalizedBuffer, gzip.BestCompression)
			if err != nil {
				return err
			}
			_, err = zipWriter.Write(frame.data)
			if err != nil {
				return err
			}
			zipWriter.Close()
			if err != nil {
				return err
			}
			dataToWrite = finalizedBuffer.Bytes()
		} else if !isCompressed && frame.isFromCompressed {
			//log.Print("rendered needs decompression")
			// decompress it
			zipReader, err := gzip.NewReader(bytes.NewReader(frame.data))
			if err != nil {
				return err
			}

			dataToWrite, err = ioutil.ReadAll(zipReader)
			if err != nil {
				return err
			}

		} else {
			//log.Print("writing rendered data directly")
			// write the data read exactly as we got it
			dataToWrite = frame.data
		}

		err = frame.writeHeader(target, uint64(len(dataToWrite)), flags)
		if err != nil {
			return err
		}

		_, err = target.Write(dataToWrite)
		if err != nil {
			return err
		}

	} else {
		// we have full data currently
		// render if needed
		if isRendered && frame.renderedElevations == nil {
			// create our rendering
			frame.internalRenderElevations() // default to relative for now
		}

		var data bytes.Buffer

		if isRendered {
			// write values
			for index, rendered := range frame.renderedElevations {
				// if we have a previous frame, take difference for higher statistical redundancy before compression
				var valueToWrite int16
				if prevFrame != nil {
					valueToWrite = rendered - prevFrame.renderedElevations[index]
				} else {
					valueToWrite = rendered
				}
				err = binary.Write(&data, binary.LittleEndian, valueToWrite)
				if err != nil {
					return err
				}
			}
		} else {
			for _, val := range frame.elevations {
				err = binary.Write(&data, binary.LittleEndian, val)
				if err != nil {
					return err
				}
			}
		}

		var finalizedData bytes.Buffer
		// compress if needed
		if isCompressed {
			zipWriter, err := gzip.NewWriterLevel(&finalizedData, gzip.BestCompression)
			if err != nil {
				return err
			}
			_, err = data.WriteTo(zipWriter)
			if err != nil {
				return err
			}
			zipWriter.Close()
			if err != nil {
				return err
			}

		} else {
			finalizedData = data
		}

		err = frame.writeHeader(target, uint64(finalizedData.Len()), flags)
		if err != nil {
			return err
		}

		_, err = finalizedData.WriteTo(target)
		if err != nil {
			return err
		}
	}

	return nil
}

func (frame *ElevationFrame) internalRenderElevations() {
	frame.renderedElevations = make([]int16, len(frame.elevations))
	for index, elevation := range frame.elevations {
		var fromSeaLevel = elevation - frame.sealevel
		if fromSeaLevel < float64(math.MinInt16) {
			frame.renderedElevations[index] = math.MinInt16
		} else if fromSeaLevel > float64(math.MaxInt16) {
			frame.renderedElevations[index] = math.MaxInt16
		} else {
			frame.renderedElevations[index] = int16(fromSeaLevel)
		}
	}
}

// reads frame header, must be called before we can read the elevation or rendered elevation data
func (frame *ElevationFrame) internalReadHeader(source io.Reader) error {
	err := binary.Read(source, binary.LittleEndian, &frame.dataReadSize)
	if err != nil {
		return err
	} else {
		//log.Printf("Elevation data size from read: %d", frame.dataReadSize)
	}
	var flags uint64
	err = binary.Read(source, binary.LittleEndian, &flags)
	if err != nil {
		return err
	}
	//log.Printf("Read flags as: %d", flags)
	if flags&IsCompressedFlag > 0 {
		frame.isFromCompressed = true
	}
	if flags&IsRenderedFlag > 0 {
		frame.isFromRendered = true
	}
	return nil
}

// reads header, and stores data unmodified in frame.data
func internalReadElevationFrame(source io.Reader) (ElevationFrame, error) {
	var frame ElevationFrame

	err := frame.internalReadHeader(source)
	if err != nil {
		return frame, err
	}

	frame.data = make([]byte, frame.dataReadSize)

	_, err = io.ReadFull(source, frame.data)
	if err != nil {
		//log.Print("Error reading elevation frame")
		return frame, err
	}

	return frame, nil
}
