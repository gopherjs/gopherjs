var $kindBool = 1;
var $kindInt = 2;
var $kindInt8 = 3;
var $kindInt16 = 4;
var $kindInt32 = 5;
var $kindInt64 = 6;
var $kindUint = 7;
var $kindUint8 = 8;
var $kindUint16 = 9;
var $kindUint32 = 10;
var $kindUint64 = 11;
var $kindUintptr = 12;
var $kindFloat32 = 13;
var $kindFloat64 = 14;
var $kindComplex64 = 15;
var $kindComplex128 = 16;
var $kindArray = 17;
var $kindChan = 18;
var $kindFunc = 19;
var $kindInterface = 20;
var $kindMap = 21;
var $kindPtr = 22;
var $kindSlice = 23;
var $kindString = 24;
var $kindStruct = 25;
var $kindUnsafePointer = 26;

var $methodSynthesizers = [];
var $addMethodSynthesizer = f => {
    if ($methodSynthesizers === null) {
        f();
        return;
    }
    $methodSynthesizers.push(f);
};
var $synthesizeMethods = () => {
    $methodSynthesizers.forEach(f => { f(); });
    $methodSynthesizers = null;
};

var $ifaceKeyFor = x => {
    if (x === $ifaceNil) {
        return 'nil';
    }
    var c = x.constructor;
    return c.string + '$' + c.keyFor(x.$val);
};

var $identity = x => { return x; };

var $typeIDCounter = 0;

var $idKey = x => {
    if (x.$id === undefined) {
        $idCounter++;
        x.$id = $idCounter;
    }
    return String(x.$id);
};

// Creates constructor functions for array pointer types. Returns a new function
// instance each time to make sure each type is independent of the other.
var $arrayPtrCtor = () => {
    return function (array) {
        this.$get = () => { return array; };
        this.$set = function (v) { typ.copy(this, v); };
        this.$val = array;
    };
}

var $newType = (size, kind, string, named, pkg, exported, constructor) => {
    var typ;
    switch (kind) {
        case $kindBool:
        case $kindInt:
        case $kindInt8:
        case $kindInt16:
        case $kindInt32:
        case $kindUint:
        case $kindUint8:
        case $kindUint16:
        case $kindUint32:
        case $kindUintptr:
        case $kindUnsafePointer:
            typ = function (v) { this.$val = v; };
            typ.wrapped = true;
            typ.wrap = (v) => new typ(v);
            typ.keyFor = $identity;
            break;

        case $kindString:
            typ = function (v) { this.$val = v; };
            typ.wrapped = true;
            typ.wrap = (v) => new typ(v);
            typ.keyFor = x => { return "$" + x; };
            break;

        case $kindFloat32:
        case $kindFloat64:
            typ = function (v) { this.$val = v; };
            typ.wrapped = true;
            typ.wrap = (v) => new typ(v);
            typ.keyFor = x => { return $floatKey(x); };
            break;

        case $kindInt64:
            typ = function (high, low) {
                this.$high = (high + Math.floor(Math.ceil(low) / 4294967296)) >> 0;
                this.$low = low >>> 0;
                this.$val = this;
            };
            typ.wrap = (v) => v;
            typ.keyFor = x => { return x.$high + "$" + x.$low; };
            break;

        case $kindUint64:
            typ = function (high, low) {
                this.$high = (high + Math.floor(Math.ceil(low) / 4294967296)) >>> 0;
                this.$low = low >>> 0;
                this.$val = this;
            };
            typ.wrap = (v) => v;
            typ.keyFor = x => { return x.$high + "$" + x.$low; };
            break;

        case $kindComplex64:
            typ = function (real, imag) {
                this.$real = $fround(real);
                this.$imag = $fround(imag);
                this.$val = this;
            };
            typ.wrap = (v) => v;
            typ.keyFor = x => { return x.$real + "$" + x.$imag; };
            break;

        case $kindComplex128:
            typ = function (real, imag) {
                this.$real = real;
                this.$imag = imag;
                this.$val = this;
            };
            typ.wrap = (v) => v;
            typ.keyFor = x => { return x.$real + "$" + x.$imag; };
            break;

        case $kindArray:
            typ = function (v) { this.$val = v; };
            typ.wrapped = true;
            typ.wrap = (v) => new typ(v);
            typ.ptr = $newType(4, $kindPtr, "*" + string, false, "", false, $arrayPtrCtor());
            typ.init = (elem, len) => {
                typ.elem = elem;
                typ.len = len;
                typ.comparable = elem.comparable;
                typ.keyFor = x => {
                    return Array.prototype.join.call($mapArray(x, e => {
                        return String(elem.keyFor(e)).replace(/\\/g, "\\\\").replace(/\$/g, "\\$");
                    }), "$");
                };
                typ.copy = (dst, src) => {
                    $copyArray(dst, src, 0, 0, src.length, elem);
                };
                typ.ptr.init(typ);
                Object.defineProperty(typ.ptr.nil, "nilCheck", { get: $throwNilPointerError });
            };
            break;

        case $kindChan:
            typ = function (v) { this.$val = v; };
            typ.wrapped = true;
            typ.wrap = (v) => new typ(v);
            typ.keyFor = $idKey;
            typ.init = (elem, sendOnly, recvOnly) => {
                typ.elem = elem;
                typ.sendOnly = sendOnly;
                typ.recvOnly = recvOnly;
            };
            break;

        case $kindFunc:
            typ = function (v) { this.$val = v; };
            typ.wrapped = true;
            typ.wrap = (v) => new typ(v);
            typ.init = (params, results, variadic) => {
                typ.params = params;
                typ.results = results;
                typ.variadic = variadic;
                typ.comparable = false;
            };
            break;

        case $kindInterface:
            typ = { implementedBy: {}, missingMethodFor: {} };
            typ.wrap = (v) => v;
            typ.keyFor = $ifaceKeyFor;
            typ.init = methods => {
                typ.methods = methods;
                methods.forEach(m => {
                    $ifaceNil[m.prop] = $throwNilPointerError;
                });
            };
            break;

        case $kindMap:
            typ = function (v) { this.$val = v; };
            typ.wrapped = true;
            typ.wrap = (v) => new typ(v);
            typ.init = (key, elem) => {
                typ.key = key;
                typ.elem = elem;
                typ.comparable = false;
            };
            break;

        case $kindPtr:
            typ = constructor || function (getter, setter, target) {
                this.$get = getter;
                this.$set = setter;
                this.$target = target;
                this.$val = this;
            };
            typ.wrap = (v) => v;
            typ.wrapped = false;
            typ.keyFor = $idKey;
            typ.init = elem => {
                typ.elem = elem;
                if (elem.kind === $kindArray) {
                    typ.wrapped = true;
                    typ.wrap = (v) => ((v === typ.nil) ? v : new typ(v));
                }
                typ.nil = new typ($throwNilPointerError, $throwNilPointerError);
            };
            break;

        case $kindSlice:
            typ = function (array) {
                if (array.constructor !== typ.nativeArray) {
                    array = new typ.nativeArray(array);
                }
                this.$array = array;
                this.$offset = 0;
                this.$length = array.length;
                this.$capacity = array.length;
                this.$val = this;
            };
            typ.wrap = (v) => v;
            typ.init = elem => {
                typ.elem = elem;
                typ.comparable = false;
                typ.nativeArray = $nativeArray(elem.kind);
                typ.nil = new typ([]);
            };
            break;

        case $kindStruct:
            typ = function (v) { this.$val = v; };
            typ.wrapped = true;
            typ.wrap = (v) => new typ(v);
            typ.ptr = $newType(4, $kindPtr, "*" + string, false, pkg, exported, constructor);
            if (string === "js.Object" && pkg === "github.com/gopherjs/gopherjs/js") {
                // *js.Object is a special case because unlike other pointers it
                // passes around a raw JS object without any GopherJS-specific
                // metadata. As a result, it must be wrapped to preserve type
                // information whenever it's used through an interface or type
                // param. However, it's now a "wrapped" type in a complete sense,
                // because it's handling is mostly special-cased at the compiler level.
                typ.ptr.wrap = (v) => new typ.ptr(v);
            }
            typ.ptr.elem = typ;
            typ.ptr.prototype.$get = function () { return this; };
            typ.ptr.prototype.$set = function (v) { typ.copy(this, v); };
            typ.init = (pkgPath, fields) => {
                typ.pkgPath = pkgPath;
                typ.fields = fields;
                fields.forEach(f => {
                    if (!f.typ.comparable) {
                        typ.comparable = false;
                    }
                });
                typ.keyFor = x => {
                    var val = x.$val;
                    return $mapArray(fields, f => {
                        return String(f.typ.keyFor(val[f.prop])).replace(/\\/g, "\\\\").replace(/\$/g, "\\$");
                    }).join("$");
                };
                typ.copy = (dst, src) => {
                    for (var i = 0; i < fields.length; i++) {
                        var f = fields[i];
                        switch (f.typ.kind) {
                            case $kindArray:
                            case $kindStruct:
                                f.typ.copy(dst[f.prop], src[f.prop]);
                                continue;
                            default:
                                dst[f.prop] = src[f.prop];
                                continue;
                        }
                    }
                };
                /* nil value */
                var properties = {};
                fields.forEach(f => {
                    properties[f.prop] = { get: $throwNilPointerError, set: $throwNilPointerError };
                });
                typ.ptr.nil = Object.create(constructor.prototype, properties);
                typ.ptr.nil.$val = typ.ptr.nil;
                /* methods for embedded fields */
                $addMethodSynthesizer(() => {
                    var synthesizeMethod = (target, m, f) => {
                        if (target.prototype[m.prop] !== undefined) { return; }
                        target.prototype[m.prop] = function(...args) {
                            var v = this.$val[f.prop];
                            if (f.typ === $jsObjectPtr) {
                                v = new $jsObjectPtr(v);
                            }
                            if (v.$val === undefined) {
                                v = new f.typ(v);
                            }
                            return v[m.prop](...args);
                        };
                    };
                    fields.forEach(f => {
                        if (f.embedded) {
                            $methodSet(f.typ).forEach(m => {
                                synthesizeMethod(typ, m, f);
                                synthesizeMethod(typ.ptr, m, f);
                            });
                            $methodSet($ptrType(f.typ)).forEach(m => {
                                synthesizeMethod(typ.ptr, m, f);
                            });
                        }
                    });
                });
            };
            break;

        default:
            $panic(new $String("invalid kind: " + kind));
    }

    switch (kind) {
        case $kindBool:
        case $kindMap:
            typ.zero = () => { return false; };
            break;

        case $kindInt:
        case $kindInt8:
        case $kindInt16:
        case $kindInt32:
        case $kindUint:
        case $kindUint8:
        case $kindUint16:
        case $kindUint32:
        case $kindUintptr:
        case $kindUnsafePointer:
        case $kindFloat32:
        case $kindFloat64:
            typ.zero = () => { return 0; };
            break;

        case $kindString:
            typ.zero = () => { return ""; };
            break;

        case $kindInt64:
        case $kindUint64:
        case $kindComplex64:
        case $kindComplex128:
            var zero = new typ(0, 0);
            typ.zero = () => { return zero; };
            break;

        case $kindPtr:
        case $kindSlice:
            typ.zero = () => { return typ.nil; };
            break;

        case $kindChan:
            typ.zero = () => { return $chanNil; };
            break;

        case $kindFunc:
            typ.zero = () => { return $throwNilPointerError; };
            break;

        case $kindInterface:
            typ.zero = () => { return $ifaceNil; };
            break;

        case $kindArray:
            typ.zero = () => {
                var arrayClass = $nativeArray(typ.elem.kind);
                if (arrayClass !== Array) {
                    return new arrayClass(typ.len);
                }
                var array = new Array(typ.len);
                for (var i = 0; i < typ.len; i++) {
                    array[i] = typ.elem.zero();
                }
                return array;
            };
            break;

        case $kindStruct:
            typ.zero = () => { return new typ.ptr(); };
            break;

        default:
            $panic(new $String("invalid kind: " + kind));
    }

    // Arithmetics operations for types that support it.
    //
    // Each operation accepts two operands and returns one result. For wrapped types operands are
    // passed as bare values and a bare value is returned.
    //
    // This methods will be called when the exact type is not known at code generation time, for
    // example, when operands are type parameters.
    switch (kind) {
        case $kindInt8:
        case $kindInt16:
        case $kindUint:
        case $kindUint8:
        case $kindUint16:
        case $kindFloat32:
        case $kindFloat64:
            typ.add = (x, y) => $truncateNumber(x + y, typ);
            typ.sub = (x, y) => $truncateNumber(x - y, typ);
            typ.mul = (x, y) => $truncateNumber(x * y, typ);
            break;
        case $kindUint32:
        case $kindUintptr:
            typ.add = (x, y) => $truncateNumber(x + y, typ);
            typ.sub = (x, y) => $truncateNumber(x - y, typ);
            typ.mul = (x, y) => $imul(x, y) >>> 0;
            break;
        case $kindInt:
        case $kindInt32:
            typ.add = (x, y) => $truncateNumber(x + y, typ);
            typ.sub = (x, y) => $truncateNumber(x - y, typ);
            typ.mul = (x, y) => $imul(x, y);
            break;
        case $kindInt64:
        case $kindUint64:
            typ.add = (x, y) => new typ(x.$high + y.$high, x.$low + y.$low);
            typ.sub = (x, y) => new typ(x.$high - y.$high, x.$low - y.$low);
            typ.mul = (x, y) => $mul64(x, y);
            break;
        case $kindComplex64:
        case $kindComplex128:
            typ.add = (x, y) => new typ(x.$real + y.$real, x.$imag + y.$imag);
            typ.sub = (x, y) => new typ(x.$real - y.$real, x.$imag - y.$imag);
            typ.mul = (x, y) => new typ(x.$real * y.$real - x.$imag * y.$imag, x.$real * y.$imag + x.$imag * y.$real);
            break;
        case $kindString:
            typ.add = (x, y) => x + y;
    }

    /**
     * convertFrom converts value src to the type typ.
     *
     * For wrapped types src must be a wrapped value, e.g. for int32 this must be an instance of
     * the $Int32 class, rather than the bare JavaScript number. This is required to determine
     * the original Go type to convert from.
     *
     * The returned value will be a representation of typ; for wrapped values it will be unwrapped;
     * for example, conversion to int32 will return a bare JavaScript number. This is required
     * to make results of type conversion expression consistent with any other expressions of the
     * same type.
     */
    typ.convertFrom = (src) => $convertIdentity(src, typ);
    switch (kind) {
        case $kindInt64:
        case $kindUint64:
            typ.convertFrom = (src) => $convertToInt64(src, typ);
            break;
        case $kindInt8:
        case $kindInt16:
        case $kindInt32:
        case $kindInt:
        case $kindUint8:
        case $kindUint16:
        case $kindUint32:
        case $kindUint:
        case $kindUintptr:
            typ.convertFrom = (src) => $convertToNativeInt(src, typ);
            break;
        case $kindFloat32:
        case $kindFloat64:
            typ.convertFrom = (src) => $convertToFloat(src, typ);
            break;
        case $kindComplex128:
        case $kindComplex64:
            typ.convertFrom = (src) => $convertToComplex(src, typ);
            break;
        case $kindString:
            typ.convertFrom = (src) => $convertToString(src, typ);
            break;
        case $kindUnsafePointer:
            typ.convertFrom = (src) => $convertToUnsafePtr(src, typ);
            break;
        case $kindBool:
            typ.convertFrom = (src) => $convertToBool(src, typ);
            break;
        case $kindInterface:
            typ.convertFrom = (src) => $convertToInterface(src, typ);
            break;
        case $kindSlice:
            typ.convertFrom = (src) => $convertToSlice(src, typ);
            break;
        case $kindPtr:
            typ.convertFrom = (src) => $convertToPointer(src, typ);
            break;
        case $kindArray:
            typ.convertFrom = (src) => $convertToArray(src, typ);
            break;
        case $kindStruct:
            typ.convertFrom = (src) => $convertToStruct(src, typ);
            break;
        case $kindMap:
            typ.convertFrom = (src) => $convertToMap(src, typ);
            break;
        case $kindChan:
            typ.convertFrom = (src) => $convertToChan(src, typ);
            break;
        case $kindFunc:
            typ.convertFrom = (src) => $convertToFunc(src, typ);
            break;
        default:
            $panic(new $String("invalid kind: " + kind));
    }

    typ.id = $typeIDCounter;
    $typeIDCounter++;
    typ.size = size;
    typ.kind = kind;
    typ.string = string;
    typ.named = named;
    typ.pkg = pkg;
    typ.exported = exported;
    typ.methods = [];
    typ.methodSetCache = null;
    typ.comparable = true;
    return typ;
};

var $methodSet = typ => {
    if (typ.methodSetCache !== null) {
        return typ.methodSetCache;
    }
    var base = {};

    var isPtr = (typ.kind === $kindPtr);
    if (isPtr && typ.elem.kind === $kindInterface) {
        typ.methodSetCache = [];
        return [];
    }

    var current = [{ typ: isPtr ? typ.elem : typ, indirect: isPtr }];

    var seen = {};

    while (current.length > 0) {
        var next = [];
        var mset = [];

        current.forEach(e => {
            if (seen[e.typ.string]) {
                return;
            }
            seen[e.typ.string] = true;

            if (e.typ.named) {
                mset = mset.concat(e.typ.methods);
                if (e.indirect) {
                    mset = mset.concat($ptrType(e.typ).methods);
                }
            }

            switch (e.typ.kind) {
                case $kindStruct:
                    e.typ.fields.forEach(f => {
                        if (f.embedded) {
                            var fTyp = f.typ;
                            var fIsPtr = (fTyp.kind === $kindPtr);
                            next.push({ typ: fIsPtr ? fTyp.elem : fTyp, indirect: e.indirect || fIsPtr });
                        }
                    });
                    break;

                case $kindInterface:
                    mset = mset.concat(e.typ.methods);
                    break;
            }
        });

        mset.forEach(m => {
            if (base[m.name] === undefined) {
                base[m.name] = m;
            }
        });

        current = next;
    }

    typ.methodSetCache = [];
    Object.keys(base).sort().forEach(name => {
        typ.methodSetCache.push(base[name]);
    });
    return typ.methodSetCache;
};

var $instantiateMethods = function(typeInstance, methodFactories, ...typeArgs) {
    for (let [method, factory] of Object.entries(methodFactories)) {
      typeInstance.prototype[method] = factory(...typeArgs);
    }
  };

var $Bool = $newType(1, $kindBool, "bool", true, "", false, null);
var $Int = $newType(4, $kindInt, "int", true, "", false, null);
var $Int8 = $newType(1, $kindInt8, "int8", true, "", false, null);
var $Int16 = $newType(2, $kindInt16, "int16", true, "", false, null);
var $Int32 = $newType(4, $kindInt32, "int32", true, "", false, null);
var $Int64 = $newType(8, $kindInt64, "int64", true, "", false, null);
var $Uint = $newType(4, $kindUint, "uint", true, "", false, null);
var $Uint8 = $newType(1, $kindUint8, "uint8", true, "", false, null);
var $Uint16 = $newType(2, $kindUint16, "uint16", true, "", false, null);
var $Uint32 = $newType(4, $kindUint32, "uint32", true, "", false, null);
var $Uint64 = $newType(8, $kindUint64, "uint64", true, "", false, null);
var $Uintptr = $newType(4, $kindUintptr, "uintptr", true, "", false, null);
var $Float32 = $newType(4, $kindFloat32, "float32", true, "", false, null);
var $Float64 = $newType(8, $kindFloat64, "float64", true, "", false, null);
var $Complex64 = $newType(8, $kindComplex64, "complex64", true, "", false, null);
var $Complex128 = $newType(16, $kindComplex128, "complex128", true, "", false, null);
var $String = $newType(8, $kindString, "string", true, "", false, null);
var $UnsafePointer = $newType(4, $kindUnsafePointer, "unsafe.Pointer", true, "unsafe", false, null);

var $nativeArray = elemKind => {
    switch (elemKind) {
        case $kindInt:
            return Int32Array;
        case $kindInt8:
            return Int8Array;
        case $kindInt16:
            return Int16Array;
        case $kindInt32:
            return Int32Array;
        case $kindUint:
            return Uint32Array;
        case $kindUint8:
            return Uint8Array;
        case $kindUint16:
            return Uint16Array;
        case $kindUint32:
            return Uint32Array;
        case $kindUintptr:
            return Uint32Array;
        case $kindFloat32:
            return Float32Array;
        case $kindFloat64:
            return Float64Array;
        default:
            return Array;
    }
};
var $toNativeArray = (elemKind, array) => {
    var nativeArray = $nativeArray(elemKind);
    if (nativeArray === Array) {
        return array;
    }
    return new nativeArray(array);
};
var $arrayTypes = {};
var $arrayType = (elem, len) => {
    var typeKey = elem.id + "$" + len;
    var typ = $arrayTypes[typeKey];
    if (typ === undefined) {
        typ = $newType(elem.size * len, $kindArray, "[" + len + "]" + elem.string, false, "", false, null);
        $arrayTypes[typeKey] = typ;
        typ.init(elem, len);
    }
    return typ;
};

var $chanType = (elem, sendOnly, recvOnly) => {
    var string = (recvOnly ? "<-" : "") + "chan" + (sendOnly ? "<- " : " ");
    if (!sendOnly && !recvOnly && (elem.string[0] == "<")) {
        string += "(" + elem.string + ")";
    } else {
        string += elem.string;
    }
    var field = sendOnly ? "SendChan" : (recvOnly ? "RecvChan" : "Chan");
    var typ = elem[field];
    if (typ === undefined) {
        typ = $newType(4, $kindChan, string, false, "", false, null);
        elem[field] = typ;
        typ.init(elem, sendOnly, recvOnly);
    }
    return typ;
};
var $Chan = function (elem, capacity) {
    if (capacity < 0 || capacity > 2147483647) {
        $throwRuntimeError("makechan: size out of range");
    }
    this.$elem = elem;
    this.$capacity = capacity;
    this.$buffer = [];
    this.$sendQueue = [];
    this.$recvQueue = [];
    this.$closed = false;
};
var $chanNil = new $Chan(null, 0);
$chanNil.$sendQueue = $chanNil.$recvQueue = { length: 0, push() { }, shift() { return undefined; }, indexOf() { return -1; } };

var $funcTypes = {};
var $funcType = (params, results, variadic) => {
    var typeKey = $mapArray(params, p => { return p.id; }).join(",") + "$" + $mapArray(results, r => { return r.id; }).join(",") + "$" + variadic;
    var typ = $funcTypes[typeKey];
    if (typ === undefined) {
        var paramTypes = $mapArray(params, p => { return p.string; });
        if (variadic) {
            paramTypes[paramTypes.length - 1] = "..." + paramTypes[paramTypes.length - 1].substr(2);
        }
        var string = "func(" + paramTypes.join(", ") + ")";
        if (results.length === 1) {
            string += " " + results[0].string;
        } else if (results.length > 1) {
            string += " (" + $mapArray(results, r => { return r.string; }).join(", ") + ")";
        }
        typ = $newType(4, $kindFunc, string, false, "", false, null);
        $funcTypes[typeKey] = typ;
        typ.init(params, results, variadic);
    }
    return typ;
};

var $interfaceTypes = {};
var $interfaceType = methods => {
    var typeKey = $mapArray(methods, m => { return m.pkg + "," + m.name + "," + m.typ.id; }).join("$");
    var typ = $interfaceTypes[typeKey];
    if (typ === undefined) {
        var string = "interface {}";
        if (methods.length !== 0) {
            string = "interface { " + $mapArray(methods, m => {
                return (m.pkg !== "" ? m.pkg + "." : "") + m.name + m.typ.string.substr(4);
            }).join("; ") + " }";
        }
        typ = $newType(8, $kindInterface, string, false, "", false, null);
        $interfaceTypes[typeKey] = typ;
        typ.init(methods);
    }
    return typ;
};
var $emptyInterface = $interfaceType([]);
var $ifaceNil = {};
var $error = $newType(8, $kindInterface, "error", true, "", false, null);
$error.init([{ prop: "Error", name: "Error", pkg: "", typ: $funcType([], [$String], false) }]);

var $mapTypes = {};
var $mapType = (key, elem) => {
    var typeKey = key.id + "$" + elem.id;
    var typ = $mapTypes[typeKey];
    if (typ === undefined) {
        typ = $newType(4, $kindMap, "map[" + key.string + "]" + elem.string, false, "", false, null);
        $mapTypes[typeKey] = typ;
        typ.init(key, elem);
    }
    return typ;
};
var $makeMap = (keyForFunc, entries) => {
    var m = new Map();
    for (var i = 0; i < entries.length; i++) {
        var e = entries[i];
        m.set(keyForFunc(e.k), e);
    }
    return m;
};

var $ptrType = elem => {
    var typ = elem.ptr;
    if (typ === undefined) {
        typ = $newType(4, $kindPtr, "*" + elem.string, false, "", elem.exported, null);
        elem.ptr = typ;
        typ.init(elem);
    }
    return typ;
};

var $newDataPointer = (data, constructor) => {
    if (constructor.elem.kind === $kindStruct) {
        return data;
    }
    return new constructor(() => { return data; }, v => { data = v; });
};

var $indexPtr = (array, index, constructor) => {
    if (constructor.kind == $kindPtr && constructor.elem.kind == $kindStruct) {
        // Pointer to a struct is represented by the underlying object itself, no wrappers needed.
        return array[index]
    }
    if (array.buffer) {
        // Pointers to the same underlying ArrayBuffer share cache.
        var cache = array.buffer.$ptr = array.buffer.$ptr || {};
        // Pointers of different primitive types are non-comparable and stored in different caches.
        var typeCache = cache[array.name] = cache[array.name] || {};
        var cacheIdx = array.BYTES_PER_ELEMENT * index + array.byteOffset;
        return typeCache[cacheIdx] || (typeCache[cacheIdx] = new constructor(() => { return array[index]; }, v => { array[index] = v; }));
    } else {
        array.$ptr = array.$ptr || {};
        return array.$ptr[index] || (array.$ptr[index] = new constructor(() => { return array[index]; }, v => { array[index] = v; }));
    }
};

var $sliceType = elem => {
    var typ = elem.slice;
    if (typ === undefined) {
        typ = $newType(12, $kindSlice, "[]" + elem.string, false, "", false, null);
        elem.slice = typ;
        typ.init(elem);
    }
    return typ;
};
var $makeSlice = (typ, length, capacity = length) => {
    if (length < 0 || length > 2147483647) {
        $throwRuntimeError("makeslice: len out of range");
    }
    if (capacity < 0 || capacity < length || capacity > 2147483647) {
        $throwRuntimeError("makeslice: cap out of range");
    }
    var array = new typ.nativeArray(capacity);
    if (typ.nativeArray === Array) {
        for (var i = 0; i < capacity; i++) {
            array[i] = typ.elem.zero();
        }
    }
    var slice = new typ(array);
    slice.$length = length;
    return slice;
};

var $structTypes = {};
var $structType = (pkgPath, fields) => {
    var typeKey = $mapArray(fields, f => { return f.name + "," + f.typ.id + "," + f.tag; }).join("$");
    var typ = $structTypes[typeKey];
    if (typ === undefined) {
        var string = "struct { " + $mapArray(fields, f => {
            var str = f.typ.string + (f.tag !== "" ? (" \"" + f.tag.replace(/\\/g, "\\\\").replace(/"/g, "\\\"") + "\"") : "");
            if (f.embedded) {
                return str;
            }
            return f.name + " " + str;
        }).join("; ") + " }";
        if (fields.length === 0) {
            string = "struct {}";
        }
        typ = $newType(0, $kindStruct, string, false, "", false, function(...args) {
            this.$val = this;
            for (var i = 0; i < fields.length; i++) {
                var f = fields[i];
                if (f.name == '_') {
                    continue;
                }
                var arg = args[i];
                this[f.prop] = arg !== undefined ? arg : f.typ.zero();
            }
        });
        $structTypes[typeKey] = typ;
        typ.init(pkgPath, fields);
    }
    return typ;
};

var $assertType = (value, type, returnTuple) => {
    var isInterface = (type.kind === $kindInterface), ok, missingMethod = "";
    if (value === $ifaceNil) {
        ok = false;
    } else if (!isInterface) {
        ok = value.constructor === type;
    } else {
        var valueTypeString = value.constructor.string;
        ok = type.implementedBy[valueTypeString];
        if (ok === undefined) {
            ok = true;
            var valueMethodSet = $methodSet(value.constructor);
            var interfaceMethods = type.methods;
            for (var i = 0; i < interfaceMethods.length; i++) {
                var tm = interfaceMethods[i];
                var found = false;
                for (var j = 0; j < valueMethodSet.length; j++) {
                    var vm = valueMethodSet[j];
                    if (vm.name === tm.name && vm.pkg === tm.pkg && vm.typ === tm.typ) {
                        found = true;
                        break;
                    }
                }
                if (!found) {
                    ok = false;
                    type.missingMethodFor[valueTypeString] = tm.name;
                    break;
                }
            }
            type.implementedBy[valueTypeString] = ok;
        }
        if (!ok) {
            missingMethod = type.missingMethodFor[valueTypeString];
        }
    }

    if (!ok) {
        if (returnTuple) {
            return [type.zero(), false];
        }
        $panic(new $packages["runtime"].TypeAssertionError.ptr(
            $packages["runtime"]._type.ptr.nil,
            (value === $ifaceNil ? $packages["runtime"]._type.ptr.nil : new $packages["runtime"]._type.ptr(value.constructor.string)),
            new $packages["runtime"]._type.ptr(type.string),
            missingMethod));
    }

    if (!isInterface) {
        value = value.$val;
    }
    if (type === $jsObjectPtr) {
        value = value.object;
    }
    return returnTuple ? [value, true] : value;
};

const $isSigned = (typ) => {
    switch (typ.kind) {
        case $kindInt:
        case $kindInt8:
        case $kindInt16:
        case $kindInt32:
        case $kindInt64:
            return true;
        default:
            return false;
    }
}

/**
 * Truncate a JavaScript number `n` according to precision of the Go type `typ`
 * it is supposed to represent.
 */
const $truncateNumber = (n, typ) => {
    switch (typ.kind) {
        case $kindInt8:
            return n << 24 >> 24;
        case $kindUint8:
            return n << 24 >>> 24;
        case $kindInt16:
            return n << 16 >> 16;
        case $kindUint16:
            return n << 16 >>> 16;
        case $kindInt32:
        case $kindInt:
            return n << 0 >> 0;
        case $kindUint32:
        case $kindUint:
        case $kindUintptr:
            return n << 0 >>> 0;
        case $kindFloat32:
            return $fround(n);
        case $kindFloat64:
            return n;
        default:
            $panic(new $String("invalid kind: " + kind));
    }
}

/**
 * Trivial type conversion function, which only accepts destination type identical to the src
 * type.
 *
 * For wrapped types, src value must be wrapped, and the return value will be unwrapped.
 */
const $convertIdentity = (src, dstType) => {
    const srcType = src.constructor;
    if (srcType === dstType) {
        // Same type, no conversion necessary.
        return srcType.wrapped ? src.$val : src;
    }
    throw new Error(`unsupported conversion from ${srcType.string} to ${dstType.string}`);
};

/**
 * Conversion to int64 and uint64 variants.
 *
 * dstType.kind must be either $kindInt64 or $kindUint64. For wrapped types, src
 * value must be wrapped. The returned value is an object instantiated by the
 * `dstType` constructor.
 */
const $convertToInt64 = (src, dstType) => {
    const srcType = src.constructor;
    if (srcType === dstType) {
        return src.$val;
    }

    switch (srcType.kind) {
        case $kindInt64:
        case $kindUint64:
            return new dstType(src.$val.$high, src.$val.$low);
        case $kindUintptr:
            // GopherJS quirk: a uintptr value may be an object converted to
            // uintptr from unsafe.Pointer. Since we don't have actual pointers,
            // we pretend it's value is 1.
            return new dstType(0, src.$val.constructor === Number ? src.$val : 1);
        default:
            return new dstType(0, src.$val);
    }
};

/**
 * Conversion to int and uint types of 32 bits or less.
 *
 * dstType.kind must be $kindInt{8,16,32} or $kindUint{8,16,32}. For wrapped
 * types, src value must be wrapped. The return value will always be a bare
 * JavaScript number, since all 32-or-less integers in GopherJS are considered
 * wrapped types.
 */
const $convertToNativeInt = (src, dstType) => {
    const srcType = src.constructor;
    // Since we are returning a bare number, identical kinds means no actual
    // conversion is required.
    if (srcType.kind === dstType.kind) {
        return src.$val;
    }

    switch (srcType.kind) {
        case $kindInt64:
        case $kindUint64:
            if ($isSigned(srcType) && $isSigned(dstType)) { // Carry over the sign.
                return $truncateNumber(src.$val.$low + ((src.$val.$high >> 31) * 4294967296), dstType);
            }
            return $truncateNumber(src.$val.$low, dstType);
        default:
            return $truncateNumber(src.$val, dstType);
    }
};

/**
 * Conversion to floating point types.
 *
 * dstType.kind must be $kindFloat{32,64}. For wrapped types, src value must be
 * wrapped. Returned value will always be a bare JavaScript number, since all
 * floating point numbers in GopherJS are considered wrapped types.
 */
const $convertToFloat = (src, dstType) => {
    const srcType = src.constructor;
    // Since we are returning a bare number, identical kinds means no actual
    // conversion is required.
    if (srcType.kind === dstType.kind) {
        return src.$val;
    }

    if (dstType.kind == $kindFloat32 && srcType.kind == $kindFloat64) {
        return $fround(src.$val);
    }

    switch (srcType.kind) {
        case $kindInt64:
        case $kindUint64:
            const val = $flatten64(src.$val);
            return (dstType.kind == $kindFloat32) ? $fround(val) : val;
        case $kindFloat64:
            return (dstType.kind == $kindFloat32) ? $fround(src.$val) : src.$val;
        default:
            return src.$val;
    }
};

/**
 * Conversion to complex types.
 *
 * dstType.kind must me $kindComplex{64,128}. Src must be another complex type.
 * Returned value will always be an oject created by the dstType constructor.
 */
const $convertToComplex = (src, dstType) => {
    const srcType = src.constructor;
    if (srcType === dstType) {
        return src;
    }

    return new dstType(src.$real, src.$imag);
};

/**
 * Conversion to string types.
 *
 * dstType.kind must be $kindString. For wrapped types, src value must be
 * wrapped. Returned value will always be a bare JavaScript string.
 */
const $convertToString = (src, dstType) => {
    const srcType = src.constructor;
    if (srcType === dstType) {
        return src.$val;
    }

    switch (srcType.kind) {
        case $kindInt64:
        case $kindUint64:
            return $encodeRune(src.$val.$low);
        case $kindInt32:
        case $kindInt16:
        case $kindInt8:
        case $kindInt:
        case $kindUint32:
        case $kindUint16:
        case $kindUint8:
        case $kindUint:
        case $kindUintptr:
            return $encodeRune(src.$val);
        case $kindString:
            return src.$val;
        case $kindSlice:
            if (srcType.elem.kind === $kindInt32) { // Runes are int32.
                return $runesToString(src.$val);
            } else if (srcType.elem.kind === $kindUint8) { // Bytes are uint8.
                return $bytesToString(src.$val);
            }
            break;
    }

    throw new Error(`Unsupported conversion from ${srcType.string} to ${dstType.string}`);
};

/**
 * Convert to unsafe.Pointer.
 */
const $convertToUnsafePtr = (src, dstType) => {
    // Unsafe pointers are not well supported by GopherJS and whatever support
    // exists is not well documented. Implementing this conversion will require
    // a great deal of great reverse engineering, and I'm not sure anyone will
    // ever need this.
    throw new Error(`Conversion between a typeparam and unsafe.Pointer is not implemented.`)
};

/**
 * Convert to boolean types.
 *
 * dstType.kind must be $kindBool. Src must be a wrapped boolean value. Returned
 * value will always be a bare JavaScript boolean.
 */
const $convertToBool = (src, dstType) => {
    return src.$val;
};

/**
 * Convert any type to an interface value.
 *
 * dstType.kind must be $kindInterface. For wrapped types, src value must be
 * wrapped. Since GopherJS represents interfaces as wrapped values of the original
 * type, the returned value is always src.
 */
const $convertToInterface = (src, dstType) => {
    return src;
};

/**
 * Convert to a slice value.
 *
 * dstType.kind must be $kindSlice. For wrapped types, src value must be wrapped.
 * The returned value is always a slice type.
 */
const $convertToSlice = (src, dstType) => {
    const srcType = src.constructor;
    if (srcType === dstType) {
        return src;
    }

    switch (srcType.kind) {
        case $kindString:
            if (dstType.elem.kind === $kindInt32) { // Runes are int32.
                return new dstType($stringToRunes(src.$val));
            } else if (dstType.elem.kind === $kindUint8) { // Bytes are uint8.
                return new dstType($stringToBytes(src.$val));
            }
            break;
        case $kindSlice:
            return $convertSliceType(src, dstType);
            break;
    }
    throw new Error(`Unsupported conversion from ${srcType.string} to ${dstType.string}`);
};

/**
* Convert to a pointer value.
*
* dstType.kind must be $kindPtr. For wrapped types (specifically, pointers
* to an array), src value must be wrapped. The returned value is a bare JS
* array (typed or untyped), or a pointer object.
*/
const $convertToPointer = (src, dstType) => {
    const srcType = src.constructor;

    if (srcType === dstType) {
        return src;
    }

    // []T → *[N]T
    if (srcType.kind == $kindSlice && dstType.elem.kind == $kindArray) {
        return $sliceToGoArray(src, dstType);
    }

    if (src === srcType.nil) {
        return dstType.nil;
    }

    switch (dstType.elem.kind) {
        case $kindArray:
            // Pointers to arrays are a wrapped type, represented by a native JS array,
            // which we return directly.
            return src.$val;
        case $kindStruct:
            return $pointerOfStructConversion(src, dstType);
        default:
            return new dstType(src.$get, src.$set, src.$target);
    }
};

/**
 * Convert to struct types.
 *
 * dstType.kind must be $kindStruct. Src must be a wrapped struct value. Returned
 * value will always be a bare JavaScript object representing the struct.
 */
const $convertToStruct = (src, dstType) => {
    // Since structs are passed by value, the conversion result must be a copy
    // of the original value, even if it is the same type.
    return $clone(src.$val, dstType);
};

/**
 * Convert to array types.
 *
 * dstType.kind must be $kindArray. Src must be a wrapped array value. Returned
 * value will always be a bare JavaScript object representing the array.
 */
const $convertToArray = (src, dstType) => {
    // Since arrays are passed by value, the conversion result must be a copy
    // of the original value, even if it is the same type.
    return $clone(src.$val, dstType);
};

/**
 * Convert to map types.
 *
 * dstType.kind must be $kindMap. Src must be a wrapped map value. Returned
 * value will always be a bare JavaScript object representing the map.
 */
const $convertToMap = (src, dstType) => {
    return src.$val;
};

/**
 * Convert to chan types.
 *
 * dstType.kind must be $kindChan. Src must be a wrapped chan value. Returned
 * value will always be a bare $Chan object representing the channel.
 */
const $convertToChan = (src, dstType) => {
    return src.$val;
};

/**
 * Convert to function types.
 *
 * dstType.kind must be $kindFunc. Src must be a wrapped function value. Returned
 * value will always be a bare JavaScript function.
 */
const $convertToFunc = (src, dstType) => {
    return src.$val;
};
