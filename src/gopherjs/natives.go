package main

// TODO cleanup global names
var prelude = `
#!/usr/bin/env node

"use strict";
Error.stackTraceLimit = -1;

var Go$obj, Go$tuple;
var Go$idCounter = 1;

var Go$nil = { Go$id: 0 };
Go$nil.Go$subslice = function(begin, end) {
	if (begin !== 0 || (end || 0) !== 0) {
		throw new GoError("runtime error: slice bounds out of range");
	}
	return null;
};
Go$nil.Go$toArray = function() { return []; }
Go$nil.array = [];
Go$nil.offset = 0;
Go$nil.length = 0;

var Go$keys = Object.keys;
var Go$min = Math.min;
var Go$fromCharCode = String.fromCharCode;

var Go$copyFields = function(from, to) {
	var keys = Object.keys(from);
	for (var i = 0; i < keys.length; i++) {
		var key = keys[i];
		to[key] = from[key];
	}
};

var Go$Uint8      = function(v) { this.v = v; this.Go$id = "Uint8$" + v; };
var Go$Uint16     = function(v) { this.v = v; this.Go$id = "Uint16$" + v; };
var Go$Uint32     = function(v) { this.v = v; this.Go$id = "Uint32$" + v; };
var Go$Int8       = function(v) { this.v = v; this.Go$id = "Int8$" + v; };
var Go$Int16      = function(v) { this.v = v; this.Go$id = "Int16$" + v; };
var Go$Int32      = function(v) { this.v = v; this.Go$id = "Int32$" + v; };
var Go$Float32    = function(v) { this.v = v; this.Go$id = "Float32$" + v; };
var Go$Float64    = function(v) { this.v = v; this.Go$id = "Float64$" + v; };
var Go$Complex64  = function(v) { this.v = v; this.Go$id = "Complex64$" + v; };
var Go$Complex128 = function(v) { this.v = v; this.Go$id = "Complex128$" + v; };
var Go$Uint       = function(v) { this.v = v; this.Go$id = "Uint$" + v; };
var Go$Int        = function(v) { this.v = v; this.Go$id = "Int$" + v; };
var Go$Uintptr    = function(v) { this.v = v; this.Go$id = "Uintptr$" + v; };
var Go$Byte       = Go$Uint8;
var Go$Rune       = Go$Int32;

var Go$Bool   = function(v) { this.v = v; this.Go$id = "Bool$" + v; };
var Go$String = function(v) { this.v = v; this.Go$id = "String$" + v; };
var Go$Func   = function(v) { this.v = v; this.Go$id = "Func$" + v; };

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
	this.low = (low + 4294967296) % 4294967296;
	this.Go$id = "Int64$" + this.high + "$" + this.low;
};
var Go$Uint64 = function(high, low) {
	this.high = (high + Math.floor(low / 4294967296) + 4294967296) % 4294967296;
	this.low = (low + 4294967296) % 4294967296;
	this.Go$id = "Uint64$" + this.high + "$" + this.low;
};
var Go$shift64 = function(x, y) {
	var p = Math.pow(2, y);
	var high = Math.floor(x.high * p % 4294967296) + Math.floor(x.low  / 4294967296 * p % 4294967296);
	var low  = Math.floor(x.low  * p % 4294967296) + Math.floor(x.high * 4294967296 * p % 4294967296);
	return new x.constructor(high, low);
};
var Go$mul64 = function(x, y) {
	var r = new x.constructor(0, 0);
	while (y.high !== 0 || y.low !== 0) {
		if ((y.low & 1) === 1) {
			r = new x.constructor(r.high + x.high, r.low + x.low);
		}
		y = Go$shift64(y, -1);
		x = Go$shift64(x,  1);
	}
	return r;
};
var Go$div64 = function(x, y, returnRemainder) {
	var r = new x.constructor(0, 0);
	var n = 0;
	while (y.high < 2147483648 && ((x.high > y.high) || (x.high === y.high && x.low > y.low))) {
		y = Go$shift64(y, 1);
		n += 1;
	}
	var i = 0;
	while (true) {
		if ((x.high > y.high) || (x.high === y.high && x.low >= y.low)) {
			x = new x.constructor(x.high - y.high, x.low - y.low);
			r = new x.constructor(r.high, r.low + 1);
		}
		if (i === n) {
			break;
		}
		y = Go$shift64(y, -1);
		r = Go$shift64(r,  1);
		i += 1;
	}
	if (returnRemainder) {
		return x;
	}
	return r;
};

var Go$Slice = function(data, length, capacity) {
	capacity = capacity || length || 0;
	data = data || new Go$Array(capacity);
	this.array = data;
	this.offset = 0;
	this.length = data.length;
	if (length !== undefined) {
		this.length = length;
	}
};
Go$Slice.prototype.Go$get = function(index) {
	return this.array[this.offset + index];
};
Go$Slice.prototype.Go$set = function(index, value) {
	this.array[this.offset + index] = value;
};
Go$Slice.prototype.Go$subslice = function(begin, end) {
	var s = new this.constructor(this.array);
	s.offset = this.offset + begin;
	s.length = this.length - begin;
	if (end !== undefined) {
		s.length = end - begin;
	}
	return s;
};
Go$Slice.prototype.Go$toArray = function() {
	if (this.array.constructor !== Array) {
		return this.array.subarray(this.offset, this.offset + this.length);
	}
	return this.array.slice(this.offset, this.offset + this.length);
};

String.prototype.Go$toSlice = function(terminateWithNull) {
	var array = new Uint8Array(terminateWithNull ? this.length + 1 : this.length);
	for (var i = 0; i < this.length; i++) {
		array[i] = this.charCodeAt(i);
	}
	if (terminateWithNull) {
		array[this.length] = 0;
	}
	return new Go$Slice(array);
};

var Go$clear = function(array) { for (var i = 0; i < array.length; i++) { array[i] = 0; }; return array; }; // TODO remove when NodeJS is behaving according to spec

var Go$Map = function(data, capacity) {
	data = data || [];
	for (var i = 0; i < data.length; i += 2) {
		this[data[i]] = { k: data[i], v: data[i + 1] };
	}
};

var Go$Interface = function(value) {
	return value;
};

var Go$Channel = function() {};

var Go$Pointer = function(getter, setter) { this.Go$get = getter; this.Go$set = setter; };

var Go$copy = function(dst, src) {
	var n = Math.min(src.length, dst.length);
	for (var i = 0; i < n; i++) {
		dst.Go$set(i, src.Go$get(i));
	}
	return n;
};

var Go$append = function(slice, toAppend) {
	if (slice === null || slice === 0) { // FIXME
		return toAppend;
	}
	if (toAppend === null) {
		return slice;
	}

	var newArray = slice.array;
	var newOffset = slice.offset;
	var newLength = slice.length + toAppend.length;

	if (slice.offset + newLength > newArray.length) {
		var c = slice.array.length - slice.offset;
		var newCapacity = Math.max(newLength, c < 1024 ? c * 2 : Math.floor(c * 5 / 4));
		newArray = new newArray.constructor(newCapacity);
		for (var i = 0; i < slice.length; i++) {
			newArray[i] = slice.array[slice.offset + i];
		}
		newOffset = 0;
	}

	for (var j = 0; j < toAppend.length; j++) {
		newArray[newOffset + slice.length + j] = toAppend.Go$get(j);
	}

	var newSlice = new slice.constructor(newArray);
	newSlice.offset = newOffset;
	newSlice.length = newLength;
	return newSlice;
};

var GoError = function(value) {
	this.message = value;
	Error.captureStackTrace(this, GoError);
};

var Go$errorStack = [];

// TODO inline
var Go$callDeferred = function(deferred) {
	for (var i = deferred.length - 1; i >= 0; i--) {
		var call = deferred[i];
		try {
			if (call.recv !== undefined) {
				call.recv[call.method].apply(call.recv, call.args);
				continue
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

var Go$isEqual = function(a, b) {
	if (a === null || b === null) {
		return a === null && b === null;
	}
	if (a.constructor !== b.constructor) {
		return false;
	}
	if (a.constructor.isNumber || a.constructor === Go$String) {
		return a.v === b.v;
	}
	throw new GoError("runtime error: comparing uncomparable type " + a.constructor);
};

var Go$print = console.log;
var Go$println = console.log;

var Go$typeOf = function(value) {
	if (value === null) {
		return null;
	}
	return value.constructor;
};

var typeAssertionFailed = function(obj) {
	throw new GoError("type assertion failed: " + obj + " (" + obj.constructor + ")");
};

var newNumericArray = function(len) {
	var a = new Go$Array(len);
	for (var i = 0; i < len; i++) {
		a[i] = 0;
	}
	return a;
};

var Go$now = function() { var msec = (new Date()).getTime(); return [Math.floor(msec / 1000), (msec % 1000) * 1000000]; };

var packages = {};

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
Go$Uint8.isNumber = Go$Uint16.isNumber = Go$Uint32.isNumber = Go$Int8.isNumber = Go$Int16.isNumber = Go$Int32.isNumber = Go$Float32.isNumber = Go$Float64.isNumber = Go$Complex64.isNumber = Go$Complex128.isNumber = Go$Uint.isNumber = Go$Int.isNumber = Go$Uintptr.isNumber = true;
Go$Int.Bits = Go$Uintptr.Bits = function() { return 32; };
Go$Float64.Bits = function() { return 64; };
Go$Complex128.Bits = function() { return 128; };

packages["reflect"] = {
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
				if (!this.DeepEqual(a.Go$get(i), b.Go$get(i))) {
					return false;
				}
			}
			return true;
		}
		var keys = Object.keys(a);
		for (var j = 0; j < keys.length; j++) {
			if (!this.DeepEqual(a[keys[j]], b[keys[j]])) {
				return false;
			}
		}
		return true;
	},
	TypeOf: function(v) {
		return v.constructor;
	},
	flag: function() {},
	Value: function() {}, 
};

packages["go/doc"] = {
	Synopsis: function(s) { return ""; }
};
`

var natives = map[string]string{
	"bytes": `
		IndexByte = function(s, c) {
			for (var i = 0; i < s.length; i++) {
				if (s.Go$get(i) === c) {
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
				if (a.Go$get(i) !== b.Go$get(i)) {
					return false;
				}
			}
			return true;
		}
	`,

	"math": `
		MaxFloat32 = 3.40282346638528859811704183484516925440e+38;
		SmallestNonzeroFloat32 = 1.401298464324817070923729583289916131280e-45;
		MaxFloat64 = 1.797693134862315708145274237317043567981e+308;
		SmallestNonzeroFloat64 = 4.940656458412465441765687928682213723651e-324;

		Abs = Math.abs;
		Exp = Math.exp;
		Exp2 = function(x) { return Math.exp(x * Math.log(2)); };
		// Exp2 = exp2; // TODO fix and use for higher precision
		Frexp = frexp;
		Ldexp = function(frac, exp) { return frac * Math.exp(exp * Math.log(2)); };
		// Ldexp = ldexp; // TODO fix and use for higher precision
		Log = Math.log;
		Log2 = log2;

		// generated from src/bitcasts/bitcasts.go
		var Float32bits = (function(f) {
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
				e = ((e + (1) + 4294967296) % 4294967296);
			}
			while (f < 8388608) {
				e = ((e - (1) + 4294967296) % 4294967296);
				if (e === 0) {
					break;
				}
				f = f * (2);
			}
			var y;
			return ((((((s | (((y = 23, y < 32 ? (e << y) : 0) + 4294967296) % 4294967296)) + 4294967296) % 4294967296) | ((((Math.floor(f) &~ 8388608) + 4294967296) % 4294967296))) + 4294967296) % 4294967296);
		});
		var Float32frombits = (function(b) {
			var s = 1;
			if ((((b & 2147483648) + 4294967296) % 4294967296) !== 0) {
				s = -1;
			}
			var y;
			var e = (((((((y = 23, y < 32 ? (b >>> y) : 0) + 4294967296) % 4294967296)) & 255) + 4294967296) % 4294967296);
			var m = (((b & 8388607) + 4294967296) % 4294967296);
			if (e === 255) {
				if (m === 0) {
					return s / 0;
				}
				return (s / 0) - (s / 0);
			}
			if (e !== 0) {
				m = ((m + (8388608) + 4294967296) % 4294967296);
			}
			if (e === 0) {
				e = 1;
			}
			return Ldexp(m, e - 127 - 23) * s;
		});
		var Float64bits = (function(f) {
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
				e = ((e + (1) + 4294967296) % 4294967296);
			}
			while (f < 4503599627370496) {
				e = ((e - (1) + 4294967296) % 4294967296);
				if (e === 0) {
					break;
				}
				f = f * (2);
			}
			var x, y;
			var x1, y1;
			var x2, y2;
			return (x2 = (x = s, y = Go$shift64(new Go$Uint64(0, e), 52), new Go$Uint64(x.high | y.high, x.low | y.low)), y2 = ((x1 = new Go$Uint64(0, Math.floor(f)), y1 = new Go$Uint64(1048576, 0), new Go$Uint64(x1.high &~ y1.high, x1.low &~ y1.low))), new Go$Uint64(x2.high | y2.high, x2.low | y2.low));
		});
		var Float64frombits = (function(b) {
			var s = 1;
			var x, y;
			var x1, y1;
			if ((x1 = (x = b, y = new Go$Uint64(2147483648, 0), new Go$Uint64(x.high & y.high, x.low & y.low)), y1 = new Go$Uint64(0, 0), x1.high !== y1.high || x1.low !== y1.low)) {
				s = -1;
			}
			var x2, y2;
			var e = (x2 = (Go$shift64(b, -52)), y2 = new Go$Uint64(0, 2047), new Go$Uint64(x2.high & y2.high, x2.low & y2.low));
			var x3, y3;
			var m = (x3 = b, y3 = new Go$Uint64(1048575, 4294967295), new Go$Uint64(x3.high & y3.high, x3.low & y3.low));
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
		Args = new Go$Slice(process.argv.slice(1));
	`,

	"runtime": `
		sizeof_C_MStats = 3696;
		getgoroot = function() { return process.env["GOROOT"] || ""; };
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
		var syscall = require("./node-syscall/build/Release/syscall");
		Syscall = syscall.Syscall;
		Syscall6 = syscall.Syscall6;
		BytePtrFromString = function(s) { return [s.Go$toSlice(true).array, null]; };
		Getenv = function(key) {
			var value = process.env[key];
			if (value === undefined) {
				return ["", false];
			}
			return [value, true];
		};
	`,

	"time": `
		now = Go$now;
	`,
}
