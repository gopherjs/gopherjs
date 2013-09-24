package main

// TODO cleanup global names
var prelude = `
#!/usr/bin/env node

"use strict";
Error.stackTraceLimit = -1;

var _idCounter = 1;
var Go$Nil = { _id: 0 };

var Slice = function(data, length, capacity) {
	capacity = capacity || length || 0;
	data = data || new Array(capacity);
	this.array = data;
	this.offset = 0;
	this.length = data.length;
	if (length !== undefined) {
		this.length = length;
	}
};

Slice.prototype.get = function(index) {
	return this.array[this.offset + index];
};

Slice.prototype.set = function(index, value) {
	this.array[this.offset + index] = value;
};

Slice.prototype.subslice = function(begin, end) {
	var s = new this.constructor(this.array);
	s.offset = this.offset + begin;
	s.length = this.length - begin;
	if (end !== undefined) {
		s.length = end - begin;
	}
	return s;
};

Slice.prototype.toArray = function() {
	return this.array.slice(this.offset, this.offset + this.length);
};

String.prototype.toSlice = function() {
	var array = new Int32Array(this.length);
	for (var i = 0; i < this.length; i++) {
		array[i] = this.charCodeAt(i);
	}
	return new Slice(array);
};

String.Kind = function() { return 24; };

Number.Kind = function() { return 2; };
Number.Bits = function() { return 32; };

var Go$Map = function(data, capacity) {
	data = data || [];
	for (var i = 0; i < data.length; i += 2) {
		this[data[i]] = { k: data[i], v: data[i + 1] };
	}
};

var Interface = function(value) {
	return value;
};

var Channel = function() {};

var _Pointer = function(getter, setter) { this.get = getter; this.set = setter; };

var copy = function(dst, src) {
	var n = Math.min(src.length, dst.length);
	for (var i = 0; i < n; i++) {
		dst.set(i, src.get(i));
	}
	return n;
};

// TODO improve performance by increasing capacity in bigger steps
var append = function(slice, toAppend) {
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
		newArray[newOffset + slice.length + j] = toAppend.get(j);
	}

	var newSlice = new slice.constructor(newArray);
	newSlice.offset = newOffset;
	newSlice.length = newLength;
	return newSlice;
};

var GoError = function(value) {
	this.value = value;
	Error.captureStackTrace(this, GoError);
};
GoError.prototype.toString = function() {
	return this.value;
}

var _error_stack = [];

// TODO inline
var callDeferred = function(deferred) {
	for (var i = deferred.length - 1; i >= 0; i--) {
		var call = deferred[i];
		try {
			call.fun.apply(call.recv, call.args);
		} catch (err) {
			_error_stack.push({ frame: getStackDepth(), error: err });
		}
	}
	var err = _error_stack[_error_stack.length - 1];
	if (err !== undefined && err.frame === getStackDepth()) {
		_error_stack.pop();
		throw err.error;
	}
}

var recover = function() {
	var err = _error_stack[_error_stack.length - 1];
	if (err === undefined || err.frame !== getStackDepth() - 2) {
		return null;
	}
	_error_stack.pop();
	return err.error.value;
};

var getStackDepth = function() {
	var s = (new Error()).stack.split("\n");
	var d = 0;
	for (var i = 0; i < s.length; i++) {
		if (s[i].indexOf("callDeferred") == -1) {
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

var print = function(a) {
	console.log(a.toArray().join(" "));
};

var println = print;

var Integer = function() {};
var Float = function() {};
var Complex = function() {};

var typeOf = function(value) {
	var type = value.constructor;
	if (type === Number) {
		return (Math.floor(value) === value) ? Integer : Float;
	}
	return type;
};

var typeAssertionFailed = function() {
	throw new Error("type assertion failed");
};

var newNumericArray = function(len) {
	var a = new Array(len);
	for (var i = 0; i < len; i++) {
		a[i] = 0;
	}
	return a;
};

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
				if (!this.DeepEqual(a.get(i), b.get(i))) {
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
	"runtime": `
		sizeof_C_MStats = 3696;
		getgoroot = function() { return process.env["GOROOT"] || ""; };
		SetFinalizer = function() {};
	`,

	"sync/atomic": `
		AddInt32 = AddInt64 = AddUint32 = AddUint64 = AddUintptr = function(addr, delta) {
			var value = addr.get() + delta;
			addr.set(value);
			return value;
		};
		CompareAndSwapInt32 = CompareAndSwapInt64 = CompareAndSwapUint32 = CompareAndSwapUint64 = CompareAndSwapUintptr = function(addr, oldVal, newVal) {
			if (addr.get() === oldVal) {
				addr.set(newVal);
				return true;
			}
			return false;
		};
		StoreInt32 = StoreInt64 = StoreUint32 = StoreUint64 = StoreUintptr = function(addr, val) {
			addr.set(val);
		};
		LoadInt32 = LoadInt64 = LoadUint32 = LoadUint64 = LoadUintptr = function(addr) {
			return addr.get();
		};
	`,

	"syscall": `
		var syscall = require("./node-syscall/build/Release/syscall");
		Syscall = syscall.Syscall;
		Getenv = function(key) {
			var value = process.env[key];
			if (value === undefined) {
				return ["", false];
			}
			return [value, true];
		};
	`,
}
