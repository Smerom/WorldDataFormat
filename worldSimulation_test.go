package worldDataFormat_test

import (
	"fmt"
	"bytes"
	"encoding/binary"
	. "github.com/Smerom/WorldDataFormat"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("WorldSimulation", func() {
	It("should return added frameSets", func() {
		var worldSim WorldSimulation
		var set FrameSet

		worldSim.AddFrameSet(set)

		// TODO need some way to test if its the same frame set
		Expect(len(worldSim.FrameSets())).To(BeNumerically("==", 1))
	    
	})
	It("should return set subdivisions", func() {
	    var worldSim WorldSimulation

	    worldSim.SetSubdivisions(10)

	    Expect(worldSim.Subdivisions()).To(BeNumerically("==", 10))
	})
	It("should return an error on write if subdivisions not set", func() {
		var worldSim WorldSimulation
	    var set FrameSet
		worldSim.AddFrameSet(set)
		var typesToWrite uint64 = AgeFrameFlag // so we write something

		var data bytes.Buffer
		err := worldSim.WriteFull(&data, false, typesToWrite)
		Expect(err).To(Equal(MissingGridDefinition))
	})

	Context("header", func() {
		Context("for trivial frame set", func() {

			var worldSim WorldSimulation
			var typesToWrite uint64
			const subdivisions = 123

			BeforeEach(func() {
				// reset
				worldSim = WorldSimulation{}

			    var set FrameSet
			    var frame Frame
			    var ageFrame AgeFrame
			    ageFrame.Age = 100
			    frame.Age = &ageFrame

			    set.AddFrame(frame)

				worldSim.AddFrameSet(set)
				worldSim.SetSubdivisions(subdivisions)
				typesToWrite = AgeFrameFlag // so we write something
			})

			It("should be valid when written in Full mode", func() {
			    var data bytes.Buffer
				err := worldSim.WriteFull(&data, false, typesToWrite)
				if err != nil {
					Fail(fmt.Sprintf("World header error: %s", err))
				}
				// min length
				Expect(data.Len()).ToNot(BeNumerically("<", 16))/* need at least version and header length */

				// check version
				var version uint64
				err = binary.Read(&data, binary.LittleEndian, &version)
				if err != nil {
					Fail("Test read error")
				}
				Expect(version).To(BeNumerically("==", WorldSimulationVersion))

				// check header length
				var length uint64
				err = binary.Read(&data, binary.LittleEndian, &length)
				if err != nil {
					Fail("Test read error")
				}
				Expect(length).To(BeNumerically("==", 24))

				// check subdivisions
				var subdivCount uint64
				err = binary.Read(&data, binary.LittleEndian, &subdivCount)
				if err != nil {
					Fail("Test read error")
				}
				Expect(subdivCount).To(BeNumerically("==", subdivisions))

				// check frame count
				var frameCount uint64
				err = binary.Read(&data, binary.LittleEndian, &frameCount)
				if err != nil {
					Fail("Test read error")
				}
				Expect(frameCount).To(BeNumerically("==", 1))

				// check type field
				var typesWritten uint64
				err = binary.Read(&data, binary.LittleEndian, &typesWritten)
				if err != nil {
					Fail("Test read error")
				}
				Expect(typesWritten).To(BeNumerically("==", typesToWrite))
			})

			It("should be valid when written in Rendered mode", func() {
			    var data bytes.Buffer
				err := worldSim.WriteRendered(&data, false, typesToWrite)
				if err != nil {
					Fail(fmt.Sprintf("World header error: %s", err))
				}
				// min length
				Expect(data.Len()).ToNot(BeNumerically("<", 16))/* need at least version and header length */
				
				// check version
				var version uint64
				err = binary.Read(&data, binary.LittleEndian, &version)
				if err != nil {
					Fail("Test read error")
				}
				Expect(version).To(BeNumerically("==", WorldSimulationVersion))

				// check header length
				var length uint64
				err = binary.Read(&data, binary.LittleEndian, &length)
				if err != nil {
					Fail("Test read error")
				}
				Expect(length).To(BeNumerically("==", 24))

				// check subdivisions
				var subdivCount uint64
				err = binary.Read(&data, binary.LittleEndian, &subdivCount)
				if err != nil {
					Fail("Test read error")
				}
				Expect(subdivCount).To(BeNumerically("==", subdivisions))

				// check frame count
				var frameCount uint64
				err = binary.Read(&data, binary.LittleEndian, &frameCount)
				if err != nil {
					Fail("Test read error")
				}
				Expect(frameCount).To(BeNumerically("==", 1))

				// check type field
				var typesWritten uint64
				err = binary.Read(&data, binary.LittleEndian, &typesWritten)
				if err != nil {
					Fail("Test read error")
				}
				Expect(typesWritten).To(BeNumerically("==", typesToWrite))
			})

		})
	})

	Context("body", func() {
	    Context("for trivial frame set", func() {

	        var worldSim WorldSimulation
	        var typesToWrite uint64
	        const subdivisions = 123

			BeforeEach(func() {
				worldSim = WorldSimulation{}

			    var set FrameSet
			    var frame Frame
			    var ageFrame AgeFrame
			    ageFrame.Age = 100
			    frame.Age = &ageFrame

			    set.AddFrame(frame)

				worldSim.AddFrameSet(set)
				worldSim.SetSubdivisions(subdivisions)
				typesToWrite = AgeFrameFlag // so we write something
			})

			It("should exist after header in Full mode", func() {
			    var data bytes.Buffer
				err := worldSim.WriteFull(&data, false, typesToWrite)
				if err != nil {
					Fail(fmt.Sprintf("World WriteFull error: %s", err))
				}
				Expect(data.Len()).To(BeNumerically(">", 40))
			})

			It("should exist after header in Rendered mode", func() {
			    var data bytes.Buffer
				err := worldSim.WriteRendered(&data, false, typesToWrite)
				if err != nil {
					Fail(fmt.Sprintf("World WriteRendered error: %s", err))
				}
				Expect(data.Len()).To(BeNumerically(">", 40))
			})
	    })
	})

})
