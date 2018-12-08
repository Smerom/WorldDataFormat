var WorldSimulationDecoder = (function () {
    function WorldSimulationDecoder(initialData) {
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
    WorldSimulationDecoder.prototype.setVertexCount = function (vcount) {
        this.vertexCount = vcount;
        this.vcountSet = true;
    };
    WorldSimulationDecoder.prototype.decodeSet = function (data) {
        var set;
        if (!this.vcountSet) {
            throw "Need some vert count!";
        }
        set = new FrameSetDecoder(data, this.typesBitField, this.vertexCount);
        return set;
    };
    return WorldSimulationDecoder;
}());
var FrameSetDecoder = (function () {
    function FrameSetDecoder(data, typeBitField, vcount) {
        this.typeOffsets = new Uint32Array(numberOfTypesSet(typeBitField));
        this.typesBitField = typeBitField;
        // read in type offsets
        var typeFieldCurrentCount = 0;
        if (isTypeFlagSet(TypeFlags.AgeFrameFlag, this.typesBitField)) {
            this.typeOffsets[typeFieldCurrentCount] = data.getUint32(32 + 8 * typeFieldCurrentCount, true);
            typeFieldCurrentCount++;
        }
        // grab elevation if applicable
        if (isTypeFlagSet(TypeFlags.ElevationFrameFlag, this.typesBitField)) {
            this.typeOffsets[typeFieldCurrentCount] = data.getUint32(32 + 8 * typeFieldCurrentCount, true);
            typeFieldCurrentCount++;
        }
        // grab satallite colors if written
        if (isTypeFlagSet(TypeFlags.SatalliteFrameFlag, this.typesBitField)) {
            this.typeOffsets[typeFieldCurrentCount] = data.getUint32(32 + 8 * typeFieldCurrentCount, true);
            typeFieldCurrentCount++;
        }
        // assumes 
        this.totalSize = data.getUint32(0, true);
        this.version = data.getUint32(8, true);
        this.frameCount = data.getUint32(24, true);
        this.vertexCount = vcount;
        this.previousFrame = null;
        this.readFrames = 0;
        this.frameData = new DataView(data.buffer.slice(4 * 8 + 8 * this.typeOffsets.length));
    }
    FrameSetDecoder.prototype.nextFrame = function () {
        var next = new Frame();
        this.readFrames++;
        if (this.readFrames <= this.frameCount) {
            var index = 0;
            // grab age if applicable
            if (isTypeFlagSet(TypeFlags.AgeFrameFlag, this.typesBitField)) {
                next.age = new AgeFrame(new DataView(this.frameData.buffer.slice(this.typeOffsets[index])));
                this.typeOffsets[index] += next.age.readBytes;
                index++;
            }
            // grab elevation if applicable
            if (isTypeFlagSet(TypeFlags.ElevationFrameFlag, this.typesBitField)) {
                var prevElevation = void 0;
                if (this.previousFrame != null) {
                    prevElevation = this.previousFrame.elevations;
                }
                else {
                    prevElevation = null;
                }
                next.elevations = new ElevationFrame(new DataView(this.frameData.buffer.slice(this.typeOffsets[index])), prevElevation, this.vertexCount);
                this.typeOffsets[index] += next.elevations.readBytes;
                index++;
            }
            // grab satallite colors if available
            if (isTypeFlagSet(TypeFlags.SatalliteFrameFlag, this.typesBitField)) {
                var prevSatallite = void 0;
                if (this.previousFrame != null) {
                    prevSatallite = this.previousFrame.satallite;
                }
                else {
                    prevSatallite = null;
                }
                next.satallite = new SatalliteFrame(new DataView(this.frameData.buffer.slice(this.typeOffsets[index])), prevSatallite, this.vertexCount);
                this.typeOffsets[index] += next.satallite.readBytes;
                index++;
            }
            this.previousFrame = next; // next will be the next frames prev
            return next;
        }
        else {
            return null;
        }
    };
    return FrameSetDecoder;
}());
var Frame = (function () {
    function Frame() {
    }
    return Frame;
}());
var AgeFrame = (function () {
    function AgeFrame(data) {
        this.age = data.getFloat64(0, true);
        this.readBytes = 8;
    }
    return AgeFrame;
}());
// rendered only
var ElevationFrame = (function () {
    function ElevationFrame(data, prevElevations, vertexCount) {
        this.elevations = new Uint8Array(2 * vertexCount);
        var dataSize;
        dataSize = data.getUint32(0, true);
        var storageFlags;
        storageFlags = new Uint32Array(2);
        storageFlags[0] = data.getUint32(8, true);
        storageFlags[1] = data.getUint32(12, true);
        // decompress if necessary
        var buff;
        if (isTypeFlagSet(TypeFlags.IsCompressedFlag, storageFlags)) {
            try {
                buff = pako.inflate(new Uint8Array(data.buffer, 16, dataSize));
            }
            catch (err) {
                console.log(err);
            }
        }
        else {
            buff = new Uint8Array(data.buffer, 16, dataSize);
        }
        // set data read from data buffer
        this.readBytes = dataSize + 16;
        // read the segments
        var byteOffset = 0;
        var bitOffset = 0;
        for (var i = 0; i < vertexCount; ++i) {
            if (bitOffset > 6) {
                bitOffset = 0;
                byteOffset++;
            }
            this.elevations[i * 2] = (buff[byteOffset] & (3 << bitOffset)) >>> bitOffset;
            bitOffset += 2;
        }
        byteOffset++; // last byte wont be included yet
        // read the values
        for (var i = 0; i < vertexCount; ++i) {
            if (prevElevations != null) {
                this.elevations[i * 2 + 1] = buff[byteOffset + i] + prevElevations.elevations[i * 2 + 1];
            }
            else {
                this.elevations[i * 2 + 1] = buff[byteOffset + i];
            }
        }
    }
    return ElevationFrame;
}());
var SatalliteFrame = (function () {
    function SatalliteFrame(data, prevSatallite, vertexCount) {
        this.colors = new Uint8Array(3 * vertexCount);
        var dataSize;
        dataSize = data.getUint32(0, true);
        var storageFlags;
        storageFlags = new Uint32Array(2);
        storageFlags[0] = data.getUint32(8, true);
        storageFlags[1] = data.getUint32(12, true);
        // decompress if necessary
        var buff;
        if (isTypeFlagSet(TypeFlags.IsCompressedFlag, storageFlags)) {
            try {
                buff = pako.inflate(new Uint8Array(data.buffer, 16, dataSize));
            }
            catch (err) {
                console.log(err);
            }
        }
        else {
            buff = new Uint8Array(data.buffer, 16, dataSize);
        }
        // set data read from data buffer
        this.readBytes = dataSize + 16;
        // read the segments
        for (var i = 0; i < vertexCount; ++i) {
            this.colors[i * 3 + 0] = buff[i];
            this.colors[i * 3 + 1] = buff[i + vertexCount];
            this.colors[i * 3 + 2] = buff[i + 2 * vertexCount];
        }
    }
    return SatalliteFrame;
}());
var TypeFlags;
(function (TypeFlags) {
    TypeFlags[TypeFlags["AgeFrameFlag"] = 0] = "AgeFrameFlag";
    TypeFlags[TypeFlags["ElevationFrameFlag"] = 1] = "ElevationFrameFlag";
    TypeFlags[TypeFlags["SatalliteFrameFlag"] = 2] = "SatalliteFrameFlag";
    TypeFlags[TypeFlags["IsAverageDiffedFlag"] = 60] = "IsAverageDiffedFlag";
    TypeFlags[TypeFlags["IsSelfDiffedFlag"] = 61] = "IsSelfDiffedFlag";
    TypeFlags[TypeFlags["IsRenderedFlag"] = 62] = "IsRenderedFlag";
    TypeFlags[TypeFlags["IsCompressedFlag"] = 63] = "IsCompressedFlag";
})(TypeFlags || (TypeFlags = {}));
function isTypeFlagSet(flag, typeField) {
    if (flag < 32) {
        return (typeField[0] & 1 << flag) != 0;
    }
    else if (flag < 64) {
        return (typeField[1] & 1 << (flag - 32)) != 0;
    }
    return false;
}
function numberOfTypesSet(typeField) {
    var count = 0;
    // check only written types
    if (isTypeFlagSet(TypeFlags.AgeFrameFlag, typeField)) {
        count++;
    }
    if (isTypeFlagSet(TypeFlags.ElevationFrameFlag, typeField)) {
        count++;
    }
    if (isTypeFlagSet(TypeFlags.SatalliteFrameFlag, typeField)) {
        count++;
    }
    return count;
}
