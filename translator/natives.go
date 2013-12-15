package translator

var Prelude = `
Error.stackTraceLimit = -1;

var Go$obj, Go$tuple;
var Go$idCounter = 1;
var Go$keys = Object.keys;
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

var Go$Map = function(data) {
	data = data || [];
	var i;
	for (i = 0; i < data.length; i += 2) {
		this[data[i]] = { k: data[i], v: data[i + 1] };
	}
};
Go$Map.Go$nil = { Go$key: function() { return "nil"; } };
(function() {
	var Go$objectProperyNames = Object.getOwnPropertyNames(Object.prototype);
	var i;
	for (i = 0; i < Go$objectProperyNames.length; i += 1) {
		Go$Map.prototype[Go$objectProperyNames[i]] = undefined;
	}
})();

var Go$Struct = function() {};
var Go$Interface = function() {};
Go$Interface.Go$string = "interface{}";
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
	if (a.Go$uncomparable || a.Go$val === undefined) { // TODO improve interfaces of maps
		throw new Go$Panic("runtime error: comparing uncomparable type " + a.constructor);
	}
	return a.Go$val === b.Go$val;
};
var Go$arrayIsEqual = function(a, b) {
	if (a === null || b === null) {
		return a === null && b === null;
	}
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

var natives = map[string]string{
	"bytes": `
		Compare = function(a, b) {
			var l = Math.min(a.length, b.length), i;
			for (i = 0; i < a.length; i += 1) {
				var va = a.array[a.offset + i];
				var vb = b.array[b.offset + i];
				if (va < vb) {
					return -1;
				}
				if (va > vb) {
					return 1;
				}
			}
			if (a.length < b.length) {
				return -1;
			}
			if (a.length > b.length) {
				return 1;
			}
			return 0;
		};
		Equal = function(a, b) {
			if (a.length !== b.length) {
				return false;
			}
			var i;
			for (i = 0; i < a.length; i += 1) {
				if (a.array[a.offset + i] !== b.array[b.offset + i]) {
					return false;
				}
			}
			return true;
		};
		IndexByte = function(s, c) {
			var i;
			for (i = 0; i < s.length; i += 1) {
				if (s.array[s.offset + i] === c) {
					return i;
				}
			}
			return -1;
		};
	`,

	"io/ioutil": `
		var blackHoles = [];
		blackHole = function() {
			return blackHoles.pop() || new (Go$sliceType(Go$Byte))(Go$makeArray(Go$ByteArray, 8192, function() { return 0; }));
		};
		blackHolePut = function(p) {
			blackHoles.push(p);
		};
	`,

	"math": `
		Abs = Math.abs;
		Acos = Math.acos;
		Asin = Math.asin;
		Atan = Math.atan;
		Atan2 = Math.atan2;
		Ceil = Math.ceil;
		Copysign = function(x, y) { return (x < 0 || 1/x === 1/-0) !== (y < 0 || 1/y === 1/-0) ? -x : x; };
		Cos = Math.cos;
		Dim = function(x, y) { return Math.max(x - y, 0); };
		Exp = Math.exp;
		Exp2 = function(x) { return Math.pow(2, x); };
		Expm1 = expm1;
		Floor = Math.floor;
		Frexp = frexp;
		Hypot = hypot;
		Inf = function(sign) { return sign >= 0 ? 1/0 : -1/0; };
		IsInf = function(f, sign) { if (f === -1/0) { return sign <= 0; } if (f === 1/0) { return sign >= 0; } return false; };
		IsNaN = function(f) { return f !== f; };
		Ldexp = function(frac, exp) {
			if (frac === 0) { return frac; };
			if (exp >= 1024) { return frac * Math.pow(2, 1023) * Math.pow(2, exp - 1023); }
			if (exp <= -1024) { return frac * Math.pow(2, -1023) * Math.pow(2, exp + 1023); }
			return frac * Math.pow(2, exp);
		};
		Log = Math.log;
		Log1p = log1p;
		Log2 = log2;
		Log10 = log10;
		Max = function(x, y) { return (x === 1/0 || y === 1/0) ? 1/0 : Math.max(x, y); };
		Min = function(x, y) { return (x === -1/0 || y === -1/0) ? -1/0 : Math.min(x, y); };
		Mod = function(x, y) { return x % y; };
		Modf = function(f) { if (f === -1/0 || f === 1/0) { return [f, 0/0] } var frac = f % 1; return [f - frac, frac]; };
		NaN = function() { return 0/0; };
		Pow = function(x, y) { return ((x === 1) || (x === -1 && (y === -1/0 || y === 1/0))) ? 1 : Math.pow(x, y); };
		Remainder = remainder;
		Signbit = function(x) { return x < 0 || 1/x === 1/-0; };
		Sin = Math.sin;
		Sincos = function(x) { return [Math.sin(x), Math.cos(x)]; };
		Sqrt = Math.sqrt;
		Tan = Math.tan;
		Trunc = function(x) { return (x === 1/0 || x === -1/0 || x !== x || 1/x === 1/-0) ? x : x >> 0; };

		// generated from bitcasts/bitcasts.go
		Float32bits = function(f) {
			var s, e;
			if (f === 0) {
				if (f === 0 && 1 / f === 1 / -0) {
					return 2147483648;
				}
				return 0;
			}
			if (!(f === f)) {
				return 2143289344;
			}
			s = 0;
			if (f < 0) {
				s = 2147483648;
				f = -f;
			}
			e = 150;
			while (f >= 1.6777216e+07) {
				f = f / (2);
				if (e === 255) {
					break;
				}
				e = (e + (1) >>> 0);
			}
			while (f < 8.388608e+06) {
				e = (e - (1) >>> 0);
				if (e === 0) {
					break;
				}
				f = f * (2);
			}
			return ((((s | (((e >>> 0) << 23) >>> 0)) >>> 0) | (((((f >> 0) >>> 0) &~ 8388608) >>> 0))) >>> 0);
		};
		Float32frombits = function(b) {
			var s, e, m;
			s = 1;
			if (!(((b & 2147483648) >>> 0) === 0)) {
				s = -1;
			}
			e = (((((b >>> 23) >>> 0)) & 255) >>> 0);
			m = ((b & 8388607) >>> 0);
			if (e === 255) {
				if (m === 0) {
					return s / 0;
				}
				return 0/0;
			}
			if (!(e === 0)) {
				m = (m + (8388608) >>> 0);
			}
			if (e === 0) {
				e = 1;
			}
			return Ldexp(m, e - 127 - 23) * s;
		};
		Float64bits = function(f) {
			var s, e, x, y, x$1, y$1, x$2, y$2;
			if (f === 0) {
				if (f === 0 && 1 / f === 1 / -0) {
					return new Go$Uint64(2147483648, 0);
				}
				return new Go$Uint64(0, 0);
			}
			if (!(f === f)) {
				return new Go$Uint64(2146959360, 1);
			}
			s = new Go$Uint64(0, 0);
			if (f < 0) {
				s = new Go$Uint64(2147483648, 0);
				f = -f;
			}
			e = 1075;
			while (f >= 9.007199254740992e+15) {
				f = f / (2);
				if (e === 2047) {
					break;
				}
				e = (e + (1) >>> 0);
			}
			while (f < 4.503599627370496e+15) {
				e = (e - (1) >>> 0);
				if (e === 0) {
					break;
				}
				f = f * (2);
			}
			return (x$2 = (x = s, y = Go$shiftLeft64(new Go$Uint64(0, e), 52), new Go$Uint64(x.high | y.high, (x.low | y.low) >>> 0)), y$2 = ((x$1 = new Go$Uint64(0, f), y$1 = new Go$Uint64(1048576, 0), new Go$Uint64(x$1.high &~ y$1.high, (x$1.low &~ y$1.low) >>> 0))), new Go$Uint64(x$2.high | y$2.high, (x$2.low | y$2.low) >>> 0));
		};
		Float64frombits = function(b) {
			var s, x, y, x$1, y$1, x$2, y$2, e, x$3, y$3, m, x$4, y$4, x$5, y$5, x$6, y$6, x$7, y$7, x$8, y$8;
			s = 1;
			if (!((x$1 = (x = b, y = new Go$Uint64(2147483648, 0), new Go$Uint64(x.high & y.high, (x.low & y.low) >>> 0)), y$1 = new Go$Uint64(0, 0), x$1.high === y$1.high && x$1.low === y$1.low))) {
				s = -1;
			}
			e = (x$2 = (Go$shiftRightUint64(b, 52)), y$2 = new Go$Uint64(0, 2047), new Go$Uint64(x$2.high & y$2.high, (x$2.low & y$2.low) >>> 0));
			m = (x$3 = b, y$3 = new Go$Uint64(1048575, 4294967295), new Go$Uint64(x$3.high & y$3.high, (x$3.low & y$3.low) >>> 0));
			if ((x$4 = e, y$4 = new Go$Uint64(0, 2047), x$4.high === y$4.high && x$4.low === y$4.low)) {
				if ((x$5 = m, y$5 = new Go$Uint64(0, 0), x$5.high === y$5.high && x$5.low === y$5.low)) {
					return s / 0;
				}
				return 0/0;
			}
			if (!((x$6 = e, y$6 = new Go$Uint64(0, 0), x$6.high === y$6.high && x$6.low === y$6.low))) {
				m = (x$7 = m, y$7 = (new Go$Uint64(1048576, 0)), new Go$Uint64(x$7.high + y$7.high, x$7.low + y$7.low));
			}
			if ((x$8 = e, y$8 = new Go$Uint64(0, 0), x$8.high === y$8.high && x$8.low === y$8.low)) {
				e = new Go$Uint64(0, 1);
			}
			return Ldexp((Go$obj = m, Go$obj.high * 4294967296 + Go$obj.low), e.low - 1023 - 52) * s;
		};
	`,

	"math/big": `
		mulWW = mulWW_g;
		divWW = divWW_g;
		addVV = addVV_g;
		subVV = subVV_g;
		addVW = addVW_g;
		subVW = subVW_g;
		shlVU = shlVU_g;
		shrVU = shrVU_g;
		mulAddVWW = mulAddVWW_g;
		addMulVVW = addMulVVW_g;
		divWVW = divWVW_g;
		bitLen = bitLen_g;
	`,

	"os": `
		Go$pkg.Args = new (Go$sliceType(Go$String))((typeof process !== 'undefined') ? process.argv.slice(1) : []);
	`,

	"reflect": `
		Go$reflect = {
			rtype: rtype, uncommonType: uncommonType, arrayType: arrayType, ptrType: ptrType, sliceType: sliceType, structType: structType, structField: structField,
			kinds: {bool: Go$pkg.Bool, int: Go$pkg.Int, int8: Go$pkg.Int8, int16: Go$pkg.Int16, int32: Go$pkg.Int32, int64: Go$pkg.Int64, uint: Go$pkg.Uint, uint8: Go$pkg.Uint8, uint16: Go$pkg.Uint16, uint32: Go$pkg.Uint32, uint64: Go$pkg.Uint64, uintptr: Go$pkg.Uintptr, float32: Go$pkg.Float32, float64: Go$pkg.Float64, complex64: Go$pkg.Complex64, complex128: Go$pkg.Complex128, array: Go$pkg.Array, chan: Go$pkg.Chan, func: Go$pkg.Func, interface: Go$pkg.Interface, map: Go$pkg.Map, ptr: Go$pkg.Ptr, slice: Go$pkg.Slice, string: Go$pkg.String, struct: Go$pkg.Struct, "unsafe.Pointer": Go$pkg.UnsafePointer}
		};

		TypeOf = function(i) {
			return i.constructor.Go$type();
		};
		ValueOf = function(i) {
			var typ = i.constructor.Go$type();
			var flag = typ.Kind() << flagKindShift;
			return new Value(typ, i.Go$val, flag);
		};

		Value.prototype.Bytes = function() {
			this.mustBe(Go$pkg.Slice);
			if (this.typ.Elem().Kind() !== Go$pkg.Uint8) {
				throw new Go$Panic("reflect.Value.Bytes of non-byte slice");
			}
			return this.val;
		};
		Value.prototype.Field = function(i) {
			this.mustBe(Go$pkg.Struct);
			var tt = this.typ.structType;
			if (i < 0 || i >= tt.fields.length) {
				throw new Go$Panic("reflect: Field index out of range");
			}
			var field = tt.fields.array[i];
			var fl = field.typ.Kind() << flagKindShift;
			return new Value(field.typ, this.val[field.name.Go$get()], fl);
		};
		Value.prototype.Index = function(i) {
			var k = this.kind();
			switch (k) {
			case Go$pkg.Array:
				var tt = this.typ.arrayType;
				if (i < 0 || i >= tt.len) {
					throw new Go$Panic("reflect: array index out of range");
				}
				var typ = tt.elem;
				var fl = this.flag & (flagRO | flagIndir | flagAddr);
				fl |= typ.Kind() << flagKindShift;
				return new Value(typ, this.val[i], fl);
			case Go$pkg.Slice:
				if (i < 0 || i >= this.val.length) {
					throw new Go$Panic("reflect: slice index out of range");
				}
				var typ = this.typ.sliceType.elem;
				var fl = flagAddr | flagIndir | (this.flag & flagRO);
				fl |= typ.Kind() << flagKindShift;
				i += this.val.offset;
				var array = this.val.array;
				return new Value(typ, new (Go$pointerType(typ))(function() { return array[i]; }, function(v) { array[i] = v; }), fl);
			case Go$pkg.String:
				if (i < 0 || i >= this.val.length) {
					throw new Go$Panic("reflect: string index out of range");
				}
				var fl = (this.flag & flagRO) | (Go$pkg.Uint8 << flagKindShift);
				return new Value(uint8Type, this.val.charCodeAt(i), fl);
			}
			throw new Go$Panic(new ValueError("reflect.Value.Index", k));
		};
		Value.prototype.Len = function() {
			var k = this.kind();
			switch (k) {
			case Go$pkg.Array:
			case Go$pkg.Slice:
			case Go$pkg.String:
				return this.val.length;
			}
			throw new Go$Panic(new ValueError("reflect.Value.Len", k));
		};
		valueInterface = function(v, safe) {
			if (v.val.constructor === v.typ.alg) {
				return v.val;
			}
			return new v.typ.alg(v.val);
		};
		Value.prototype.String = function() {
			switch (this.kind()) {
			case Go$pkg.Invalid:
				return "<invalid Value>";
			case Go$pkg.String:
				if ((this.flag & flagIndir) != 0) {
					return this.val.Go$get();
				}
				return this.val;
			}
			return "<" + this.typ.String() + " Value>";
		};
		
		DeepEqual = function(a, b) { // TODO use package version
			var i;
			if (a === b) {
				return true;
			}
			if (a.constructor === Number) {
				return false;
			}
			if (a.constructor !== b.constructor) {
				return false;
			}
			if (a.length !== undefined) {
				if (a.length !== b.length) {
					return false;
				}
				if (a.array !== undefined) {
					for (i = 0; i < a.length; i += 1) {
						if (!this.DeepEqual(a.array[a.offset + i], b.array[b.offset + i])) {
							return false;
						}
					}
				} else {
					for (i = 0; i < a.length; i += 1) {
						if (!this.DeepEqual(a[i], b[i])) {
							return false;
						}
					}
				}
				return true;
			}
			var keys = Object.keys(a), j;
			for (j = 0; j < keys.length; j += 1) {
				var key = keys[j];
				if (key !== "Go$id" && key !== "Go$val" && !this.DeepEqual(a[key], b[key])) {
					return false;
				}
			}
			return true;
		};
	`,

	"runtime": `
		Go$throwRuntimeError = function(msg) { throw new Go$Panic(new errorString(msg)) };
		sizeof_C_MStats = 3712;
		getgoroot = function() { return (typeof process !== 'undefined') ? (process.env["GOROOT"] || "") : "/"; };
		Caller = function(skip) {
			var line = Go$getStack()[skip + 3];
			if (line === undefined) {
				return [0, "", 0, false];
			}
			var parts = line.substring(line.indexOf("(") + 1, line.indexOf(")")).split(":");
			return [0, parts[0], parseInt(parts[1]), true];
		};
		GC = function() {};
		GOMAXPROCS = function(n) {
			if (n > 1) {
				Go$throwRuntimeError("GOMAXPROCS != 1 is not possible in JavaScript.")
			}
			return 1;
		};
		Goexit = function() { throw new Go$Exit(); };
		ReadMemStats = function() {};
		SetFinalizer = function() {};
	`,

	"strings": `
		IndexByte = function(s, c) { return s.indexOf(String.fromCharCode(c)); };
	`,

	"sync": `
		runtime_Syncsemcheck = function() {};
		Go$pointerType(copyChecker).prototype.check = function() {};
	`,

	"sync/atomic": `
		AddInt32 = AddInt64 = AddUint32 = AddUint64 = AddUintptr = function(addr, delta) {
			var value = addr.Go$get() + delta;
			addr.Go$set(value);
			return value;
		};
		CompareAndSwapInt32 = CompareAndSwapInt64 = CompareAndSwapUint32 = CompareAndSwapUint64 = CompareAndSwapUintptr = function(addr, oldVal, newVal) {
			if (addr.Go$get() === oldVal) {
				addr.Go$set(newVal);
				return true;
			}
			return false;
		};
		StoreInt32 = StoreInt64 = StoreUint32 = StoreUint64 = StoreUintptr = function(addr, val) {
			addr.Go$set(val);
		};
		LoadInt32 = LoadInt64 = LoadUint32 = LoadUint64 = LoadUintptr = function(addr) {
			return addr.Go$get();
		};
	`,

	"syscall": `
		if (typeof process !== 'undefined') {
			var syscall = require("syscall");
			Syscall = syscall.Syscall;
			Syscall6 = syscall.Syscall6;
			RawSyscall = syscall.Syscall;
			RawSyscall6 = syscall.Syscall6;
			BytePtrFromString = function(s) { return [Go$stringToBytes(s, true), null]; };

			var envkeys = Object.keys(process.env);
			envs = new (Go$sliceType(Go$String))(new Array(envkeys.length));
			var i;
			for(i = 0; i < envkeys.length; i += 1) {
				envs.array[i] = envkeys[i] + "=" + process.env[envkeys[i]];
			}
		} else {
			Go$pkg.Go$setSyscall = function(f) {
				Syscall = Syscall6 = RawSyscall = RawSyscall6 = f;
			}
			Go$pkg.Go$setSyscall(function() { throw "Syscalls not available in browser." });
			envs = new (Go$sliceType(Go$String))(new Array(0));
		}
	`,

	"testing": `
		Go$pkg.RunTests2 = function(pkgPath, dir, names, tests) {
			if (tests.length === 0) {
				console.log("?   \t" + pkgPath + "\t[no test files]");
				return;
			}
			os.Open(dir)[0].Chdir();
			var start = time.Now(), status = "ok  ", i;
			for (i = 0; i < tests.length; i += 1) {
				var t = new T(new common(new sync.RWMutex(), Go$sliceType(Go$Byte).Go$nil, false, false, time.Now(), new time.Duration(0, 0), null, null), names[i], null);
				var err = null;
				try {
					if (chatty.Go$get()) {
						console.log("=== RUN " + t.name);
					}
					tests[i](t);
				} catch (e) {
					if (e.constructor !== Go$Exit) {
						t.Fail();
						err = e;
					}
				}
				t.common.duration = time.Now().Sub(t.common.start);
				t.report();
				if (err !== null) {
					throw err;
				}
				if (t.common.failed) {
					status = "FAIL";
				}
			}
			var duration = time.Now().Sub(start);
			fmt.Printf("%s\t%s\t%.3fs\n", new (Go$sliceType(Go$Interface))([new Go$String(status), new Go$String(pkgPath), new Go$Float64(duration.Seconds())]));
		};
	`,

	"time": `
		now = Go$now;
		After = function() { Go$throwRuntimeError("not supported by GopherJS: time.After (use time.AfterFunc instead)") };
		AfterFunc = function(d, f) {
			setTimeout(f, Go$div64(d, Go$pkg.Millisecond).low);
			return null;
		};
		NewTimer = function() { Go$throwRuntimeError("not supported by GopherJS: time.NewTimer (use time.AfterFunc instead)") };
		Sleep = function() { Go$throwRuntimeError("not supported by GopherJS: time.Sleep (use time.AfterFunc instead)") };
		Tick = function() { Go$throwRuntimeError("not supported by GopherJS: time.Tick (use time.AfterFunc instead)") };
	`,
}
