Error.stackTraceLimit = Infinity;

var $NaN = NaN;
var $global, $module;
if (typeof window !== "undefined") { /* web page */
    $global = window;
} else if (typeof self !== "undefined") { /* web worker */
    $global = self;
} else if (typeof global !== "undefined") { /* Node.js */
    $global = global;
    $global.require = require;
} else { /* others (e.g. Nashorn) */
    $global = this;
}

if ($global === undefined || $global.Array === undefined) {
    throw new Error("no global object found");
}
if (typeof module !== "undefined") {
    $module = module;
}

if (!$global.fs && $global.require) {
    try {
        var fs = $global.require('fs');
        if (typeof fs === "object" && fs !== null && Object.keys(fs).length !== 0) {
            $global.fs = fs;
        }
    } catch (e) { /* Ignore if the module couldn't be loaded. */ }
}

if (!$global.fs) {
    var outputBuf = "";
    var decoder = new TextDecoder("utf-8");
    $global.fs = {
        constants: { O_WRONLY: -1, O_RDWR: -1, O_CREAT: -1, O_TRUNC: -1, O_APPEND: -1, O_EXCL: -1 }, // unused
        writeSync: function writeSync(fd, buf) {
            outputBuf += decoder.decode(buf);
            var nl = outputBuf.lastIndexOf("\n");
            if (nl != -1) {
                console.log(outputBuf.substr(0, nl));
                outputBuf = outputBuf.substr(nl + 1);
            }
            return buf.length;
        },
        write: function write(fd, buf, offset, length, position, callback) {
            if (offset !== 0 || length !== buf.length || position !== null) {
                callback(enosys());
                return;
            }
            var n = this.writeSync(fd, buf);
            callback(null, n);
        }
    };
}

var $linknames = {} // Collection of functions referenced by a go:linkname directive.
var $packages = {}, $idCounter = 0;
var $keys = m => { return m ? Object.keys(m) : []; };
var $flushConsole = () => { };
var $throwRuntimeError; /* set by package "runtime" */
var $throwNilPointerError = () => { $throwRuntimeError("invalid memory address or nil pointer dereference"); };
var $call = (fn, rcvr, args) => { return fn.apply(rcvr, args); };
var $makeFunc = fn => { return function(...args) { return $externalize(fn(this, new ($sliceType($jsObjectPtr))($global.Array.prototype.slice.call(args, []))), $emptyInterface); }; };
var $unused = v => { };
var $print = console.log;
// Under Node we can emulate print() more closely by avoiding a newline.
if (($global.process !== undefined) && $global.require) {
    try {
        var util = $global.require('util');
        $print = function(...args) { $global.process.stderr.write(util.format.apply(this, args)); };
    } catch (e) {
        // Failed to require util module, keep using console.log().
    }
}
var $println = console.log

var $initAllLinknames = () => {
    var names = $keys($packages);
    for (var i = 0; i < names.length; i++) {
        var f = $packages[names[i]]["$initLinknames"];
        if (typeof f == 'function') {
            f();
        }
    }
}

var $mapArray = (array, f) => {
    var newArray = new array.constructor(array.length);
    for (var i = 0; i < array.length; i++) {
        newArray[i] = f(array[i]);
    }
    return newArray;
};

// $mapIndex returns the value of the given key in m, or undefined if m is nil/undefined or not a map
var $mapIndex = (m, key) => {
    return typeof m.get === "function" ? m.get(key) : undefined;
};
// $mapDelete deletes the key and associated value from m.  If m is nil/undefined or not a map, $mapDelete is a no-op
var $mapDelete = (m, key) => {
    typeof m.delete === "function" && m.delete(key)
};
// Returns a method bound to the receiver instance, safe to invoke as a 
// standalone function. Bound function is cached for later reuse.
var $methodVal = (recv, name) => {
    var vals = recv.$methodVals || {};
    recv.$methodVals = vals; /* noop for primitives */
    var f = vals[name];
    if (f !== undefined) {
        return f;
    }
    var method = recv[name];
    f = method.bind(recv);
    vals[name] = f;
    return f;
};

var $methodExpr = (typ, name) => {
    var method = typ.prototype[name];
    if (method.$expr === undefined) {
        method.$expr = (...args) => {
            $stackDepthOffset--;
            try {
                if (typ.wrapped) {
                    args[0] = new typ(args[0]);
                }
                return Function.call.apply(method, args);
            } finally {
                $stackDepthOffset++;
            }
        };
    }
    return method.$expr;
};

var $ifaceMethodExprs = {};
var $ifaceMethodExpr = name => {
    var expr = $ifaceMethodExprs["$" + name];
    if (expr === undefined) {
        expr = $ifaceMethodExprs["$" + name] = (...args) => {
            $stackDepthOffset--;
            try {
                return Function.call.apply(args[0][name], args);
            } finally {
                $stackDepthOffset++;
            }
        };
    }
    return expr;
};

var $subslice = (slice, low, high, max) => {
    if (high === undefined) {
        high = slice.$length;
    }
    if (max === undefined) {
        max = slice.$capacity;
    }
    if (low < 0 || high < low || max < high || high > slice.$capacity || max > slice.$capacity) {
        $throwRuntimeError("slice bounds out of range");
    }
    if (slice === slice.constructor.nil) {
        return slice;
    }
    var s = new slice.constructor(slice.$array);
    s.$offset = slice.$offset + low;
    s.$length = high - low;
    s.$capacity = max - low;
    return s;
};

var $substring = (str, low, high) => {
    if (low < 0 || high < low || high > str.length) {
        $throwRuntimeError("slice bounds out of range");
    }
    return str.substring(low, high);
};

// Convert Go slice to an equivalent JS array type.
var $sliceToNativeArray = slice => {
    if (slice.$array.constructor !== Array) {
        return slice.$array.subarray(slice.$offset, slice.$offset + slice.$length);
    }
    return slice.$array.slice(slice.$offset, slice.$offset + slice.$length);
};

// Convert Go slice to a pointer to an underlying Go array.
// 
// Note that an array pointer can be represented by an "unwrapped" native array
// type, and it will be wrapped back into its Go type when necessary.
var $sliceToGoArray = (slice, arrayPtrType) => {
    var arrayType = arrayPtrType.elem;
    if (arrayType !== undefined && slice.$length < arrayType.len) {
        $throwRuntimeError("cannot convert slice with length " + slice.$length + " to pointer to array with length " + arrayType.len);
    }
    if (slice == slice.constructor.nil) {
        return arrayPtrType.nil; // Nil slice converts to nil array pointer.
    }
    if (slice.$array.constructor !== Array) {
        return slice.$array.subarray(slice.$offset, slice.$offset + arrayType.len);
    }
    if (slice.$offset == 0 && slice.$length == slice.$capacity && slice.$length == arrayType.len) {
        return slice.$array;
    }
    if (arrayType.len == 0) {
        return new arrayType([]);
    }

    // Array.slice (unlike TypedArray.subarray) returns a copy of an array range,
    // which is not sharing memory with the original one, which violates the spec
    // for slice to array conversion. This is incompatible with the Go spec, in
    // particular that the assignments to the array elements would be visible in
    // the slice. Prefer to fail explicitly instead of creating subtle bugs.
    $throwRuntimeError("gopherjs: non-numeric slice to underlying array conversion is not supported for subslices");
};

// Convert between compatible slice types (e.g. native and names).
var $convertSliceType = (slice, desiredType) => {
    if (slice == slice.constructor.nil) {
        return desiredType.nil; // Preserve nil value.
    }

    return $subslice(new desiredType(slice.$array), slice.$offset, slice.$offset + slice.$length);
}

var $decodeRune = (str, pos) => {
    var c0 = str.charCodeAt(pos);

    if (c0 < 0x80) {
        return [c0, 1];
    }

    if (c0 !== c0 || c0 < 0xC0) {
        return [0xFFFD, 1];
    }

    var c1 = str.charCodeAt(pos + 1);
    if (c1 !== c1 || c1 < 0x80 || 0xC0 <= c1) {
        return [0xFFFD, 1];
    }

    if (c0 < 0xE0) {
        var r = (c0 & 0x1F) << 6 | (c1 & 0x3F);
        if (r <= 0x7F) {
            return [0xFFFD, 1];
        }
        return [r, 2];
    }

    var c2 = str.charCodeAt(pos + 2);
    if (c2 !== c2 || c2 < 0x80 || 0xC0 <= c2) {
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

    var c3 = str.charCodeAt(pos + 3);
    if (c3 !== c3 || c3 < 0x80 || 0xC0 <= c3) {
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

var $encodeRune = r => {
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

var $stringToBytes = str => {
    var array = new Uint8Array(str.length);
    for (var i = 0; i < str.length; i++) {
        array[i] = str.charCodeAt(i);
    }
    return array;
};

var $bytesToString = slice => {
    if (slice.$length === 0) {
        return "";
    }
    var str = "";
    for (var i = 0; i < slice.$length; i += 10000) {
        str += String.fromCharCode.apply(undefined, slice.$array.subarray(slice.$offset + i, slice.$offset + Math.min(slice.$length, i + 10000)));
    }
    return str;
};

var $stringToRunes = str => {
    var array = new Int32Array(str.length);
    var rune, j = 0;
    for (var i = 0; i < str.length; i += rune[1], j++) {
        rune = $decodeRune(str, i);
        array[j] = rune[0];
    }
    return array.subarray(0, j);
};

var $runesToString = slice => {
    if (slice.$length === 0) {
        return "";
    }
    var str = "";
    for (var i = 0; i < slice.$length; i++) {
        str += $encodeRune(slice.$array[slice.$offset + i]);
    }
    return str;
};

var $copyString = (dst, src) => {
    var n = Math.min(src.length, dst.$length);
    for (var i = 0; i < n; i++) {
        dst.$array[dst.$offset + i] = src.charCodeAt(i);
    }
    return n;
};

var $copySlice = (dst, src) => {
    var n = Math.min(src.$length, dst.$length);
    $copyArray(dst.$array, src.$array, dst.$offset, src.$offset, n, dst.constructor.elem);
    return n;
};

var $copyArray = (dst, src, dstOffset, srcOffset, n, elem) => {
    if (n === 0 || (dst === src && dstOffset === srcOffset)) {
        return;
    }

    if (src.subarray) {
        dst.set(src.subarray(srcOffset, srcOffset + n), dstOffset);
        return;
    }

    switch (elem.kind) {
        case $kindArray:
        case $kindStruct:
            if (dst === src && dstOffset > srcOffset) {
                for (var i = n - 1; i >= 0; i--) {
                    elem.copy(dst[dstOffset + i], src[srcOffset + i]);
                }
                return;
            }
            for (var i = 0; i < n; i++) {
                elem.copy(dst[dstOffset + i], src[srcOffset + i]);
            }
            return;
    }

    if (dst === src && dstOffset > srcOffset) {
        for (var i = n - 1; i >= 0; i--) {
            dst[dstOffset + i] = src[srcOffset + i];
        }
        return;
    }
    for (var i = 0; i < n; i++) {
        dst[dstOffset + i] = src[srcOffset + i];
    }
};

var $clone = (src, type) => {
    var clone = type.zero();
    type.copy(clone, src);
    return clone;
};

var $pointerOfStructConversion = (obj, type) => {
    if (obj.$proxies === undefined) {
        obj.$proxies = {};
        obj.$proxies[obj.constructor.string] = obj;
    }
    var proxy = obj.$proxies[type.string];
    if (proxy === undefined) {
        var properties = {};
        for (var i = 0; i < type.elem.fields.length; i++) {
            (fieldProp => {
                properties[fieldProp] = {
                    get() { return obj[fieldProp]; },
                    set(value) { obj[fieldProp] = value; }
                };
            })(type.elem.fields[i].prop);
        }
        proxy = Object.create(type.prototype, properties);
        proxy.$val = proxy;
        obj.$proxies[type.string] = proxy;
        proxy.$proxies = obj.$proxies;
    }
    return proxy;
};

var $append = function (slice) {
    return $internalAppend(slice, arguments, 1, arguments.length - 1);
};

var $appendSlice = (slice, toAppend) => {
    if (toAppend.constructor === String) {
        var bytes = $stringToBytes(toAppend);
        return $internalAppend(slice, bytes, 0, bytes.length);
    }
    return $internalAppend(slice, toAppend.$array, toAppend.$offset, toAppend.$length);
};

var $internalAppend = (slice, array, offset, length) => {
    if (length === 0) {
        return slice;
    }

    var newArray = slice.$array;
    var newOffset = slice.$offset;
    var newLength = slice.$length + length;
    var newCapacity = slice.$capacity;

    if (newLength > newCapacity) {
        newOffset = 0;
        newCapacity = Math.max(newLength, slice.$capacity < 1024 ? slice.$capacity * 2 : Math.floor(slice.$capacity * 5 / 4));

        if (slice.$array.constructor === Array) {
            newArray = slice.$array.slice(slice.$offset, slice.$offset + slice.$length);
            newArray.length = newCapacity;
            var zero = slice.constructor.elem.zero;
            for (var i = slice.$length; i < newCapacity; i++) {
                newArray[i] = zero();
            }
        } else {
            newArray = new slice.$array.constructor(newCapacity);
            newArray.set(slice.$array.subarray(slice.$offset, slice.$offset + slice.$length));
        }
    }

    $copyArray(newArray, array, newOffset + slice.$length, offset, length, slice.constructor.elem);

    var newSlice = new slice.constructor(newArray);
    newSlice.$offset = newOffset;
    newSlice.$length = newLength;
    newSlice.$capacity = newCapacity;
    return newSlice;
};

var $equal = (a, b, type) => {
    if (type === $jsObjectPtr) {
        return a === b;
    }
    switch (type.kind) {
        case $kindComplex64:
        case $kindComplex128:
            return a.$real === b.$real && a.$imag === b.$imag;
        case $kindInt64:
        case $kindUint64:
            return a.$high === b.$high && a.$low === b.$low;
        case $kindArray:
            if (a.length !== b.length) {
                return false;
            }
            for (var i = 0; i < a.length; i++) {
                if (!$equal(a[i], b[i], type.elem)) {
                    return false;
                }
            }
            return true;
        case $kindStruct:
            for (var i = 0; i < type.fields.length; i++) {
                var f = type.fields[i];
                if (!$equal(a[f.prop], b[f.prop], f.typ)) {
                    return false;
                }
            }
            return true;
        case $kindInterface:
            return $interfaceIsEqual(a, b);
        default:
            return a === b;
    }
};

var $interfaceIsEqual = (a, b) => {
    if (a === $ifaceNil || b === $ifaceNil) {
        return a === b;
    }
    if (a.constructor !== b.constructor) {
        return false;
    }
    if (a.constructor === $jsObjectPtr) {
        return a.object === b.object;
    }
    if (!a.constructor.comparable) {
        $throwRuntimeError("comparing uncomparable type " + a.constructor.string);
    }
    return $equal(a.$val, b.$val, a.constructor);
};

var $unsafeMethodToFunction = (typ, name, isPtr) => {
    if (isPtr) {
        return (r, ...args) => {
            var ptrType = $ptrType(typ);
            if (r.constructor != ptrType) {
                switch (typ.kind) {
                    case $kindStruct:
                        r = $pointerOfStructConversion(r, ptrType);
                        break;
                    case $kindArray:
                        r = new ptrType(r);
                        break;
                    default:
                        r = new ptrType(r.$get, r.$set, r.$target);
                }
            }
            return r[name](...args);
        };
    } else {
        return (r, ...args) => {
            var ptrType = $ptrType(typ);
            if (r.constructor != ptrType) {
                switch (typ.kind) {
                    case $kindStruct:
                        r = $clone(r, typ);
                        break;
                    case $kindSlice:
                        r = $convertSliceType(r, typ);
                        break;
                    case $kindComplex64:
                    case $kindComplex128:
                        r = new typ(r.$real, r.$imag);
                        break;
                    default:
                        r = new typ(r);
                }
            }
            return r[name](...args);
        };
    }
};

var $id = x => {
    return x;
};

var $instanceOf = (x, y) => {
    return x instanceof y;
};

var $typeOf = x => {
    return typeof (x);
};

// The builtin unsafe.SliceData returns nil if the slice is nil, otherwise
// returns a pointer to the first index of the array, &s[0].
// If the slice is empty (cap == 0), it returns an "unspecified memory address"
// or for JS, a pointer to the empty array.
var $sliceData = (slice, typ) => {
    if (slice === typ.nil) {
        return $ptrType(typ.elem).nil;
    }
    return $indexPtr(slice.$array, slice.$offset, typ.elem);
};
