package worldDataFormat

import (
	"io"
	"log"
	"io/ioutil"
	"encoding/binary"
	"bytes"
	"compress/gzip"
	"image"
	_ "image/png"
)

type RenderedColor struct {
	Red byte
	Green byte
	Blue byte
}

type SatalliteFrame struct {
	colors []RenderedColor

	data []byte

	dataReadSize uint64
	isFromCompressed bool
}


func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	} else if value > max {
		return max
	}
	return value
}

func (frame *SatalliteFrame)SetColorsFromData(tempurature []float64, precipitation []float64, elevations []float64, colorGuide image.Image) {
	// hard coded ranges
	var minTemp, maxTemp float64
	minTemp = -10
	maxTemp = 30
	var minPrecip, maxPrecip float64
	minPrecip = 0
	maxPrecip = 4.16

	frame.colors = make([]RenderedColor, len(tempurature))
	for i := 0; i < len(tempurature); i++ {
		if elevations[i] > 9620 {
			var xTemp, yPrecip int
			var ourTemp float64 = tempurature[i]
			var ourPrecip float64 = precipitation[i]
			xTemp = int(float64(colorGuide.Bounds().Max.X - colorGuide.Bounds().Min.X) * clamp( (ourTemp - minTemp) / (maxTemp - minTemp), 0, 1) )
			yPrecip = int(float64(colorGuide.Bounds().Max.Y - colorGuide.Bounds().Min.Y) * clamp( (ourPrecip - minPrecip) / (maxPrecip - minPrecip), 0, 1) )

			if xTemp == colorGuide.Bounds().Max.X - colorGuide.Bounds().Min.X {
				xTemp -= 1
			}
			if yPrecip == colorGuide.Bounds().Max.Y - colorGuide.Bounds().Min.Y{
				yPrecip -= 1
			}

			theColor := colorGuide.At(colorGuide.Bounds().Min.X + xTemp, colorGuide.Bounds().Min.Y + yPrecip)
			r, g, b, _ := theColor.RGBA()
			// scale colors so they are bytes instead of uint32 in range of [0, 0xffff]
			frame.colors[i].Red = byte(r / 256)
			frame.colors[i].Green = byte(g / 256)
			frame.colors[i].Blue = byte(b / 256)

			if frame.colors[i].Red == 0 && frame.colors[i].Green == 0 && frame.colors[i].Blue == 0 {
				log.Printf("colors: %d, %d, %d", r, g, b)
				log.Printf("Temp: %e", ourTemp)
				log.Printf("Precip: %e", ourPrecip)
			}
		} else if tempurature[i] < -6 {
			frame.colors[i].Red = 255
			frame.colors[i].Green = 255
			frame.colors[i].Blue = 255
		} else {
			frame.colors[i].Red = 0
			frame.colors[i].Green = 0
			frame.colors[i].Blue = 255
		}
	}
}

func (frame *SatalliteFrame)Colors() []RenderedColor {
	return frame.colors
}

func (frame *SatalliteFrame)WriteFull(target io.Writer, isCompressed bool) error {
	return RenderedOnlyFrame
}

func (frame *SatalliteFrame)WriteRendered(target io.Writer, isCompressed bool) error {
	return frame.internalWrite(target, isCompressed, nil)
}

func ReadSatalliteFrame(source io.Reader) (SatalliteFrame, error) {
	return internalReadSatalliteFrame(source)
}


func (frame *SatalliteFrame)writeHeader(target io.Writer, dataSize uint64, flags uint64) error {
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

func (frame *SatalliteFrame)internalWrite(target io.Writer, isCompressed bool, prevFrame *SatalliteFrame) error {
var err error
	if len(frame.colors) == 0 && frame.data == nil{
		return NoData
	}
	var flags uint64
	if isCompressed {
		flags = flags | IsCompressedFlag
	}

	// check if we have valid stored data
	
	//log.Print("writing from rendered")

	// compress if needed
	var dataToWrite []byte
	// check if previously written data exists
	if len(frame.data) != 0 {
		if isCompressed && !frame.isFromCompressed {
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
		// we need to interlace the colors
		var red []byte = make([]byte, len(frame.colors))
		var green []byte = make([]byte, len(frame.colors))
		var blue []byte = make([]byte, len(frame.colors))

		for index, color := range frame.colors {
			red[index] = color.Red
			green[index] = color.Green
			blue[index] = color.Blue
		}

		var colorBuffer bytes.Buffer
		_, err = colorBuffer.Write(red)
		if err != nil {
			return err
		}
		_, err = colorBuffer.Write(green)
		if err != nil {
			return err
		}
		_, err = colorBuffer.Write(blue)
		if err != nil {
			return err
		}

		if isCompressed {
			var finalizedBuffer bytes.Buffer
			zipWriter, err := gzip.NewWriterLevel(&finalizedBuffer, gzip.BestCompression)
			if err != nil {
				return err
			}
			_, err = colorBuffer.WriteTo(zipWriter)
			if err != nil {
				return err
			}
			zipWriter.Close()
			if err != nil {
				return err
			}
			dataToWrite = finalizedBuffer.Bytes()
		} else {
			dataToWrite = colorBuffer.Bytes()
		}

		err = frame.writeHeader(target, uint64(len(dataToWrite)), flags)
		if err != nil {
			return err
		}

		_, err = target.Write(dataToWrite)
		if err != nil {
			return err
		}
	}
	return nil
}

func (frame *SatalliteFrame)internalReadHeader(source io.Reader) error {
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
	if flags & IsCompressedFlag > 0 {
		frame.isFromCompressed = true
	}
	return nil
}

func internalReadSatalliteFrame(source io.Reader) (SatalliteFrame, error) {
	var frame SatalliteFrame

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