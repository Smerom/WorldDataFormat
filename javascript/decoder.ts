declare var pako: any;

class WorldSimulationDecoder {
	version: number;
	subdivisionCount: number;
	frameSetCount: number;
	typesBitField: Uint32Array;

	vertexCount: number;
	vcountSet: Boolean;

	readBytes: number;

	constructor(initialData: DataView) {
		this.typesBitField = new Uint32Array(2);

		// assume the most significant half of each is zero
		this.version = initialData.getUint32(0, true);
		this.subdivisionCount = initialData.getUint32(16, true);
		this.frameSetCount = initialData.getUint32(24, true);

		this.typesBitField[0] = initialData.getUint32(32, true);
		this.typesBitField[1] = initialData.getUint32(36, true);

		this.vcountSet = false;
		this.vertexCount = 0;

		this.readBytes = 40; // currently 40 byte header
	}

	setVertexCount(vcount: number) {
		this.vertexCount = vcount;
		this.vcountSet = true;
	}

	decodeSet(data: DataView): FrameSetDecoder {
		let set: FrameSetDecoder;
		if(!this.vcountSet) {
			throw "Need some vert count!";
			
		}

		set = new FrameSetDecoder(data, this.typesBitField, this.vertexCount);

		return set;
	}
}



class FrameSetDecoder {
	totalSize: number;
	version: number;
	frameCount: number;
	readFrames: number;
	typeOffsets: Uint32Array;
	frameData: DataView;
	typesBitField: Uint32Array;

	vertexCount: number;

	previousFrame: Frame;

	constructor(data: DataView, typeBitField: Uint32Array, vcount: number) {
		this.typeOffsets = new Uint32Array(numberOfTypesSet(typeBitField));
		this.typesBitField = typeBitField;

		// read in type offsets
		let typeFieldCurrentCount: number = 0;
		if(isTypeFlagSet(TypeFlags.AgeFrameFlag, this.typesBitField)) {
			this.typeOffsets[typeFieldCurrentCount] = data.getUint32(32 + 8*typeFieldCurrentCount, true)
			typeFieldCurrentCount++;
		}
		
		// grab elevation if applicable
		if(isTypeFlagSet(TypeFlags.ElevationFrameFlag, this.typesBitField)) {
			this.typeOffsets[typeFieldCurrentCount] = data.getUint32(32 + 8*typeFieldCurrentCount, true)
			typeFieldCurrentCount++;
		}

		// grab satallite colors if written
		if (isTypeFlagSet(TypeFlags.SatalliteFrameFlag, this.typesBitField)) {
			this.typeOffsets[typeFieldCurrentCount] = data.getUint32(32 + 8*typeFieldCurrentCount, true)
			typeFieldCurrentCount++;
		}

		// assumes 
		this.totalSize = data.getUint32(0, true);
		this.version = data.getUint32(8, true);
		this.frameCount = data.getUint32(24, true);

		this.vertexCount = vcount;
		this.previousFrame = null;
		this.readFrames = 0;

		this.frameData = new DataView(data.buffer.slice(4*8 + 8*this.typeOffsets.length));
	}

	nextFrame(): Frame {
		let next: Frame = new Frame();

		this.readFrames++;

		if(this.readFrames <= this.frameCount) {
			let index = 0
			// grab age if applicable
			if(isTypeFlagSet(TypeFlags.AgeFrameFlag, this.typesBitField)) {
				next.age = new AgeFrame(new DataView(this.frameData.buffer.slice(this.typeOffsets[index])));
				this.typeOffsets[index] += next.age.readBytes;
				index++;
			}
			
			// grab elevation if applicable
			if(isTypeFlagSet(TypeFlags.ElevationFrameFlag, this.typesBitField)) {
				let prevElevation: ElevationFrame
				if(this.previousFrame != null) {
					prevElevation = this.previousFrame.elevations;
				} else {
					prevElevation = null;
				}
				next.elevations = new ElevationFrame(new DataView(this.frameData.buffer.slice(this.typeOffsets[index])), prevElevation, this.vertexCount);
				this.typeOffsets[index] += next.elevations.readBytes;
				index++;
			}

			// grab satallite colors if available
			if(isTypeFlagSet(TypeFlags.SatalliteFrameFlag, this.typesBitField)) {
				let prevSatallite: SatalliteFrame
				if(this.previousFrame != null) {
					prevSatallite = this.previousFrame.satallite;
				} else {
					prevSatallite = null;
				}
				next.satallite = new SatalliteFrame(new DataView(this.frameData.buffer.slice(this.typeOffsets[index])), prevSatallite, this.vertexCount);
				this.typeOffsets[index] += next.satallite.readBytes;
				index++;
			}

			this.previousFrame = next; // next will be the next frames prev
			return next;
		} else {
			return null;
		}
	}
}

class Frame {
	age: AgeFrame;
	elevations: ElevationFrame;
	satallite: SatalliteFrame;
}

class AgeFrame {
	age: number;
	readBytes: number;

	constructor(data: DataView) {
		this.age = data.getFloat64(0, true);
		this.readBytes = 8;
	}
}

// rendered only
class ElevationFrame {
	elevations: Int16Array;
	readBytes: number;

	constructor(data: DataView, prevElevations: ElevationFrame, vertexCount: number) {
		let dataSize: number;
		dataSize = data.getUint32(0, true)
		
		let storageFlags: Uint32Array;
		storageFlags = new Uint32Array(2);
		storageFlags[0] = data.getUint32(8, true)
		storageFlags[1] = data.getUint32(12, true)

		// decompress if necessary
		let buff: Uint8Array;
		if(isTypeFlagSet(TypeFlags.IsCompressedFlag, storageFlags)) {
			try {
				// inflate and convert to int 16 array
				buff = pako.inflate(new Uint8Array(data.buffer, 16, dataSize));
				this.elevations = new Int16Array(buff.buffer)
			} catch (err) {
				console.log(err);
			}
		} else {
			this.elevations = new Int16Array(data.buffer, 16, dataSize);
		}
		// TODO: calculate elevations from previous frame if set

		// set data read from data buffer
		this.readBytes = dataSize + 16;
	}
}

class SatalliteFrame {
	colors: Uint8Array;
	readBytes: number;

	constructor(data: DataView, prevSatallite: SatalliteFrame, vertexCount: number) {
		this.colors = new Uint8Array(3*vertexCount);

		let dataSize: number;
		dataSize = data.getUint32(0, true)
		
		let storageFlags: Uint32Array;
		storageFlags = new Uint32Array(2);
		storageFlags[0] = data.getUint32(8, true)
		storageFlags[1] = data.getUint32(12, true)

		// decompress if necessary
		let buff: Uint8Array;
		if(isTypeFlagSet(TypeFlags.IsCompressedFlag, storageFlags)) {
			try {
				buff = pako.inflate(new Uint8Array(data.buffer, 16, dataSize));
			} catch (err) {
				console.log(err);
			}
		} else {
			buff = new Uint8Array(data.buffer, 16, dataSize);
		}
		// set data read from data buffer
		this.readBytes = dataSize + 16;

		// read the segments
		for (var i = 0; i < vertexCount; ++i) {
			this.colors[i*3 + 0] = buff[i];
			this.colors[i*3 + 1] = buff[i + vertexCount];
			this.colors[i*3 + 2] = buff[i + 2*vertexCount];
		}
	}
}


enum TypeFlags {
	AgeFrameFlag = 0,
	ElevationFrameFlag = 1,
	SatalliteFrameFlag = 2,

	IsAverageDiffedFlag = 60,
	IsSelfDiffedFlag = 61,
	IsRenderedFlag = 62,
	IsCompressedFlag = 63,
}


function isTypeFlagSet(flag: TypeFlags, typeField: Uint32Array): Boolean {
	if(flag < 32) {
		return (typeField[0] & 1 << flag) != 0;
	} else if (flag < 64) {
		return (typeField[1] & 1 << (flag - 32)) != 0;
	}
	return false
}

function numberOfTypesSet(typeField: Uint32Array): number {
	let count = 0
	// check only written types
	if(isTypeFlagSet(TypeFlags.AgeFrameFlag, typeField)) {
		count ++;
	}
	if(isTypeFlagSet(TypeFlags.ElevationFrameFlag, typeField)) {
		count ++;
	}
	if(isTypeFlagSet(TypeFlags.SatalliteFrameFlag, typeField)) {
		count ++;
	}

	return count
}