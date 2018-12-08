package worldDataFormat_test

import (
	"fmt"
	"bytes"
	"encoding/binary"
	. "github.com/Smerom/WorldDataFormat"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("FrameSet", func() {
	It("should return added frames", func() {
	    var set FrameSet
		var frame Frame

		set.AddFrame(frame)

		// TODO need some way to test if its the same frame set
		Expect(len(set.Frames())).To(Equal(1))
	})

	Context("header", func() {
	    Context("for trivial frames", func() {
	    	var set FrameSet
	    	var frameTypes uint64

	    	BeforeEach(func() {
	    		var testSubFrame AgeFrame
	    		testSubFrame.Age = 10
	    		var testFrame Frame
	    		testFrame.Age = &testSubFrame
	    		set = FrameSet{}
	    	    set.AddFrame(testFrame)
	    	    frameTypes = AgeFrameFlag // so we test trivial frames, not trivial writes
	    	})

	    	It("should be valid when written in Full mode", func() {
	    	    var data bytes.Buffer
				err := set.WriteFull(&data, false, frameTypes)
				if err != nil {
					Fail(fmt.Sprintf("World header error: %s", err))
				}
				// min length
				Expect(data.Len()).ToNot(BeNumerically("<", 24))
				var dataLen uint64 = uint64(data.Len())

				// check total size
				var totalSize uint64
				err = binary.Read(&data, binary.LittleEndian, &totalSize)
				if err != nil {
					Fail("Test read error")
				}
				Expect(totalSize).To(BeNumerically("==", dataLen))

				// check version
				var version uint64
				err = binary.Read(&data, binary.LittleEndian, &version)
				if err != nil {
					Fail("Test read error")
				}
				Expect(version).To(BeNumerically("==", FrameSetVersion))

				// check Header Length (with only one offset)
				var headerLen uint64
				err = binary.Read(&data, binary.LittleEndian, &headerLen)
				if err != nil {
					Fail("Test read error")
				}
				Expect(headerLen).To(BeNumerically("==", 16))

				// check frame count
				var frameCount uint64
				err = binary.Read(&data, binary.LittleEndian, &frameCount)
				if err != nil {
					Fail("Test read error")
				}
				Expect(frameCount).To(BeNumerically("==", 1))

				// check offset
				var offset uint64
				err = binary.Read(&data, binary.LittleEndian, &offset)
				if err != nil {
					Fail("Test read error")
				}
				Expect(offset).To(BeNumerically("==", 0))
	    	})

	    	It("should be valid when written in Rendered mode", func() {
	    	    var data bytes.Buffer
				err := set.WriteRendered(&data, false, frameTypes)
				if err != nil {
					Fail(fmt.Sprintf("World header error: %s", err))
				}
				// min length
				Expect(data.Len()).ToNot(BeNumerically("<", 24))
				var dataLen uint64 = uint64(data.Len())

				// check total size
				var totalSize uint64
				err = binary.Read(&data, binary.LittleEndian, &totalSize)
				if err != nil {
					Fail("Test read error")
				}
				Expect(totalSize).To(BeNumerically("==", dataLen))

				// check version
				var version uint64
				err = binary.Read(&data, binary.LittleEndian, &version)
				if err != nil {
					Fail("Test read error")
				}
				Expect(version).To(BeNumerically("==", FrameSetVersion))

				// check Header Length (with only one offset)
				var headerLen uint64
				err = binary.Read(&data, binary.LittleEndian, &headerLen)
				if err != nil {
					Fail("Test read error")
				}
				Expect(headerLen).To(BeNumerically("==", 16))

				// check frame count
				var frameCount uint64
				err = binary.Read(&data, binary.LittleEndian, &frameCount)
				if err != nil {
					Fail("Test read error")
				}
				Expect(frameCount).To(BeNumerically("==", 1))

				// check offset
				var offset uint64
				err = binary.Read(&data, binary.LittleEndian, &offset)
				if err != nil {
					Fail("Test read error")
				}
				Expect(offset).To(BeNumerically("==", 0))
	    	})
	        
	    })
	})
})
