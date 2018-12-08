package worldDataFormat

import (
	"errors"
)

var NoData = errors.New("No Data")
var InvalidData = errors.New("Invalid Data")
var MissingData = errors.New("Missing Data")
var MissingGridDefinition = errors.New("Grid Definition was not set")

var RenderedOnlyFrame = errors.New("Frame type must be rendered.")

var IncompatibleVersion = errors.New("Incompatible Version")