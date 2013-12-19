package translator

var Prelude = `
Error.stackTraceLimit = -1;

var go$obj, go$tuple;
var go$idCounter = 1;
var go$keys = function(m) { return m ? Object.keys(m) : []; };
var go$min = Math.min;
var go$throwRuntimeError, go$reflect, go$newStringPointer;

var go$mapArray = function(array, f) {
	var newArray = new array.constructor(array.length), i;
	for (i = 0; i < array.length; i += 1) {
		newArray[i] = f(array[i]);
	}
	return newArray;
};

var go$cache = function(v) {
	return function() {
		if (v.constructor === Function) {
			v = v();
		}
		return v;
	};
};

var go$newType = function(name, kind) {
	var typ;
	switch(kind) {
	case "Bool":
	case "Int":
	case "Int8":
	case "Int16":
	case "Int32":
	case "Uint":
	case "Uint8" :
	case "Uint16":
	case "Uint32":
	case "Uintptr":
	case "Float32":
	case "Float64":
	case "String":
	case "UnsafePointer":
		typ = function(v) { this.go$val = v; };
		typ.go$kind = kind;
		typ.prototype.go$key = function() { return name + "$" + this.go$val; };
		break;

	case "Int64":
		typ = function(high, low) {
			this.high = (high + Math.floor(Math.ceil(low) / 4294967296)) >> 0;
			this.low = low >>> 0;
			this.go$val = this;
		};
		typ.prototype.go$key = function() { return name + "$" + this.high + "$" + this.low; };
		break;

	case "Uint64":
		typ = function(high, low) {
			this.high = (high + Math.floor(Math.ceil(low) / 4294967296)) >>> 0;
			this.low = low >>> 0;
			this.go$val = this;
		};
		typ.prototype.go$key = function() { return name + "$" + this.high + "$" + this.low; };
		break

	case "Complex64":
	case "Complex128":
		typ = function(real, imag) {
			this.real = real;
			this.imag = imag;
			this.go$val = this;
		};
		typ.prototype.go$key = function() { return name + "$" + this.real + "$" + this.imag; };
		break;

	case "Array":
		typ = function(v) { this.go$val = v; };
		typ.init = function(elem, len) {
			typ.reflectType = go$cache(function() {
				var rt = new go$reflect.rtype(0, 0, 0, 0, 0, go$reflect.kinds.Array, typ, undefined, go$newStringPointer(name), undefined, undefined);
				rt.arrayType = new go$reflect.arrayType(rt, elem.reflectType(), undefined, len);
				return rt;
			});
		};
		break;

	case "Func":
		typ = function(v) { this.go$val = v; };
		typ.init = function(params, results, isVariadic) {
			typ.reflectType = go$cache(function() {
				var rt = new go$reflect.rtype(0, 0, 0, 0, 0, go$reflect.kinds.Func, typ, undefined, go$newStringPointer(name), undefined, undefined);
				var typeSlice = (go$sliceType(go$ptrType(go$reflect.rtype)));
				rt.funcType = new go$reflect.funcType(rt, isVariadic, new typeSlice(go$mapArray(params, function(p) { return p.reflectType(); })), new typeSlice(go$mapArray(results, function(p) { return p.reflectType(); })));
				return rt;
			});
		};
		typ.prototype.go$uncomparable = true;
		break;

	case "Interface":
		typ = { go$implementedBy: [] };
		break;

	case "Map":
		typ = function(v) { this.go$val = v; };
		typ.init = function(key, elem) {
			typ.reflectType = go$cache(function() {
				var rt = new go$reflect.rtype(0, 0, 0, 0, 0, go$reflect.kinds.Map, typ, undefined, go$newStringPointer(name), undefined, undefined);
				rt.mapType = new go$reflect.mapType(rt, key.reflectType(), elem.reflectType(), undefined, undefined);
				return rt;
			});
		};
		typ.prototype.go$uncomparable = true;
		break;

	case "Ptr":
		typ = function(getter, setter) {
			this.go$get = getter;
			this.go$set = setter;
			this.go$val = this;
		};
		typ.init = function(elem) {
			typ.nil = new typ(go$throwNilPointerError, go$throwNilPointerError);
			typ.reflectType = go$cache(function() {
				var rt = new go$reflect.rtype(0, 0, 0, 0, 0, go$reflect.kinds.Ptr, typ, undefined, go$newStringPointer(name), undefined, undefined);
				rt.ptrType = new go$reflect.ptrType(rt, elem.reflectType());
				return rt;
			});
		}
		break;

	case "Slice":
		var nativeArray;
		typ = function(array) {
			if (array.constructor !== nativeArray) {
				array = new nativeArray(array);
			}
			this.array = array;
			this.offset = 0;
			this.length = array.length;
			this.capacity = array.length;
			this.go$val = this;
		};
		typ.make = function(length, capacity, zero) {
			capacity = capacity || length;
			var array = new nativeArray(capacity), i;
			for (i = 0; i < capacity; i += 1) {
				array[i] = zero();
			}
			var slice = new typ(array);
			slice.length = length;
			return slice;
		};
		typ.init = function(elem) {
			nativeArray = go$nativeArray(elem.go$kind);
			typ.nil = new typ([]);
			typ.reflectType = go$cache(function() {
				var rt = new go$reflect.rtype(0, 0, 0, 0, 0, go$reflect.kinds.Slice, typ, undefined, go$newStringPointer(name), undefined, undefined);
				rt.sliceType = new go$reflect.sliceType(rt, elem.reflectType());
				return rt;
			});
		};
		typ.prototype.go$uncomparable = true;
		break;

	default:
		throw new Go$Panic("invalid kind: " + kind);
	}
	typ.string = name;
	typ.reflectType = go$cache(function() {
		var size = ({ Int: 4, Int8: 1, Int16: 2, Int32: 4, Int64: 8, Uint: 4, Uint8: 1, Uint16: 2, Uint32: 4, Uint64: 8, Uintptr: 4, Float32: 4, Float64: 8, UnsafePointer: 4 })[kind] || 0;
		return new go$reflect.rtype(size, 0, 0, 0, 0, go$reflect.kinds[kind], typ, undefined, go$newStringPointer(name), undefined, undefined);
	});
	return typ;
};

var Go$Bool          = go$newType("bool",           "Bool");
var Go$Int           = go$newType("int",            "Int");
var Go$Int8          = go$newType("int8",           "Int8");
var Go$Int16         = go$newType("int16",          "Int16");
var Go$Int32         = go$newType("int32",          "Int32");
var Go$Int64         = go$newType("int64",          "Int64");
var Go$Uint          = go$newType("uint",           "Uint");
var Go$Uint8         = go$newType("uint8",          "Uint8" );
var Go$Uint16        = go$newType("uint16",         "Uint16");
var Go$Uint32        = go$newType("uint32",         "Uint32");
var Go$Uint64        = go$newType("uint64",         "Uint64");
var Go$Uintptr       = go$newType("uintptr",        "Uintptr");
var Go$Float32       = go$newType("float32",        "Float32");
var Go$Float64       = go$newType("float64",        "Float64");
var Go$Complex64     = go$newType("complex64",      "Complex64");
var Go$Complex128    = go$newType("complex128",     "Complex128");
var Go$String        = go$newType("string",         "String");
var Go$UnsafePointer = go$newType("unsafe.Pointer", "UnsafePointer");

var go$nativeArray = function(elemKind) {
	return ({ Int: Int32Array, Int8: Int8Array, Int16: Int16Array, Int32: Int32Array, Uint: Uint32Array, Uint8: Uint8Array, Uint16: Uint16Array, Uint32: Uint32Array, Uintptr: Uint32Array, Float32: Float32Array, Float64: Float64Array })[elemKind] || Array;
};
var go$toNativeArray = function(elemKind, array) {
	var nativeArray = go$nativeArray(elemKind);
	if (nativeArray === Array) {
		return array;
	}
	return new nativeArray(array);
};
var go$makeNativeArray = function(elemKind, length, zero) {
	var array = new (go$nativeArray(elemKind))(length), i;
	for (i = 0; i < length; i += 1) {
		array[i] = zero();
	}
	return array;
};
var go$arrayTypes = {};
var go$arrayType = function(elem, len) {
	var name = "[" + len + "]" + elem.string;
	var typ = go$arrayTypes[name];
	if (typ === undefined) {
		typ = go$newType(name, "Array");
		typ.init(elem, len);
		go$arrayTypes[name] = typ;
	}
	return typ;
};

var Go$Channel = function() {};

var go$funcTypes = {};
var go$funcType = function(params, results, isVariadic) {
	var paramTypes = go$mapArray(params, function(p) { return p.string; });
	if (isVariadic) {
		paramTypes[paramTypes.length - 1] = "..." + paramTypes[paramTypes.length - 1].substr(2);
	}
	var name = "func(" + paramTypes.join(", ") + ")";
	if (results.length === 1) {
		name += " " + results[0].string;
	} else if (results.length > 1) {
		name += " (" + go$mapArray(results, function(r) { return r.string; }).join(", ") + ")"
	}
	var typ = go$funcTypes[name];
	if (typ === undefined) {
		typ = go$newType(name, "Func");
		typ.init(params, results, isVariadic);
    go$funcTypes[name] = typ;
	}
	return typ;
};

var Go$Interface = function() {};
Go$Interface.string = "interface{}";
Go$Interface.nil = { go$key: function() { return "nil"; } };

var Go$Map = function() {};
(function() {
	var names = Object.getOwnPropertyNames(Object.prototype), i;
	for (i = 0; i < names.length; i += 1) {
		Go$Map.prototype[names[i]] = undefined;
	}
})();
var go$mapTypes = {};
var go$mapType = function(key, elem) {
  var name = "map[" + key.string + "]" + elem.string;
  var typ = go$mapTypes[name];
  if (typ === undefined) {
  	typ = go$newType(name, "Map");
  	typ.init(key, elem);
    go$mapTypes[name] = typ;
  }
  return typ;
};

var go$throwNilPointerError = function() { go$throwRuntimeError("invalid memory address or nil pointer dereference"); };
var go$ptrType = function(elem) {
	var typ = elem.Go$Pointer;
	if (typ === undefined) {
		typ = go$newType("*" + elem.string, "Ptr");
		typ.init(elem);
		elem.Go$Pointer = typ;
	}
	return typ;
};

var go$sliceType = function(elem) {
	var typ = elem.Go$Slice;
	if (typ === undefined) {
		typ = go$newType("[]" + elem.string, "Slice");
		typ.init(elem);
		elem.Go$Slice = typ;
	}
	return typ;
};

var Go$Struct = function() {};

var Go$StringPointer = go$ptrType(Go$String);
go$newStringPointer = function(str) {
	return new Go$StringPointer(function() { return str; }, function(v) { str = v; });
};
var go$newDataPointer = function(data, constructor) {
	return new constructor(function() { return data; }, function(v) { data = v; });
};

var go$structNil = function(constructor) {
	var nil = new constructor();
	var fields = Object.keys(nil), i;
	for (i = 0; i < fields.length; i++) {
		var field = fields[i];
		if (field !== "go$id" && field !== "go$val") {
			Object.defineProperty(nil, field, { get: go$throwNilPointerError, set: go$throwNilPointerError });
		}
	}
	return nil;
};

var go$flatten64 = function(x) {
	return x.high * 4294967296 + x.low;
};
var go$shiftLeft64 = function(x, y) {
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
var go$shiftRightInt64 = function(x, y) {
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
var go$shiftRightUint64 = function(x, y) {
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
var go$mul64 = function(x, y) {
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
var go$div64 = function(x, y, returnRemainder) {
	if (y.high === 0 && y.low === 0) {
		go$throwRuntimeError("integer divide by zero");
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

var go$divComplex = function(n, d) {
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

var go$subslice = function(slice, low, high, max) {
	if (low < 0 || high < low || max < high || high > slice.capacity || max > slice.capacity) {
		go$throwRuntimeError("slice bounds out of range");
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

var go$sliceToArray = function(slice) {
	if (slice.length === 0) {
		return [];
	}
	if (slice.array.constructor !== Array) {
		return slice.array.subarray(slice.offset, slice.offset + slice.length);
	}
	return slice.array.slice(slice.offset, slice.offset + slice.length);
};

var go$decodeRune = function(str, pos) {
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

var go$encodeRune = function(r) {
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

var go$stringToBytes = function(str, terminateWithNull) {
	var array = new Uint8Array(terminateWithNull ? str.length + 1 : str.length), i;
	for (i = 0; i < str.length; i += 1) {
		array[i] = str.charCodeAt(i);
	}
	if (terminateWithNull) {
		array[str.length] = 0;
	}
	return array;
};

var go$bytesToString = function(slice) {
	if (slice.length === 0) {
		return "";
	}
	var str = "", i;
	for (i = 0; i < slice.length; i += 10000) {
		str += String.fromCharCode.apply(null, slice.array.subarray(slice.offset + i, slice.offset + Math.min(slice.length, i + 10000)));
	}
	return str;
};

var go$stringToRunes = function(str) {
	var array = new Int32Array(str.length);
	var rune, i, j = 0;
	for (i = 0; i < str.length; i += rune[1], j += 1) {
		rune = go$decodeRune(str, i);
		array[j] = rune[0];
	}
	return array.subarray(0, j);
};

var go$runesToString = function(slice) {
	if (slice.length === 0) {
		return "";
	}
	var str = "", i;
	for (i = 0; i < slice.length; i += 1) {
		str += go$encodeRune(slice.array[slice.offset + i]);
	}
	return str;
};

var go$externalizeString = function(intStr) {
	var extStr = "", rune, i, j = 0;
	for (i = 0; i < intStr.length; i += rune[1], j += 1) {
		rune = go$decodeRune(intStr, i);
		extStr += String.fromCharCode(rune[0]);
	}
	return extStr;
};

var go$internalizeString = function(extStr) {
	var intStr = "", i;
	for (i = 0; i < extStr.length; i += 1) {
		intStr += go$encodeRune(extStr.charCodeAt(i));
	}
	return intStr;
};

var go$copySlice = function(dst, src) {
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

var go$copyString = function(dst, src) {
	var n = Math.min(src.length, dst.length), i;
	for (i = 0; i < n; i += 1) {
		dst.array[dst.offset + i] = src.charCodeAt(i);
	}
	return n;
};

var go$copyArray = function(dst, src) {
	var i;
	for (i = 0; i < src.length; i += 1) {
		dst[i] = src[i];
	}
};

var go$append = function(slice, toAppend) {
	if (toAppend.length === 0) {
		return slice;
	}

	var newArray = slice.array;
	var newOffset = slice.offset;
	var newLength = slice.length + toAppend.length;
	var newCapacity = slice.capacity;

	if (newLength > newCapacity) {
		newCapacity = Math.max(newLength, newCapacity < 1024 ? newCapacity * 2 : Math.floor(newCapacity * 5 / 4));

		if (newArray.constructor === Array) {
			if (newOffset !== 0 || newArray.length !== newOffset + slice.capacity) {
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

var go$error = {};
var Go$Panic = function(value) {
	this.value = value;
	if (value.constructor === Go$String) {
		this.message = value.go$val;
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
var go$wrapJavaScriptError = function(err) {
	if (err.constructor === Go$Exit) {
		throw err;
	}
	var panic = new Go$Panic(err);
	panic.stack = err.stack;
	return panic;
};

var go$errorStack = [];

// TODO inline
var go$callDeferred = function(deferred) {
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
			go$errorStack.push({ frame: go$getStackDepth(), error: err });
		}
	}
	var err = go$errorStack[go$errorStack.length - 1];
	if (err !== undefined && err.frame === go$getStackDepth()) {
		go$errorStack.pop();
		throw err.error;
	}
};

var go$recover = function() {
	var err = go$errorStack[go$errorStack.length - 1];
	if (err === undefined || err.frame !== go$getStackDepth() - 2) {
		return null;
	}
	go$errorStack.pop();
	return err.error.value;
};

var go$getStack = function() {
	return (new Error()).stack.split("\n");
};

var go$getStackDepth = function() {
	var s = go$getStack(), d = 0, i;
	for (i = 0; i < s.length; i += 1) {
		if (s[i].indexOf("go$callDeferred") == -1) {
			d += 1;
		}
	}
	return d;
};

var go$interfaceIsEqual = function(a, b) {
	if (a === null || b === null) {
		return a === null && b === null;
	}
	if (a.constructor !== b.constructor) {
		return false;
	}
	if (a.go$uncomparable) {
		throw new Go$Panic("runtime error: comparing uncomparable type " + a.constructor);
	}
	return a.go$val === b.go$val;
};
var go$arrayIsEqual = function(a, b) {
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
var go$sliceIsEqual = function(a, ai, b, bi) {
	return a.array === b.array && a.offset + ai === b.offset + bi;
};
var go$pointerIsEqual = function(a, b) {
	if (a === b) {
		return true;
	}
	if (a.go$get === go$throwNilPointerError || b.go$get === go$throwNilPointerError) {
		return a.go$get === go$throwNilPointerError && b.go$get === go$throwNilPointerError;
	}
	var old = a.go$get();
	var dummy = new Object();
	a.go$set(dummy);
	var equal = b.go$get() === dummy;
	a.go$set(old);
	return equal;
};

var go$typeAssertionFailed = function(obj) {
	throw new Go$Panic("type assertion failed: " + obj + " (" + obj.constructor + ")");
};

var go$now = function() { var msec = (new Date()).getTime(); return [new Go$Int64(0, Math.floor(msec / 1000)), (msec % 1000) * 1000000]; };

var go$packages = {};

go$packages["go/doc"] = {
	Synopsis: function(s) { return ""; }
};
`
