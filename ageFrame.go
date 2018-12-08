package worldDataFormat

import (
	//"log"
	"io"
	"encoding/binary"
)

type AgeFrame struct {
	Age float64
}

func (frame *AgeFrame)WriteAll(target io.Writer) error {
	return frame.internalWrite(target)
}

func (frame *AgeFrame)internalWrite(target io.Writer) error {
	var err error
	//log.Printf("Writing age: %e", frame.Age)
	err = binary.Write(target, binary.LittleEndian, frame.Age)
	return err
}

func internalReadAgeFrame(source io.Reader) (AgeFrame, error) {
	var frame AgeFrame

	err := binary.Read(source, binary.LittleEndian, &frame.Age)
	if err != nil {
		return frame, err
	} else {
		//log.Printf("Read age of: %e", frame.Age)
	}

	return frame, nil
}