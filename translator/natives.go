package translator

var Prelude = `
Error.stackTraceLimit = -1;

var Go$obj, Go$tuple;
var Go$idCounter = 1;

var Go$keys = Object.keys;
var Go$min = Math.min;
var Go$charToString = String.fromCharCode;

var Go$Uint8      = function(v) { this.v = v; };
Go$Uint8.prototype.Go$key = function() { return "Uint8$" + this.v; };
var Go$Uint16     = function(v) { this.v = v; };
Go$Uint16.prototype.Go$key = function() { return "Uint16$" + this.v; };
var Go$Uint32     = function(v) { this.v = v; };
Go$Uint32.prototype.Go$key = function() { return "Uint32$" + this.v; };
var Go$Int8       = function(v) { this.v = v; };
Go$Int8.prototype.Go$key = function() { return "Int8$" + this.v; };
var Go$Int16      = function(v) { this.v = v; };
Go$Int16.prototype.Go$key = function() { return "Int16$" + this.v; };
var Go$Int32      = function(v) { this.v = v; };
Go$Int32.prototype.Go$key = function() { return "Int32$" + this.v; };
var Go$Float32    = function(v) { this.v = v; };
Go$Float32.prototype.Go$key = function() { return "Float32$" + this.v; };
var Go$Float64    = function(v) { this.v = v; };
Go$Float64.prototype.Go$key = function() { return "Float64$" + this.v; };
var Go$Complex64  = function(v) { this.v = v; };
Go$Complex64.prototype.Go$key = function() { return "Complex64$" + this.v; };
var Go$Complex128 = function(v) { this.v = v; };
Go$Complex128.prototype.Go$key = function() { return "Complex128$" + this.v; };
var Go$Uint       = function(v) { this.v = v; };
Go$Uint.prototype.Go$key = function() { return "Uint$" + this.v; };
var Go$Int        = function(v) { this.v = v; };
Go$Int.prototype.Go$key = function() { return "Int$" + this.v; };
var Go$Uintptr    = function(v) { this.v = v; };
Go$Uintptr.prototype.Go$key = function() { return "Uintptr$" + this.v; };
var Go$Byte       = Go$Uint8;
var Go$Rune       = Go$Int32;

var Go$Bool   = function(v) { this.v = v; };
Go$Bool.prototype.Go$key = function() { return "Bool$" + this.v; };
var Go$String = function(v) { this.v = v; };
Go$String.prototype.Go$key = function() { return "String$" + this.v; };
var Go$Func   = function(v) { this.v = v; };
Go$Func.prototype.Go$key = function() { return "Func$" + this.v; };

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

var Go$Int64 = function(high, low) {
	this.high = (high + Math.floor(low / 4294967296)) | 0;
	this.low = low >>> 0;
};
Go$Int64.prototype.Go$key = function() { return "Int64$" + this.high + "$" + this.low; };

var Go$Uint64 = function(high, low) {
	this.high = (high + Math.floor(low / 4294967296)) >>> 0;
	this.low = low >>> 0;
};
Go$Uint64.prototype.Go$key = function() { return "Uint64$" + this.high + "$" + this.low; };

var Go$shiftLeft64 = function(x, y) {
	var high = 0;
	var low = 0;
	if (y < 32) {
		high = x.high << y;
		low = (x.low << y) >>> 0;
		if (y < 64) {
			high |= x.low >>> (32 - y);
		}
	} else if (y < 64) {
		high = x.low << (y - 32);
	}
	return new x.constructor(high, low);
};
var Go$shiftRightInt64 = function(x, y) {
	var high = 0;
	var low = 0;
	if (y < 32) {
		high = x.high >> y;
		low = x.low >>> y;
		if (y < 64) {
			low = (low | x.high << (32 - y)) >>> 0;
		}
	} else if (y < 64) {
		low = x.high >> (y - 32);
	}
	return new x.constructor(high, low);
};
var Go$shiftRightUint64 = function(x, y) {
	var high = 0;
	var low = 0;
	if (y < 32) {
		high = x.high >>> y;
		low = x.low >>> y;
		if (y < 64) {
			low = (low | x.high << (32 - y)) >>> 0;
		}
	} else if (y < 64) {
		low = x.high >>> (y - 32);
	}
	return new x.constructor(high, low);
};
var Go$mul64 = function(x, y) {
	var s = 1;
	if (y.high < 0) {
		y = new Go$Uint64(-y.high, -y.low);
		s = -1;
	}
	var r = new x.constructor(0, 0);
	while (y.high !== 0 || y.low !== 0) {
		if ((y.low & 1) === 1) {
			r = new x.constructor(r.high + x.high, r.low + x.low);
		}
		y = Go$shiftRightUint64(y, 1);
		x = Go$shiftLeft64(x, 1);
	}
	return new x.constructor(r.high * s, r.low * s);
};
var Go$div64 = function(x, y, returnRemainder) {
	var typ = x.constructor;
	var s = 1;
	if (y.high < 0) {
		s = -1;
	}
	y = new Go$Uint64(y.high * s, y.low * s);
	if (x.high < 0) {
		x = new Go$Uint64(-x.high, -x.low);
		s *= -1;
	}
	var r = new Go$Uint64(0, 0);
	var n = 0;
	while (y.high < 2147483648 && ((x.high > y.high) || (x.high === y.high && x.low > y.low))) {
		y = Go$shiftLeft64(y, 1);
		n += 1;
	}
	var i = 0;
	while (true) {
		if ((x.high > y.high) || (x.high === y.high && x.low >= y.low)) {
			x = new Go$Uint64(x.high - y.high, x.low - y.low);
			r = new Go$Uint64(r.high, r.low + 1);
		}
		if (i === n) {
			break;
		}
		y = Go$shiftRightUint64(y, 1);
		r = Go$shiftLeft64(r, 1);
		i += 1;
	}
	if (returnRemainder) {
		return x;
	}
	return new typ(r.high * s, r.low * s);
};

var Go$Slice = function(array) {
	this.array = array;
	this.offset = 0;
	this.length = array && array.length;
};
Go$Slice.Go$nil = new Go$Slice({ isNil: true, length: 0 });

var Go$subslice = function(slice, begin, end) {
	var s = new slice.constructor(slice.array);
	s.offset = slice.offset + begin;
	s.length = slice.length - begin;
	if (end !== undefined) {
		s.length = end - begin;
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
}

var Go$encodeRune = function(r) {
	if (r <= 0x7F) {
		return String.fromCharCode(r);
	}
	if (r <= 0x7FF) {
		return String.fromCharCode(0xC0 | r >> 6, 0x80 | (r & 0x3F));
	}
	if (r > 0x10FFFF) {
		r = 0xFFFD;
	}
	if (0xD800 <= r && r <= 0xDFFF) {
		r = 0xFFFD;
	}
	if (r <= 0xFFFF) {
		return String.fromCharCode(0xE0 | r >> 12, 0x80 | (r >> 6 & 0x3F), 0x80 | (r & 0x3F));
	}
	return String.fromCharCode(0xF0 | r >> 18, 0x80 | (r >> 12 & 0x3F), 0x80 | (r >> 6 & 0x3F), 0x80 | (r & 0x3F));
};

var Go$stringToBytes = function(str, terminateWithNull) {
	var array = new Uint8Array(terminateWithNull ? str.length + 1 : str.length);
	for (var i = 0; i < str.length; i++) {
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
	var str = "";
	for (var i = 0; i < slice.length; i += 10000) {
		str += String.fromCharCode.apply(null, slice.array.subarray(slice.offset + i, slice.offset + Math.min(slice.length, i + 10000)));
	}
	return str;
};

var Go$stringToRunes = function(str) {
	var array = new Int32Array(str.length);
	var rune;
	var j = 0;
	for (var i = 0; i < str.length; i += rune[1], j++) {
		rune = Go$decodeRune(str, i);
		array[j] = rune[0];
	}
	return array.subarray(0, j);
}

var Go$runesToString = function(slice) {
	if (slice.length === 0) {
		return "";
	}
	var str = "";
	for (var i = 0; i < slice.length; i++) {
		str += Go$encodeRune(slice.array[slice.offset + i]);
	}
	return str;
};

var Go$externalizeString = function(intStr) {
	var extStr = "";
	var rune;
	var j = 0;
	for (var i = 0; i < intStr.length; i += rune[1], j++) {
		rune = Go$decodeRune(intStr, i);
		extStr += String.fromCharCode(rune[0]);
	}
	return extStr;
};

var Go$internalizeString = function(extStr) {
	var intStr = "";
	for (var i = 0; i < extStr.length; i++) {
		intStr += Go$encodeRune(extStr.charCodeAt(i));
	}
	return intStr;
};

var Go$clear = function(array, zero) { for (var i = 0; i < array.length; i++) { array[i] = zero; } return array; }; // TODO do not use for typed arrays when NodeJS is behaving according to spec

var Go$Map = function(data) {
	data = data || [];
	for (var i = 0; i < data.length; i += 2) {
		this[data[i]] = { k: data[i], v: data[i + 1] };
	}
};
Go$Map.Go$nil = { Go$key: function() { return "nil"; } };
var Go$objectProperyNames = Object.getOwnPropertyNames(Object.prototype);
for (var i = 0; i < Go$objectProperyNames.length; i++) {
	Go$Map.prototype[Go$objectProperyNames[i]] = undefined;
}

var Go$Interface = function(value) {
	return value;
};

var Go$Channel = function() {};

var Go$Pointer = function(getter, setter) { this.Go$get = getter; this.Go$set = setter; };

var Go$copy = function(dst, src) {
	if (src.length === 0 || dst.length === 0) {
		return 0;
	}
	var n = Math.min(src.length, dst.length);
	if (dst.array.constructor !== Array) {
		dst.array.set(src.array.subarray(src.offset, src.offset + n), dst.offset);
		return n;
	}
	for (var i = 0; i < n; i++) {
		dst.array[dst.offset + i] = src.array[src.offset + i];
	}
	return n;
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

	if (newOffset + newLength > newArray.length) {
		var c = newArray.length - newOffset;
		var newCapacity = Math.max(newLength, c < 1024 ? c * 2 : Math.floor(c * 5 / 4));

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

	var leftOffset = newOffset + slice.length;
	var rightOffset = toAppend.offset;
	for (var j = 0; j < toAppend.length; j++) {
		newArray[leftOffset + j] = toAppend.array[rightOffset + j];
	}

	var newSlice = new slice.constructor(newArray);
	newSlice.offset = newOffset;
	newSlice.length = newLength;
	return newSlice;
};

var Go$error = {};
var Go$Panic = function(value) {
	this.message = value;
	Error.captureStackTrace(this, Go$Panic);
};

// TODO improve error wrapping
var Go$wrapJavaScriptError = function(err) {
	var panic = new Go$Panic(err);
	panic.stack = err.stack;
	return panic;
};

var Go$errorStack = [];

// TODO inline
var Go$callDeferred = function(deferred) {
	for (var i = deferred.length - 1; i >= 0; i--) {
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
}

var Go$recover = function() {
	var err = Go$errorStack[Go$errorStack.length - 1];
	if (err === undefined || err.frame !== Go$getStackDepth() - 2) {
		return null;
	}
	Go$errorStack.pop();
	return err.error.message;
};

var Go$getStackDepth = function() {
	var s = (new Error()).stack.split("\n");
	var d = 0;
	for (var i = 0; i < s.length; i++) {
		if (s[i].indexOf("Go$callDeferred") == -1) {
			d++;
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
	if (a.v !== undefined && a.constructor !== Go$Func) {
		return a.v === b.v;
	}
	throw new Go$Panic("runtime error: comparing uncomparable type " + a.constructor);
};
var Go$sliceIsEqual = function(a, ai, b, bi) {
	return a.array === b.array && a.offset + ai === b.offset + bi;
};

var Go$typeAssertionFailed = function(obj) {
	throw new Go$Panic("type assertion failed: " + obj + " (" + obj.constructor + ")");
};

var Go$now = function() { var msec = (new Date()).getTime(); return [Math.floor(msec / 1000), (msec % 1000) * 1000000]; };

var Go$packages = {};

// --- fake reflect package ---

Go$Bool.Kind    = function() { return 1; };
Go$Int.Kind     = function() { return 2; };
Go$Int8.Kind    = function() { return 3; };
Go$Int16.Kind   = function() { return 4; };
Go$Int32.Kind   = function() { return 5; };
Go$Int64.Kind   = function() { return 6; };
Go$Uint.Kind    = function() { return 7; };
Go$Uint8.Kind   = function() { return 8; };
Go$Uint16.Kind  = function() { return 9; };
Go$Uint32.Kind  = function() { return 10; };
Go$Uint64.Kind  = function() { return 11; };
Go$Uintptr.Kind = function() { return 12; };
Go$Float32.Kind = function() { return 13; };
Go$Float64.Kind = function() { return 14; };
Go$Complex64    = function() { return 15; };
Go$Complex128   = function() { return 16; };
Go$String.Kind  = function() { return 24; };
Go$Int.Bits = Go$Uintptr.Bits = function() { return 32; };
Go$Float64.Bits = function() { return 64; };
Go$Complex128.Bits = function() { return 128; };

Go$packages["reflect"] = {
	DeepEqual: function(a, b) {
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
			for (var i = 0; i < a.length; i++) {
				if (!this.DeepEqual(a.array[a.offset + i], b.array[b.offset + i])) {
					return false;
				}
			}
			return true;
		}
		var keys = Object.keys(a);
		for (var j = 0; j < keys.length; j++) {
			var key = keys[j];
			if (key !== "Go$id" && !this.DeepEqual(a[key], b[key])) {
				return false;
			}
		}
		return true;
	},
	TypeOf: function(v) {
		return v.constructor;
	},
	flag: function() {},
	Value: function() {}
};

Go$packages["go/doc"] = {
	Synopsis: function(s) { return ""; }
};
`

var natives = map[string]string{
	"bytes": `
		IndexByte = function(s, c) {
			for (var i = 0; i < s.length; i++) {
				if (s.array[s.offset + i] === c) {
					return i;
				}
			}
			return -1;
		};
		Equal = function(a, b) {
			if (a.length !== b.length) {
				return false;
			}
			for (var i = 0; i < a.length; i++) {
				if (a.array[a.offset + i] !== b.array[b.offset + i]) {
					return false;
				}
			}
			return true;
		}
	`,

	"math": `
		Go$pkg.MaxFloat32 = 3.40282346638528859811704183484516925440e+38;
		Go$pkg.SmallestNonzeroFloat32 = 1.401298464324817070923729583289916131280e-45;
		Go$pkg.MaxFloat64 = 1.797693134862315708145274237317043567981e+308;
		Go$pkg.SmallestNonzeroFloat64 = 4.940656458412465441765687928682213723651e-324;

		Abs = Math.abs;
		Exp = Math.exp;
		Exp2 = function(x) { return Math.exp(x * Math.log(2)); };
		// Exp2 = exp2; // TODO fix and use for higher precision
		Frexp = frexp;
		Ldexp = function(frac, exp) { return frac * Math.exp(exp * Math.log(2)); };
		// Ldexp = ldexp; // TODO fix and use for higher precision
		Log = Math.log;
		Log2 = log2;

		// generated from bitcasts/bitcasts.go
		Float32bits = (function(f) {
			if (f === 0) {
				return 0;
			}
			if (f !== f) {
				return 4294967295;
			}
			var s = 0;
			if (f < 0) {
				s = 2147483648;
				f = -f;
			}
			var e = 150;
			while (f >= 16777216) {
				f = f / (2);
				if (e === 255) {
					break;
				}
				e = (e + (1) >>> 0);
			}
			while (f < 8388608) {
				e = (e - (1) >>> 0);
				if (e === 0) {
					break;
				}
				f = f * (2);
			}
			var y;
			return ((((s | ((y = 23, y < 32 ? (e << y) : 0) >>> 0)) >>> 0) | (((Math.floor(f) &~ 8388608) >>> 0))) >>> 0);
		});
		Float32frombits = (function(b) {
			var s = 1;
			if (((b & 2147483648) >>> 0) !== 0) {
				s = -1;
			}
			var y;
			var e = (((((y = 23, y < 32 ? (b >>> y) : 0) >>> 0)) & 255) >>> 0);
			var m = ((b & 8388607) >>> 0);
			if (e === 255) {
				if (m === 0) {
					return s / 0;
				}
				return (s / 0) - (s / 0);
			}
			if (e !== 0) {
				m = (m + (8388608) >>> 0);
			}
			if (e === 0) {
				e = 1;
			}
			return Ldexp(m, e - 127 - 23) * s;
		});
		Float64bits = (function(f) {
			if (f === 0) {
				return new Go$Uint64(0, 0);
			}
			if (f !== f) {
				return new Go$Uint64(4294967295, 4294967295);
			}
			var s = new Go$Uint64(0, 0);
			if (f < 0) {
				s = new Go$Uint64(2147483648, 0);
				f = -f;
			}
			var e = 1075;
			while (f >= 9007199254740992) {
				f = f / (2);
				if (e === 2047) {
					break;
				}
				e = (e + (1) >>> 0);
			}
			while (f < 4503599627370496) {
				e = (e - (1) >>> 0);
				if (e === 0) {
					break;
				}
				f = f * (2);
			}
			var x, y;
			var x1, y1;
			var x2, y2;
			return (x2 = (x = s, y = Go$shiftLeft64(new Go$Uint64(0, e), 52), new Go$Uint64(x.high | y.high, (x.low | y.low) >>> 0)), y2 = ((x1 = new Go$Uint64(0, Math.floor(f)), y1 = new Go$Uint64(1048576, 0), new Go$Uint64(x1.high &~ y1.high, (x1.low &~ y1.low) >>> 0))), new Go$Uint64(x2.high | y2.high, (x2.low | y2.low) >>> 0));
		});
		Float64frombits = (function(b) {
			var s = 1;
			var x, y;
			var x1, y1;
			if ((x1 = (x = b, y = new Go$Uint64(2147483648, 0), new Go$Uint64(x.high & y.high, (x.low & y.low) >>> 0)), y1 = new Go$Uint64(0, 0), x1.high !== y1.high || x1.low !== y1.low)) {
				s = -1;
			}
			var x2, y2;
			var e = (x2 = (Go$shiftRightUint64(b, 52)), y2 = new Go$Uint64(0, 2047), new Go$Uint64(x2.high & y2.high, (x2.low & y2.low) >>> 0));
			var x3, y3;
			var m = (x3 = b, y3 = new Go$Uint64(1048575, 4294967295), new Go$Uint64(x3.high & y3.high, (x3.low & y3.low) >>> 0));
			var x4, y4;
			if ((x4 = e, y4 = new Go$Uint64(0, 2047), x4.high === y4.high && x4.low === y4.low)) {
				var x5, y5;
				if ((x5 = m, y5 = new Go$Uint64(0, 0), x5.high === y5.high && x5.low === y5.low)) {
					return s / 0;
				}
				return (s / 0) - (s / 0);
			}
			var x6, y6;
			if ((x6 = e, y6 = new Go$Uint64(0, 0), x6.high !== y6.high || x6.low !== y6.low)) {
				var x7, y7;
				m = (x7 = m, y7 = (new Go$Uint64(1048576, 0)), new Go$Uint64(x7.high + y7.high, x7.low + y7.low));
			}
			var x8, y8;
			if ((x8 = e, y8 = new Go$Uint64(0, 0), x8.high === y8.high && x8.low === y8.low)) {
				e = new Go$Uint64(0, 1);
			}
			return Ldexp((Go$obj = m, Go$obj.high * 4294967296 + Go$obj.low), (e.low | 0) - 1023 - 52) * s;
		});
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
		Go$pkg.Args = new Go$Slice(Go$webMode ? [] : process.argv.slice(1));
	`,

	"runtime": `
		Go$pkg.sizeof_C_MStats = 3696;
		getgoroot = function() { return Go$webMode ? "/" : (process.env["GOROOT"] || ""); };
		SetFinalizer = function() {};
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
		if (!Go$webMode) {
			var syscall = require("syscall");
			Syscall = syscall.Syscall;
			Syscall6 = syscall.Syscall6;
			RawSyscall = syscall.Syscall;
			RawSyscall6 = syscall.Syscall6;
			BytePtrFromString = function(s) { return [Go$stringToBytes(s, true), null]; };

			var envkeys = Object.keys(process.env);
			Go$pkg.envs = new Go$Slice(new Array(envkeys.length));
			for(var i = 0; i < envkeys.length; i++) {
				Go$pkg.envs.array[i] = envkeys[i] + "=" + process.env[envkeys[i]];
			}
		} else {
			Go$pkg.Go$setSyscall = function(f) {
				Syscall = Syscall6 = RawSyscall = RawSyscall6 = f;
			}
			Go$pkg.Go$setSyscall(function() { throw "Syscalls not available in browser." });
			Go$pkg.envs = new Go$Slice(new Array(0));
		}
	`,

	"time": `
		now = Go$now;
		AfterFunc = function(d, f) {
			setTimeout(f, Go$div64(d, Go$pkg.Millisecond).low);
			return null;
		};
	`,
}
