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

var Go$Array = Array;
var Go$Boolean = Boolean;
var Go$Function = Function;
var Go$keys = Object.keys;

var Go$copyFields = function(from, to) {
	var keys = Object.keys(from);
	for (var i = 0; i < keys.length; i++) {
		var key = keys[i];
		to[key] = from[key];
	}
};

var Go$Int64 = function(high, low) {
	this.high = (high + Math.floor(low / 4294967296) + 4294967296) | 0;
	this.low = (low + 4294967296) % 4294967296;
};
var Go$Uint64 = function(high, low) {
	this.high = (high + Math.floor(low / 4294967296) + 4294967296) % 4294967296;
	this.low = (low + 4294967296) % 4294967296;
};
var Go$shift64 = function(x, y) {
	var p = Math.pow(2, y);
	var high = Math.floor((x.high * p % 4294967296) + (x.low  / 4294967296 * p % 4294967296));
	var low  = Math.floor((x.low  * p % 4294967296) + (x.high * 4294967296 * p % 4294967296));
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

var Go$String = String;
Go$String.prototype.Go$toSlice = function(terminateWithNull) {
	var array = new Uint8Array(terminateWithNull ? this.length + 1 : this.length);
	for (var i = 0; i < this.length; i++) {
		array[i] = this.charCodeAt(i);
	}
	if (terminateWithNull) {
		array[this.length] = 0;
	}
	return new Go$Slice(array);
};
Go$String.Kind = function() { return 24; };

var Go$clear = function(array) { for (var i = 0; i < array.length; i++) { array[i] = 0; }; return array; }; // TODO remove when NodeJS is behaving according to spec

Number.Kind = function() { return 2; };
Number.Bits = function() { return 32; };

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

// TODO improve performance by increasing capacity in bigger steps
var Go$append = function(slice, toAppend) {
	if (slice === null) {
		return toAppend;
	}

	var newArray = slice.array;
	var newOffset = slice.offset;
	var newLength = slice.length + toAppend.length;

	if (slice.offset + newLength > newArray.length) {
		newArray = new newArray.constructor(newLength);
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

var _isEqual = function(a, b) {
	if (a === null || b === null) {
		return a === null && b === null;
	}
	if (a.constructor !== b.constructor) {
		return false;
	}
	if (a.constructor === Number || a.constructor === String) {
		return a === b;
	}
	throw new Error("runtime error: comparing uncomparable type " + a.constructor);
};

var Go$print = console.log;
var Go$println = console.log;

var Go$Integer = function() {};
var Go$Float = function() {};
var Go$Complex = function() {};

var Go$typeOf = function(value) {
	if (value === null) {
		return null;
	}
	var type = value.constructor;
	if (type === Number) {
		return (Math.floor(value) === value) ? Go$Integer : Go$Float;
	}
	return type;
};

var typeAssertionFailed = function(obj) {
	throw new Error("type assertion failed: " + obj + " (" + obj.constructor + ")");
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
		for (var j = 0; j < keys.length;	j++) {
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
