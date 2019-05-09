package worldDataFormat_test

import (
	"bytes"
	"encoding/binary"
	"math/rand"

	. "github.com/Smerom/WorldDataFormat"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func randomAgeFrame(rnd *rand.Rand) AgeFrame {
	return AgeFrame{
		Age: rnd.Float64(),
	}
}

var _ = Describe("AgeFrame", func() {
	It("should write the correct 64bit float for age", func() {
		// try a bunch
		for i := 0; i < 10000; i++ {
			frame := randomAgeFrame(rand.New(rand.NewSource(12345)))
			var setAge = frame.Age

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
		}

	})

})
