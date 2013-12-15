package translator

var Prelude = `
Error.stackTraceLimit = -1;

var Go$obj, Go$tuple;
var Go$idCounter = 1;
var Go$keys = function(m) { return m ? Object.keys(m) : []; };
var Go$min = Math.min;
var Go$throwRuntimeError, Go$reflect, Go$newStringPointer;

var Go$cache = function(v) {
	return function() {
		if (v.constructor === Function) {
			v = v();
		}
		return v;
	};
};

var newWrappedType = function(name, size) {
	var typ = function(v) { this.Go$val = v; };
	typ.Go$string = name;
	typ.Go$type = Go$cache(function() {
		return new Go$reflect.rtype(size, 0, 0, 0, 0, Go$reflect.kinds[name], typ, null, Go$newStringPointer(name), null, null);
	});
	typ.prototype.Go$key = function() { return name + "$" + this.Go$val; };
	return typ;
};

var Go$Uint8   = newWrappedType("uint8", 1);
var Go$Uint16  = newWrappedType("uint16", 2);
var Go$Uint32  = newWrappedType("uint32", 4);
var Go$Int8    = newWrappedType("int8", 1);
var Go$Int16   = newWrappedType("int16", 2);
var Go$Int32   = newWrappedType("int32", 4);
var Go$Float32 = newWrappedType("float32", 4);
var Go$Float64 = newWrappedType("float64", 8);
var Go$Uint    = newWrappedType("uint", 4);
var Go$Int     = newWrappedType("int", 4);
var Go$Uintptr = newWrappedType("uintptr", 4);
var Go$Byte    = Go$Uint8;
var Go$Rune    = Go$Int32;

var Go$Bool    = newWrappedType("bool", 0);
var Go$String  = newWrappedType("string", 0);

var Go$Func    = newWrappedType("func", 0);
Go$Func.prototype.Go$uncomparable = true;

var Go$UnsafePointer = newWrappedType("unsafe.Pointer", 4);

var Go$Array           = Array;
var Go$Uint8Array      = Uint8Array;
var Go$Uint16Array     = Uint16Array;
var Go$Uint32Array     = Uint32Array;
var Go$Uint64Array     = Array;
var Go$Int8Array       = Int8Array;
var Go$Int16Array      = Int16Array;
var Go$Int32Array      = Int32Array;
var Go$Int64Array      = Array;
var Go$Float32Array    = Float32Array;
var Go$Float64Array    = Float64Array;
var Go$Complex64Array  = Array;
var Go$Complex128Array = Array;
var Go$UintArray       = Uint32Array;
var Go$IntArray        = Int32Array;
var Go$UintptrArray    = Uint32Array;
var Go$ByteArray       = Go$Uint8Array;
var Go$RuneArray       = Go$Int32Array;

var Go$arrayTypes = {};
var Go$arrayType = function(elem, len) {
	var typeString = "[" + len + "]" + elem.Go$string;
	var typ = Go$arrayTypes[typeString];
	if (typ === undefined) {
		typ = function(v) { this.Go$val = v; };
		typ.Go$string = typeString;
		typ.Go$type = Go$cache(function() {
			var rt = new Go$reflect.rtype(0, 0, 0, 0, 0, Go$reflect.kinds.array, null, null, Go$newStringPointer(typeString), null, null);
			rt.arrayType = new Go$reflect.arrayType(rt, elem.Go$type(), null, len);
			return rt;
		});
		Go$arrayTypes[typeString] = typ;
	}
	return typ;
};

var Go$sliceType = function(elem) {
	var typ = elem.Go$Slice;
	if (typ === undefined) {
		typ = function(array) {
			this.array = array;
			this.offset = 0;
			this.length = array && array.length;
			this.capacity = this.length;
			this.Go$val = this;
		};
		typ.Go$string = "[]" + elem.Go$string;
		typ.Go$nil = new typ({ isNil: true, length: 0 });
		typ.Go$type = Go$cache(function() {
			var rt = new Go$reflect.rtype(0, 0, 0, 0, 0, Go$reflect.kinds.slice, null, null, Go$newStringPointer(typ.Go$string), null, null);
			rt.sliceType = new Go$reflect.sliceType(rt, elem.Go$type());
			return rt;
		});
		typ.prototype.Go$uncomparable = true;
		elem.Go$Slice = typ;
	}
	return typ;
};

var Go$mapTypes = {};
var Go$mapType = function(key, elem) {
  var typeString = "map[" + key.Go$string + "]" + elem.Go$string;
  var typ = Go$mapTypes[typeString];
  if (typ === undefined) {
    typ = function(v) { this.Go$val = v; };
		typ.Go$string = typeString;
		typ.Go$type = Go$cache(function() {
			var rt = new Go$reflect.rtype(0, 0, 0, 0, 0, Go$reflect.kinds.map, null, null, Go$newStringPointer(typeString), null, null);
			rt.mapType = new Go$reflect.mapType(rt, key.Go$type(), elem.Go$type(), null, null);
			return rt;
		});
		typ.prototype.Go$uncomparable = true;
    Go$mapTypes[typeString] = typ;
  }
  return typ;
};

var Go$Map = function() {};
(function() {
	var names = Object.getOwnPropertyNames(Object.prototype), i;
	for (i = 0; i < names.length; i += 1) {
		Go$Map.prototype[names[i]] = undefined;
	}
})();

var Go$throwNilPointerError = function() { Go$throwRuntimeError("invalid memory address or nil pointer dereference"); };
var Go$pointerType = function(elem) {
	var typ = elem.Go$Pointer;
	if (typ === undefined) {
		typ = function(getter, setter) {
			this.Go$get = getter;
			this.Go$set = setter;
			this.Go$val = this;
		};
		typ.Go$string = "*" + elem.Go$string;
		typ.Go$nil = new typ(Go$throwNilPointerError, Go$throwNilPointerError);
		typ.Go$type = Go$cache(function() {
			var rt = new Go$reflect.rtype(0, 0, 0, 0, 0, Go$reflect.kinds.ptr, null, null, Go$newStringPointer(typ.Go$string), null, null);
			rt.ptrType = new Go$reflect.ptrType(rt, elem.Go$type());
			return rt;
		});
		elem.Go$Pointer = typ;
	}
	return typ;
};

var Go$StringPointer = Go$pointerType(Go$String);
Go$newStringPointer = function(str) {
	return new Go$StringPointer(function() { return str; }, function(v) { str = v; });
};
var Go$newDataPointer = function(data, constructor) {
	return new constructor(function() { return data; }, function(v) { data = v; });
};

var Go$structNil = function(constructor) {
	var nil = new constructor();
	var fields = Object.keys(nil), i;
	for (i = 0; i < fields.length; i++) {
		var field = fields[i];
		if (field !== "Go$id" && field !== "Go$val") {
			Object.defineProperty(nil, field, { get: Go$throwNilPointerError, set: Go$throwNilPointerError });
		}
	}
	return nil;
};

var Go$Int64 = function(high, low) {
	this.high = (high + Math.floor(Math.ceil(low) / 4294967296)) >> 0;
	this.low = low >>> 0;
	this.Go$val = this;
};
Go$Int64.prototype.Go$key = function() { return "Int64$" + this.high + "$" + this.low; };
Go$Int64.Go$type = Go$cache(function() { return new Go$reflect.rtype(8, 0, 0, 0, 0, Go$reflect.kinds.int64, Go$Int64, null, Go$newStringPointer("int64"), null, null); });

var Go$Uint64 = function(high, low) {
	this.high = (high + Math.floor(Math.ceil(low) / 4294967296)) >>> 0;
	this.low = low >>> 0;
	this.Go$val = this;
};
Go$Uint64.prototype.Go$key = function() { return "Uint64$" + this.high + "$" + this.low; };
Go$Uint64.Go$type = Go$cache(function() { return new Go$reflect.rtype(8, 0, 0, 0, 0, Go$reflect.kinds.uint64, Go$Uint64, null, Go$newStringPointer("int64"), null, null); });

var Go$flatten64 = function(x) {
	return x.high * 4294967296 + x.low;
};
var Go$shiftLeft64 = function(x, y) {
	if (y === 0) {
		return x;
	}
	if (y < 32) {
		return new x.constructor(x.high << y | x.low >>> (32 - y), (x.low << y) >>> 0);
	}
	if (y < 64) {
		return new x.constructor(x.low << (y - 32), 0);
	}
	return new x.constructor(0, 0);
};
var Go$shiftRightInt64 = function(x, y) {
	if (y === 0) {
		return x;
	}
	if (y < 32) {
		return new x.constructor(x.high >> y, (x.low >>> y | x.high << (32 - y)) >>> 0);
	}
	if (y < 64) {
		return new x.constructor(x.high >> 31, (x.high >> (y - 32)) >>> 0);
	}
	if (x.high < 0) {
		return new x.constructor(-1, 4294967295);
	}
	return new x.constructor(0, 0);
};
var Go$shiftRightUint64 = function(x, y) {
	if (y === 0) {
		return x;
	}
	if (y < 32) {
		return new x.constructor(x.high >>> y, (x.low >>> y | x.high << (32 - y)) >>> 0);
	}
	if (y < 64) {
		return new x.constructor(0, x.high >>> (y - 32));
	}
	return new x.constructor(0, 0);
};
var Go$mul64 = function(x, y) {
	var high = 0, low = 0, i;
	if ((y.low & 1) !== 0) {
		high = x.high;
		low = x.low;
	}
	for (i = 1; i < 32; i += 1) {
		if ((y.low & 1<<i) !== 0) {
			high += x.high << i | x.low >>> (32 - i);
			low += (x.low << i) >>> 0;
		}
	}
	for (i = 0; i < 32; i += 1) {
		if ((y.high & 1<<i) !== 0) {
			high += x.low << i;
		}
	}
	return new x.constructor(high, low);
};
var Go$div64 = function(x, y, returnRemainder) {
	if (y.high === 0 && y.low === 0) {
		Go$throwRuntimeError("integer divide by zero");
	}

	var s = 1;
	var rs = 1;

	var xHigh = x.high;
	var xLow = x.low;
	if (xHigh < 0) {
		s = -1;
		rs = -1;
		xHigh = -xHigh;
		if (xLow !== 0) {
			xHigh -= 1;
			xLow = 4294967296 - xLow;
		}
	}

	var yHigh = y.high;
	var yLow = y.low;
	if (y.high < 0) {
		s *= -1;
		yHigh = -yHigh;
		if (yLow !== 0) {
			yHigh -= 1;
			yLow = 4294967296 - yLow;
		}
	}

	var high = 0, low = 0, n = 0, i;
	while (yHigh < 2147483648 && ((xHigh > yHigh) || (xHigh === yHigh && xLow > yLow))) {
		yHigh = (yHigh << 1 | yLow >>> 31) >>> 0;
		yLow = (yLow << 1) >>> 0;
		n += 1;
	}
	for (i = 0; i <= n; i += 1) {
		high = high << 1 | low >>> 31;
		low = (low << 1) >>> 0;
		if ((xHigh > yHigh) || (xHigh === yHigh && xLow >= yLow)) {
			xHigh = xHigh - yHigh;
			xLow = xLow - yLow;
			if (xLow < 0) {
				xHigh -= 1;
				xLow += 4294967296;
			}
			low += 1;
			if (low === 4294967296) {
				high += 1;
				low = 0;
			}
		}
		yLow = (yLow >>> 1 | yHigh << (32 - 1)) >>> 0;
		yHigh = yHigh >>> 1;
	}

	if (returnRemainder) {
		return new x.constructor(xHigh * rs, xLow * rs);
	}
	return new x.constructor(high * s, low * s);
};

var Go$Complex64  = function(real, imag) {
	this.real = real;
	this.imag = imag;
	this.Go$val = this;
};
Go$Complex64.prototype.Go$key = function() { return "Complex64$" + this.Go$val; };
Go$Complex64.Go$type = Go$cache(function() { return new Go$reflect.rtype(8, 0, 0, 0, 0, Go$reflect.kinds.complex64, Go$Complex64, null, Go$newStringPointer("complex64"), null, null); });

var Go$Complex128  = function(real, imag) {
	this.real = real;
	this.imag = imag;
	this.Go$val = this;
};
Go$Complex128.prototype.Go$key = function() { return "Complex128$" + this.Go$val; };
Go$Complex128.Go$type = Go$cache(function() { return new Go$reflect.rtype(16, 0, 0, 0, 0, Go$reflect.kinds.complex128, Go$Complex128, null, Go$newStringPointer("complex128"), null, null); });

var Go$divComplex = function(n, d) {
	var ninf = n.real === 1/0 || n.real === -1/0 || n.imag === 1/0 || n.imag === -1/0;
	var dinf = d.real === 1/0 || d.real === -1/0 || d.imag === 1/0 || d.imag === -1/0;
	var nnan = !ninf && (n.real !== n.real || n.imag !== n.imag);
	var dnan = !dinf && (d.real !== d.real || d.imag !== d.imag);
	if(nnan || dnan) {
		return new n.constructor(0/0, 0/0);
	}
	if (ninf && !dinf) {
		return new n.constructor(1/0, 1/0);
	}
	if (!ninf && dinf) {
		return new n.constructor(0, 0);
	}
	if (d.real === 0 && d.imag === 0) {
		if (n.real === 0 && n.imag === 0) {
			return new n.constructor(0/0, 0/0);
		}
		return new n.constructor(1/0, 1/0);
	}
	var a = Math.abs(d.real);
	var b = Math.abs(d.imag);
	if (a <= b) {
		var ratio = d.real / d.imag;
		var denom = d.real * ratio + d.imag;
		return new n.constructor((n.real * ratio + n.imag) / denom, (n.imag * ratio - n.real) / denom);
	}
	var ratio = d.imag / d.real;
	var denom = d.imag * ratio + d.real;
	return new n.constructor((n.imag * ratio + n.real) / denom, (n.imag - n.real * ratio) / denom);
};

var Go$subslice = function(slice, low, high, max) {
	if (low < 0 || high < low || max < high || high > slice.capacity || max > slice.capacity) {
		Go$throwRuntimeError("slice bounds out of range");
	}
	var s = new slice.constructor(slice.array);
	s.offset = slice.offset + low;
	s.length = slice.length - low;
	s.capacity = slice.capacity - low;
	if (high !== undefined) {
		s.length = high - low;
	}
	if (max !== undefined) {
		s.capacity = max - low;
	}
	return s;
};

var Go$sliceToArray = function(slice) {
	if (slice.length === 0) {
		return [];
	}
	if (slice.array.constructor !== Array) {
		return slice.array.subarray(slice.offset, slice.offset + slice.length);
	}
	return slice.array.slice(slice.offset, slice.offset + slice.length);
};

var Go$decodeRune = function(str, pos) {
	var c0 = str.charCodeAt(pos)

	if (c0 < 0x80) {
		return [c0, 1];
	}

	if (c0 === NaN || c0 < 0xC0) {
		return [0xFFFD, 1];
	}

	var c1 = str.charCodeAt(pos + 1)
	if (c1 === NaN || c1 < 0x80 || 0xC0 <= c1) {
		return [0xFFFD, 1];
	}

	if (c0 < 0xE0) {
		var r = (c0 & 0x1F) << 6 | (c1 & 0x3F);
		if (r <= 0x7F) {
			return [0xFFFD, 1];
		}
		return [r, 2];
	}

	var c2 = str.charCodeAt(pos + 2)
	if (c2 === NaN || c2 < 0x80 || 0xC0 <= c2) {
		return [0xFFFD, 1];
	}

	if (c0 < 0xF0) {
		var r = (c0 & 0x0F) << 12 | (c1 & 0x3F) << 6 | (c2 & 0x3F);
		if (r <= 0x7FF) {
			return [0xFFFD, 1];
		}
		if (0xD800 <= r && r <= 0xDFFF) {
			return [0xFFFD, 1];
		}
		return [r, 3];
	}

	var c3 = str.charCodeAt(pos + 3)
	if (c3 === NaN || c3 < 0x80 || 0xC0 <= c3) {
		return [0xFFFD, 1];
	}

	if (c0 < 0xF8) {
		var r = (c0 & 0x07) << 18 | (c1 & 0x3F) << 12 | (c2 & 0x3F) << 6 | (c3 & 0x3F);
		if (r <= 0xFFFF || 0x10FFFF < r) {
			return [0xFFFD, 1];
		}
		return [r, 4];
	}

	return [0xFFFD, 1];
};

var Go$encodeRune = function(r) {
	if (r < 0 || r > 0x10FFFF || (0xD800 <= r && r <= 0xDFFF)) {
		r = 0xFFFD;
	}
	if (r <= 0x7F) {
		return String.fromCharCode(r);
	}
	if (r <= 0x7FF) {
		return String.fromCharCode(0xC0 | r >> 6, 0x80 | (r & 0x3F));
	}
	if (r <= 0xFFFF) {
		return String.fromCharCode(0xE0 | r >> 12, 0x80 | (r >> 6 & 0x3F), 0x80 | (r & 0x3F));
	}
	return String.fromCharCode(0xF0 | r >> 18, 0x80 | (r >> 12 & 0x3F), 0x80 | (r >> 6 & 0x3F), 0x80 | (r & 0x3F));
};

var Go$stringToBytes = function(str, terminateWithNull) {
	var array = new Uint8Array(terminateWithNull ? str.length + 1 : str.length), i;
	for (i = 0; i < str.length; i += 1) {
		array[i] = str.charCodeAt(i);
	}
	if (terminateWithNull) {
		array[str.length] = 0;
	}
	return array;
};

var Go$bytesToString = function(slice) {
	if (slice.length === 0) {
		return "";
	}
	var str = "", i;
	for (i = 0; i < slice.length; i += 10000) {
		str += String.fromCharCode.apply(null, slice.array.subarray(slice.offset + i, slice.offset + Math.min(slice.length, i + 10000)));
	}
	return str;
};

var Go$stringToRunes = function(str) {
	var array = new Int32Array(str.length);
	var rune, i, j = 0;
	for (i = 0; i < str.length; i += rune[1], j += 1) {
		rune = Go$decodeRune(str, i);
		array[j] = rune[0];
	}
	return array.subarray(0, j);
};

var Go$runesToString = function(slice) {
	if (slice.length === 0) {
		return "";
	}
	var str = "", i;
	for (i = 0; i < slice.length; i += 1) {
		str += Go$encodeRune(slice.array[slice.offset + i]);
	}
	return str;
};

var Go$externalizeString = function(intStr) {
	var extStr = "", rune, i, j = 0;
	for (i = 0; i < intStr.length; i += rune[1], j += 1) {
		rune = Go$decodeRune(intStr, i);
		extStr += String.fromCharCode(rune[0]);
	}
	return extStr;
};

var Go$internalizeString = function(extStr) {
	var intStr = "", i;
	for (i = 0; i < extStr.length; i += 1) {
		intStr += Go$encodeRune(extStr.charCodeAt(i));
	}
	return intStr;
};

var Go$makeArray = function(constructor, length, zero) { // TODO do not use for typed arrays when NodeJS is behaving according to spec
	var array = new constructor(length), i;
	for (i = 0; i < length; i += 1) {
		array[i] = zero();
	}
	return array;
};

var Go$mapArray = function(array, f) {
	var newArray = new array.constructor(array.length), i;
	for (i = 0; i < array.length; i += 1) {
		newArray[i] = f(array[i]);
	}
	return newArray;
};

var Go$Struct = function() {};
var Go$Interface = function() {};
Go$Interface.Go$string = "interface{}";
Go$Interface.Go$nil = { Go$key: function() { return "nil"; } };
var Go$Channel = function() {};

var Go$copySlice = function(dst, src) {
	var n = Math.min(src.length, dst.length), i;
	if (dst.array.constructor !== Array && n !== 0) {
		dst.array.set(src.array.subarray(src.offset, src.offset + n), dst.offset);
		return n;
	}
	for (i = 0; i < n; i += 1) {
		dst.array[dst.offset + i] = src.array[src.offset + i];
	}
	return n;
};

var Go$copyString = function(dst, src) {
	var n = Math.min(src.length, dst.length), i;
	for (i = 0; i < n; i += 1) {
		dst.array[dst.offset + i] = src.charCodeAt(i);
	}
	return n;
};

var Go$copyArray = function(dst, src) {
	var i;
	for (i = 0; i < src.length; i += 1) {
		dst[i] = src[i];
	}
};

var Go$append = function(slice, toAppend) {
	if (toAppend.length === 0) {
		return slice;
	}
	if (slice.array.isNil) {
		// this must be a new array, don't just return toAppend
		slice = new toAppend.constructor(new toAppend.array.constructor(0));
	}

	var newArray = slice.array;
	var newOffset = slice.offset;
	var newLength = slice.length + toAppend.length;
	var newCapacity = slice.capacity;

	if (newLength > newCapacity) {
		var c = newArray.length - newOffset;
		newCapacity = Math.max(newLength, c < 1024 ? c * 2 : Math.floor(c * 5 / 4));

		if (newArray.constructor === Array) {
			if (newOffset !== 0) {
				newArray = newArray.slice(newOffset);
			}
			newArray.length = newCapacity;
		} else {
			newArray = new newArray.constructor(newCapacity);
			newArray.set(slice.array.subarray(newOffset))
		}
		newOffset = 0;
	}

	var leftOffset = newOffset + slice.length, rightOffset = toAppend.offset, i;
	for (i = 0; i < toAppend.length; i += 1) {
		newArray[leftOffset + i] = toAppend.array[rightOffset + i];
	}

	var newSlice = new slice.constructor(newArray);
	newSlice.offset = newOffset;
	newSlice.length = newLength;
	newSlice.capacity = newCapacity;
	return newSlice;
};

var Go$error = {};
var Go$Panic = function(value) {
	this.value = value;
	if (value.constructor === Go$String) {
		this.message = value.Go$val;
	} else if (value.Error !== undefined) {
		this.message = value.Error();
	} else if (value.String !== undefined) {
		this.message = value.String();
	} else {
		this.message = value;
	}
	Error.captureStackTrace(this, Go$Panic);
};
var Go$Exit = function() {
	Error.captureStackTrace(this, Go$Exit);
};

// TODO improve error wrapping
var Go$wrapJavaScriptError = function(err) {
	if (err.constructor === Go$Exit) {
		throw err;
	}
	var panic = new Go$Panic(err);
	panic.stack = err.stack;
	return panic;
};

var Go$errorStack = [];

// TODO inline
var Go$callDeferred = function(deferred) {
	var i;
	for (i = deferred.length - 1; i >= 0; i -= 1) {
		var call = deferred[i];
		try {
			if (call.recv !== undefined) {
				call.recv[call.method].apply(call.recv, call.args);
				continue;
			}
			call.fun.apply(undefined, call.args);
		} catch (err) {
			Go$errorStack.push({ frame: Go$getStackDepth(), error: err });
		}
	}
	var err = Go$errorStack[Go$errorStack.length - 1];
	if (err !== undefined && err.frame === Go$getStackDepth()) {
		Go$errorStack.pop();
		throw err.error;
	}
};

var Go$recover = function() {
	var err = Go$errorStack[Go$errorStack.length - 1];
	if (err === undefined || err.frame !== Go$getStackDepth() - 2) {
		return null;
	}
	Go$errorStack.pop();
	return err.error.value;
};

var Go$getStack = function() {
	return (new Error()).stack.split("\n");
};

var Go$getStackDepth = function() {
	var s = Go$getStack(), d = 0, i;
	for (i = 0; i < s.length; i += 1) {
		if (s[i].indexOf("Go$callDeferred") == -1) {
			d += 1;
		}
	}
	return d;
};

var Go$interfaceIsEqual = function(a, b) {
	if (a === null || b === null) {
		return a === null && b === null;
	}
	if (a.constructor !== b.constructor) {
		return false;
	}
	if (a.Go$uncomparable) {
		throw new Go$Panic("runtime error: comparing uncomparable type " + a.constructor);
	}
	return a.Go$val === b.Go$val;
};
var Go$arrayIsEqual = function(a, b) {
	if (a.length != b.length) {
		return false;
	}
	var i;
	for (i = 0; i < a.length; i += 1) {
		if (a[i] !== b[i]) {
			return false;
		}
	}
	return true;
};
var Go$sliceIsEqual = function(a, ai, b, bi) {
	return a.array === b.array && a.offset + ai === b.offset + bi;
};
var Go$pointerIsEqual = function(a, b) {
	if (a === b) {
		return true;
	}
	if (a.Go$get === Go$throwNilPointerError || b.Go$get === Go$throwNilPointerError) {
		return a.Go$get === Go$throwNilPointerError && b.Go$get === Go$throwNilPointerError;
	}
	var old = a.Go$get();
	var dummy = new Object();
	a.Go$set(dummy);
	var equal = b.Go$get() === dummy;
	a.Go$set(old);
	return equal;
};

var Go$typeAssertionFailed = function(obj) {
	throw new Go$Panic("type assertion failed: " + obj + " (" + obj.constructor + ")");
};

var Go$now = function() { var msec = (new Date()).getTime(); return [new Go$Int64(0, Math.floor(msec / 1000)), (msec % 1000) * 1000000]; };

var Go$packages = {};

Go$packages["go/doc"] = {
	Synopsis: function(s) { return ""; }
};
`
