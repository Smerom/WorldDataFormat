package worldDataFormat_test

import (
	"bytes"
	"encoding/binary"

	. "github.com/Smerom/WorldDataFormat"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("AgeFrame", func() {
	It("should write the correct 64bit float for age", func() {
	    var frame AgeFrame
	    const setAge = 171.002301
	    frame.Age = setAge

	    var data bytes.Buffer

	    frame.WriteAll(&data)

	    // check length
	    Expect(data.Len()).To(BeNumerically("==", 8))

	    // check value
	    var writtenAge float64
	    err := binary.Read(&data, binary.LittleEndian, &writtenAge)
	    if err != nil {
	    	Fail("Test read error")
	    }
	    Expect(writtenAge).To(BeNumerically("==", setAge))

	})

})
