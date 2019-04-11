package worldDataFormat_test

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"

	. "github.com/Smerom/WorldDataFormat"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ElevationFrame", func() {
	Context("without data it should return an error", func() {
		var newFrame ElevationFrame
		var buf bytes.Buffer

		BeforeEach(func() {
			newFrame = ElevationFrame{}
			buf = bytes.Buffer{}
		})

		Specify("from WriteFull", func() {
			err := newFrame.WriteFull(&buf, false)
			Expect(err).To(Equal(NoData))
		})

		Specify("from WriteRendered", func() {
			err := newFrame.WriteRendered(&buf, false)
			Expect(err).To(Equal(NoData))
		})

	})

	Context("with full elevations", func() {
		var fullFrame ElevationFrame
		var testElev []float64 = []float64{9620 - 3900, 9620 - 300, 9620 + 1234, 9620 + 4586, 9620 + 12300, 9620 + 17000}

		BeforeEach(func() {
			fullFrame = ElevationFrame{}
			fullFrame.SetElevations(testElev)
		})

		It("should return set elevations", func() {
			for index, returnedElev := range fullFrame.Elevations() {
				Expect(returnedElev).To(BeNumerically("==", testElev[index]))
			}
		})

		// TODO: update for new binning method
		PIt("should write the correct compressed data", func() {
			var buf bytes.Buffer
			err := fullFrame.WriteRendered(&buf, true)

			// read in the data size
			var size uint64
			err = binary.Read(&buf, binary.LittleEndian, &size)
			if err != nil {
				Fail("EEEROR")
			}
			var flags uint64
			err = binary.Read(&buf, binary.LittleEndian, &flags)
			if err != nil {
				Fail("EAAROR")
			}

			Expect(buf.Len()).To(BeNumerically("==", size))

			GinkgoWriter.Write([]byte(fmt.Sprintf("Compressed Data length: %d \n", buf.Len())))

			zipReader, err := gzip.NewReader(&buf)
			if err != nil {
				Fail("Zip read error")
			}
			var dataToWrite []byte
			dataToWrite, err = ioutil.ReadAll(zipReader)
			if err != nil {
				Fail("Zip write error")
			}

			GinkgoWriter.Write([]byte("Data written is: \n"))
			GinkgoWriter.Write([]byte(fmt.Sprintf("%08b %08b <> %08b %08b %08b %08b %08b %08b", dataToWrite[0], dataToWrite[1], dataToWrite[2], dataToWrite[3], dataToWrite[4], dataToWrite[5], dataToWrite[6], dataToWrite[7])))
			GinkgoWriter.Write([]byte("\n"))

			Expect(len(dataToWrite)).To(BeNumerically("==", 8))

			var section byte
			section = (dataToWrite[0] & (3 << 0)) >> 0
			Expect(section).To(BeNumerically("==", 0))

			section = (dataToWrite[0] & (3 << 2)) >> 2
			Expect(section).To(BeNumerically("==", 0))

			section = (dataToWrite[0] & (3 << 4)) >> 4
			Expect(section).To(BeNumerically("==", 1))

			section = (dataToWrite[0] & (3 << 6)) >> 6
			Expect(section).To(BeNumerically("==", 2))

			section = (dataToWrite[1] & (3 << 0)) >> 0
			Expect(section).To(BeNumerically("==", 3))

			section = (dataToWrite[1] & (3 << 2)) >> 2
			Expect(section).To(BeNumerically("==", 3))

			/* for reference
				if fromSeaLevel < -3800 {
			    	rendered.section = 0
			    	rendered.value = 0
			    } else if fromSeaLevel < 0 {
			    	rendered.section = 0
			    	rendered.value = byte((fromSeaLevel + 3800)/3800 * 255)
			    } else if fromSeaLevel < 3000 {
			    	rendered.section = 1
			    	rendered.value = byte(fromSeaLevel / 3000 * 255)
			    } else if fromSeaLevel < 7000 {
			    	rendered.section = 2
			    	rendered.value = byte((fromSeaLevel - 3000) / 4000 * 255)
			    } else if fromSeaLevel < 14000 {
			    	rendered.section = 3
			    	rendered.value = byte((fromSeaLevel - 7000) / 7000 * 255)
			    } else {
			    	rendered.section = 3
			    	rendered.value = 255
			    }*/
			Expect(dataToWrite[2]).To(BeNumerically("==", 0))
			Expect(dataToWrite[3]).To(BeNumerically("==", byte(((testElev[1]-9620)+3800)/3800*255)))
			Expect(dataToWrite[4]).To(BeNumerically("==", byte((testElev[2]-9620)/3000*255)))
			Expect(dataToWrite[5]).To(BeNumerically("==", byte(((testElev[3]-9620)-3000)/4000*255)))
			Expect(dataToWrite[6]).To(BeNumerically("==", byte(((testElev[4]-9620)-7000)/7000*255)))
			Expect(dataToWrite[7]).To(BeNumerically("==", 255))

		})

		It("should return the same data the second time", func() {
			var buf bytes.Buffer
			err := fullFrame.WriteRendered(&buf, true)
			if err != nil {
				Fail("Initial Write fail")
			}

			var initialWrite bytes.Buffer
			var tee io.Reader
			tee = io.TeeReader(&buf, &initialWrite)

			frame, err := ReadElevationFrame(tee)
			if err != nil {
				Fail("Read Fail")
			}

			var secondWrite bytes.Buffer
			err = frame.WriteRendered(&secondWrite, true)
			if err != nil {
				Fail("Second Write Fail")
			}

			var areSame int
			areSame = bytes.Compare(initialWrite.Bytes(), secondWrite.Bytes())
			if areSame != 0 {
				Fail("Written Bytes not the same")
			}

		})
	})

})
