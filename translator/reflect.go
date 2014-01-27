package translator

func init() {
	pkgNatives["reflect"] = map[string]string{
		"init": `
			go$reflect = {
				rtype: rtype.Ptr, uncommonType: uncommonType.Ptr, method: method.Ptr, arrayType: arrayType.Ptr, chanType: chanType.Ptr, funcType: funcType.Ptr, interfaceType: interfaceType.Ptr, mapType: mapType.Ptr, ptrType: ptrType.Ptr, sliceType: sliceType.Ptr, structType: structType.Ptr,
				imethod: imethod.Ptr, structField: structField.Ptr,
				kinds: { Bool: go$pkg.Bool, Int: go$pkg.Int, Int8: go$pkg.Int8, Int16: go$pkg.Int16, Int32: go$pkg.Int32, Int64: go$pkg.Int64, Uint: go$pkg.Uint, Uint8: go$pkg.Uint8, Uint16: go$pkg.Uint16, Uint32: go$pkg.Uint32, Uint64: go$pkg.Uint64, Uintptr: go$pkg.Uintptr, Float32: go$pkg.Float32, Float64: go$pkg.Float64, Complex64: go$pkg.Complex64, Complex128: go$pkg.Complex128, Array: go$pkg.Array, Chan: go$pkg.Chan, Func: go$pkg.Func, Interface: go$pkg.Interface, Map: go$pkg.Map, Ptr: go$pkg.Ptr, Slice: go$pkg.Slice, String: go$pkg.String, Struct: go$pkg.Struct, UnsafePointer: go$pkg.UnsafePointer },
				RecvDir: go$pkg.RecvDir, SendDir: go$pkg.SendDir, BothDir: go$pkg.BothDir
			};

			var isWrapped = function(typ) {
				switch (typ.Kind()) {
				case go$pkg.Bool:
				case go$pkg.Int:
				case go$pkg.Int8:
				case go$pkg.Int16:
				case go$pkg.Int32:
				case go$pkg.Uint:
				case go$pkg.Uint8:
				case go$pkg.Uint16:
				case go$pkg.Uint32:
				case go$pkg.Uintptr:
				case go$pkg.Float32:
				case go$pkg.Float64:
				case go$pkg.Array:
				case go$pkg.Map:
				case go$pkg.Func:
				case go$pkg.String:
				case go$pkg.Struct:
					return true;
				case go$pkg.Ptr:
					return typ.Elem().Kind() === go$pkg.Array;
				}
				return false;
			};
			var fieldName = function(field, i) {
				if (field.name.go$get === go$throwNilPointerError) {
					var ntyp = field.typ;
					if (ntyp.Kind() === go$pkg.Ptr) {
						ntyp = ntyp.Elem().common();
					}
					return ntyp.Name();
				}
				var name = field.name.go$get();
				if (name === "_" || go$reservedKeywords.indexOf(name) != -1) {
					return name + "$" + i;
				}
				return name;
			};
			var copyStruct = function(dst, src, typ) {
				var fields = typ.structType.fields.array, i;
				for (i = 0; i < fields.length; i += 1) {
					var field = fields[i];
					var name = fieldName(field, i);
					dst[name] = src[name];
				}
			};
			var deepValueEqual = function(v1, v2, visited) {
				if (!v1.IsValid() || !v2.IsValid()) {
					return !v1.IsValid() && !v2.IsValid();
				}
				if (v1.Type() !== v2.Type()) {
					return false;
				}

				var i;
				switch(v1.Kind()) {
				case go$pkg.Array:
				case go$pkg.Map:
				case go$pkg.Slice:
				case go$pkg.Struct:
					for (i = 0; i < visited.length; i += 1) {
						var entry = visited[i];
						if (v1.val === entry[0] && v2.val === entry[1]) {
							return true;
						}
					}
					visited.push([v1.val, v2.val]);
				}

				switch(v1.Kind()) {
				case go$pkg.Array:
				case go$pkg.Slice:
					if (v1.Kind() === go$pkg.Slice) {
						if (v1.IsNil() !== v2.IsNil()) {
							return false;
						}
						if (v1.iword() === v2.iword()) {
							return true;
						}
					}
					var n = v1.Len();
					if (n !== v2.Len()) {
						return false;
					}
					for (i = 0; i < n; i += 1) {
						if (!deepValueEqual(v1.Index(i), v2.Index(i), visited)) {
							return false;
						}
					}
					return true;
				case go$pkg.Interface:
					if (v1.IsNil() || v2.IsNil()) {
						return v1.IsNil() && v2.IsNil();
					}
					return deepValueEqual(v1.Elem(), v2.Elem(), visited);
				case go$pkg.Ptr:
					return deepValueEqual(v1.Elem(), v2.Elem(), visited);
				case go$pkg.Struct:
					var n = v1.NumField();
					for (i = 0; i < n; i += 1) {
						if (!deepValueEqual(v1.Field(i), v2.Field(i), visited)) {
							return false;
						}
					}
					return true;
				case go$pkg.Map:
					if (v1.IsNil() !== v2.IsNil()) {
						return false;
					}
					if (v1.iword() === v2.iword()) {
						return true;
					}
					var keys = v1.MapKeys();
					if (keys.length !== v2.Len()) {
						return false;
					}
					for (i = 0; i < keys.length; i++) {
						var k = keys.array[i];
						if (!deepValueEqual(v1.MapIndex(k), v2.MapIndex(k), visited)) {
							return false;
						}
					}
					return true;
				case go$pkg.Func:
					return v1.IsNil() && v2.IsNil();
				}

				return go$interfaceIsEqual(valueInterface(v1, false), valueInterface(v2, false));
			};
			var zeroVal = function(typ) {
				switch (typ.Kind()) {
				case go$pkg.Bool:
					return false;
				case go$pkg.Int:
				case go$pkg.Int8:
				case go$pkg.Int16:
				case go$pkg.Int32:
				case go$pkg.Uint:
				case go$pkg.Uint8:
				case go$pkg.Uint16:
				case go$pkg.Uint32:
				case go$pkg.Uintptr:
				case go$pkg.Float32:
				case go$pkg.Float64:
					return 0;
				case go$pkg.Int64:
				case go$pkg.Uint64:
				case go$pkg.Complex64:
				case go$pkg.Complex128:
					return new typ.jsType(0, 0);
				case go$pkg.Array:
					var elemType = typ.Elem();
					return go$makeNativeArray(elemType.jsType.kind, typ.Len(), function() { return zeroVal(elemType); });
				case go$pkg.Func:
					return go$throwNilPointerError;
				case go$pkg.Interface:
					return null;
				case go$pkg.Map:
					return false;
				case go$pkg.Chan:
				case go$pkg.Ptr:
				case go$pkg.Slice:
					return typ.jsType.nil;
				case go$pkg.String:
					return "";
				case go$pkg.Struct:
					return new typ.jsType.Ptr();
				default:
					throw go$panic(new ValueError.Ptr("reflect.Zero", this.kind()));
				}
			};
		`,

		"TypeOf": `function(i) {
			if (i === null) {
				return null;
			}
			if (i.constructor.kind === undefined) { // js.Object
				return Go$String.reflectType();
			}
			return i.constructor.reflectType();
		}`,
		"ValueOf": `function(i) {
			if (i === null) {
				return new Value.Ptr();
			}
			if (i.constructor.kind === undefined) { // js.Object
				return new Value.Ptr(Go$String.reflectType(), String(i), go$pkg.String << flagKindShift);
			}
			var typ = i.constructor.reflectType();
			return new Value.Ptr(typ, i.go$val, typ.Kind() << flagKindShift);
		}`,
		"arrayOf": `function(n, t) {
			return go$arrayType(t.jsType, n).reflectType();
		}`,
		"ChanOf": `function(dir, t) {
			return go$chanType(t.jsType, dir === go$pkg.SendDir, dir === go$pkg.RecvDir).reflectType();
		}`,
		"MapOf": `function(key, elem) {
			switch (key.Kind()) {
			case go$pkg.Func:
			case go$pkg.Map:
			case go$pkg.Slice:
				throw go$panic("reflect.MapOf: invalid key type " + key.String());
			}
			return go$mapType(key.jsType, elem.jsType).reflectType();
		}`,
		"rtype.ptrTo": `function() {
			return go$ptrType(this.jsType).reflectType();
		}`,
		"SliceOf": `function(t) {
			return go$sliceType(t.jsType).reflectType();
		}`,
		"Zero": `function(typ) {
			return new Value.Ptr(typ, zeroVal(typ), typ.Kind() << flagKindShift);
		}`,
		"unsafe_New": `function(typ) {
			switch (typ.Kind()) {
			case go$pkg.Struct:
				return new typ.jsType.Ptr();
			case go$pkg.Array:
				return zeroVal(typ);
			default:
				return go$newDataPointer(zeroVal(typ), typ.ptrTo().jsType);
			}
		}`,
		"makechan": `function(typ, size) {
			return new typ.jsType();
		}`,
		"makeComplex": `function(f, v, typ) {
			return new Value.Ptr(typ, new typ.jsType(v.real, v.imag), f | (typ.Kind() << flagKindShift));
		}`,
		"MakeFunc": `function(typ, fn) {
			var fv = function() {
				var args = new Array(typ.NumIn()), i;
				for (i = 0; i < typ.NumIn(); i += 1) {
					var t = typ.In(i);
					args[i] = new Value.Ptr(t, arguments[i], t.Kind() << flagKindShift);
				}
				var resultsSlice = fn(new (go$sliceType(Value.Ptr))(args));
				switch (typ.NumOut()) {
				case 0:
					return;
				case 1:
					return resultsSlice.array[resultsSlice.offset].iword();
				default:
					var results = new Array(typ.NumOut());
					for (i = 0; i < typ.NumOut(); i += 1) {
						results[i] = resultsSlice.array[resultsSlice.offset + i].iword();
					}
					return results;
				}
			}
		  return new Value.Ptr(typ, fv, go$pkg.Func << flagKindShift);
		}`,
		"makeInt": `function(f, bits, typ) {
			var val;
			switch (typ.Kind()) {
			case go$pkg.Int8:
				val = bits.low << 24 >> 24;
				break;
			case go$pkg.Int16:
				val = bits.low << 16 >> 16;
				break;
			case go$pkg.Int:
			case go$pkg.Int32:
				val = bits.low >> 0;
				break;
			case go$pkg.Int64:
				return new Value.Ptr(typ, go$newDataPointer(new Go$Int64(bits.high, bits.low), typ.ptrTo().jsType), f | flagIndir | (go$pkg.Int64 << flagKindShift));
			case go$pkg.Uint8:
				val = bits.low << 24 >>> 24;
				break;
			case go$pkg.Uint16:
				val = bits.low << 16 >>> 16;
				break;
			case go$pkg.Uint64:
				return new Value.Ptr(typ, go$newDataPointer(bits, typ.ptrTo().jsType), f | flagIndir | (go$pkg.Int64 << flagKindShift));
			case go$pkg.Uint:
			case go$pkg.Uint32:
			case go$pkg.Uintptr:
				val = bits.low >>> 0;
				break;
			}
			return new Value.Ptr(typ, val, f | (typ.Kind() << flagKindShift));
		}`,
		"MakeSlice": `function(typ, len, cap) {
			if (typ.Kind() !== go$pkg.Slice) {
				throw go$panic("reflect.MakeSlice of non-slice type");
			}
			if (len < 0) {
				throw go$panic("reflect.MakeSlice: negative len");
			}
			if (cap < 0) {
				throw go$panic("reflect.MakeSlice: negative cap");
			}
			if (len > cap) {
				throw go$panic("reflect.MakeSlice: len > cap");
			}
			return new Value.Ptr(typ.common(), typ.jsType.make(len, cap, function() { return zeroVal(typ.Elem()); }), go$pkg.Slice << flagKindShift);
		}`,
		"cvtDirect": `function(v, typ) {
			var srcVal = v.iword();
			if (srcVal === v.typ.jsType.nil) {
				return new Value.Ptr(typ, typ.jsType.nil, v.flag);
			}

			var val;
			switch (typ.Kind()) {
			case go$pkg.Chan:
				val = new typ.jsType();
				break;
			case go$pkg.Slice:
				val = new typ.jsType(srcVal.array);
				val.length = srcVal.length;
				val.cap = srcVal.cap;
				break;
			case go$pkg.Ptr:
				if (typ.Elem().Kind() === go$pkg.Struct) {
					if (typ.Elem() === v.typ.Elem()) {
						val = srcVal;
					}
					val = new typ.jsType();
					copyStruct(val, srcVal, typ.Elem());
					break;
				}
				val = new typ.jsType(srcVal.go$get, srcVal.go$set);
				break;
			case go$pkg.Struct:
				val = new typ.jsType.Ptr();
				copyStruct(val, srcVal, typ);
				break;
			case go$pkg.Array:
			case go$pkg.Func:
			case go$pkg.Interface:
			case go$pkg.Map:
			case go$pkg.String:
				val = srcVal;
				break;
			default:
				throw go$panic(new ValueError.Ptr("reflect.Convert", typ.Kind()));
			}
			return new Value.Ptr(typ, val, (v.flag & flagRO) | (typ.Kind() << flagKindShift));
		}`,
		"cvtStringBytes": `function(v, typ) {
			return new Value.Ptr(typ, new typ.jsType(go$stringToBytes(v.iword())), (v.flag & flagRO) | (go$pkg.Slice << flagKindShift));
		}`,
		"cvtStringRunes": `function(v, typ) {
			return new Value.Ptr(typ, new typ.jsType(go$stringToRunes(v.iword())), (v.flag & flagRO) | (go$pkg.Slice << flagKindShift));
		}`,
		"makemap": `function(t) {
			return new Go$Map();
		}`,
		"mapaccess": `function(t, m, key) {
			var entry = m[key.go$key ? key.go$key() : key];
			if (entry === undefined) {
				return [undefined, false];
			}
			return [entry.v, true];
		}`,
		"mapassign": `function(t, m, key, val, ok) {
			if (!ok) {
				delete m[key.go$key ? key.go$key() : key];
				return;
			}
			if (t.Elem().kind === go$pkg.Struct) {
				var newVal = {};
				copyStruct(newVal, val, t.Elem());
				val = newVal;
			}
			m[key.go$key ? key.go$key() : key] = { k: key, v: val };
		}`,
		"maplen": `function(m) {
			return go$keys(m).length;
		}`,
		"mapiterinit": `function(t, m) {
			return [m, go$keys(m), 0];
		}`,
		"mapiterkey": `function(it) {
			var key = it[1][it[2]];
			return [it[0][key].k, true];
		}`,
		"mapiternext": `function(it) {
			it[2] += 1;
		}`,
		"chancap":   `function(ch) { go$notSupported("channels"); }`,
		"chanclose": `function(ch) { go$notSupported("channels"); }`,
		"chanlen":   `function(ch) { go$notSupported("channels"); }`,
		"chanrecv":  `function(t, ch, nb) { go$notSupported("channels"); }`,
		"chansend":  `function(t, ch, val, nb) { go$notSupported("channels"); }`,
		"valueInterface": `function(v, safe) {
			if (v.flag === 0) {
				throw go$panic(new ValueError.Ptr("reflect.Value.Interface", 0));
			}
			if (safe && (v.flag & flagRO) !== 0) {
				throw go$panic("reflect.Value.Interface: cannot return value obtained from unexported field or method")
			}
			if ((v.flag & flagMethod) !== 0) {
				v = makeMethodValue("Interface", v);
			}
			if (isWrapped(v.typ)) {
				return new v.typ.jsType(v.iword());
			}
			return v.iword();
		}`,
		"makeMethodValue": `function(op, v) {
			if ((v.flag & flagMethod) === 0) {
				throw go$panic("reflect: internal error: invalid use of makePartialFunc");
			}

			var tuple = methodReceiver(op, v, v.flag >> flagMethodShift);
			var fn = tuple[1];
			var rcvr = tuple[2];
			var fv = function() { return fn.apply(rcvr, arguments); };
			return new Value.Ptr(v.Type(), fv, (v.flag & flagRO) | (go$pkg.Func << flagKindShift));
		}`,
		"methodReceiver": `function(op, v, i) {
			var m, t;
			if (v.typ.Kind() === go$pkg.Interface) {
				var tt = v.typ.interfaceType;
				if (i < 0 || i >= tt.methods.length) {
					throw go$panic("reflect: internal error: invalid method index");
				}
				if (v.IsNil()) {
					throw go$panic("reflect: " + op + " of method on nil interface value");
				}
				m = tt.methods.array[i];
				t = m.typ;
			} else {
				var ut = v.typ.uncommon();
				if (ut === uncommonType.Ptr.nil || i < 0 || i >= ut.methods.length) {
					throw go$panic("reflect: internal error: invalid method index");
				}
				m = ut.methods.array[i];
				t = m.mtyp;
			}
			if (m.pkgPath.go$get !== go$throwNilPointerError) {
				throw go$panic("reflect: " + op + " of unexported method");
			}
			var name = m.name.go$get()
			if (go$reservedKeywords.indexOf(name) !== -1) {
				name += "$";
			}
			var rcvr = v.iword();
			if (isWrapped(v.typ)) {
				rcvr = new v.typ.jsType(rcvr);
			}
			return [t, rcvr[name], rcvr];
		}
		ifaceE2I = function(t, src, dst) {
			dst.go$set(src);
		}`,
		"methodName": `function() {
			return "?FIXME?";
		}`,
		"Copy": `function(dst, src) {
			var dk = dst.kind();
			if (dk !== go$pkg.Array && dk !== go$pkg.Slice) {
				throw go$panic(new ValueError.Ptr("reflect.Copy", dk));
			}
			if (dk === go$pkg.Array) {
				dst.mustBeAssignable();
			}
			dst.mustBeExported();

			var sk = src.kind();
			if (sk !== go$pkg.Array && sk != go$pkg.Slice) {
				throw go$panic(new ValueError.Ptr("reflect.Copy", sk));
			}
			src.mustBeExported();

			typesMustMatch("reflect.Copy", dst.typ.Elem(), src.typ.Elem());

			var dstVal = dst.iword();
			if (dk === go$pkg.Array) {
				dstVal = new (go$sliceType(dst.typ.Elem().jsType))(dstVal);
			}
			var srcVal = src.iword();
			if (sk === go$pkg.Array) {
				srcVal = new (go$sliceType(src.typ.Elem().jsType))(srcVal);
			}
			return go$copySlice(dstVal, srcVal);
		}`,

		"uncommonType.Method": `function(i) {
			if (this === uncommonType.Ptr.nil || i < 0 || i >= this.methods.length) {
				throw go$panic("reflect: Method index out of range");
			}
			var p = this.methods.array[i];
			var fl = go$pkg.Func << flagKindShift;
			var pkgPath = "";
			if (p.pkgPath.go$get !== go$throwNilPointerError) {
				pkgPath = p.pkgPath.go$get();
				fl |= flagRO;
			}
			var mt = p.typ;
			var name = p.name.go$get();
			if (go$reservedKeywords.indexOf(name) !== -1) {
				name += "$";
			}
			var fn = function(rcvr) {
				return rcvr[name].apply(rcvr, Array.prototype.slice.apply(arguments, [1]));
			}
			return new Method.Ptr(p.name.go$get(), pkgPath, mt, new Value.Ptr(mt, fn, fl), i);
		}`,

		"Value.iword": `function() {
			if ((this.flag & flagIndir) !== 0 && this.typ.Kind() !== go$pkg.Array && this.typ.Kind() !== go$pkg.Struct) {
				return this.val.go$get();
			}
			return this.val;
		}`,
		"Value.Bytes": `function() {
			this.mustBe(go$pkg.Slice);
			if (this.typ.Elem().Kind() !== go$pkg.Uint8) {
				throw go$panic("reflect.Value.Bytes of non-byte slice");
			}
			return this.iword();
		}`,
		"Value.call": `function(op, args) {
			var t = this.typ, fn, rcvr;

			if ((this.flag & flagMethod) !== 0) {
				var tuple = methodReceiver(op, this, this.flag >> flagMethodShift);
				t = tuple[0];
				fn = tuple[1];
				rcvr = tuple[2];
			} else {
				fn = this.iword();
			}

			if (fn === go$throwNilPointerError) {
				throw go$panic("reflect.Value.Call: call of nil function");
			}

			var isSlice = (op === "CallSlice");
			var n = t.NumIn();
			if (isSlice) {
				if (!t.IsVariadic()) {
					throw go$panic("reflect: CallSlice of non-variadic function");
				}
				if (args.length < n) {
					throw go$panic("reflect: CallSlice with too few input arguments");
				}
				if (args.length > n) {
					throw go$panic("reflect: CallSlice with too many input arguments");
				}
			} else {
				if (t.IsVariadic()) {
					n -= 1;
				}
				if (args.length < n) {
					throw go$panic("reflect: Call with too few input arguments");
				}
				if (!t.IsVariadic() && args.length > n) {
					throw go$panic("reflect: Call with too many input arguments");
				}
			}
			var i;
			for (i = 0; i < args.length; i += 1) {
				if (args.array[args.offset + i].Kind() === go$pkg.Invalid) {
					throw go$panic("reflect: " + op + " using zero Value argument");
				}
			}
			for (i = 0; i < n; i += 1) {
				var xt = args.array[args.offset + i].Type(), targ = t.In(i);
				if (!xt.AssignableTo(targ)) {
					throw go$panic("reflect: " + op + " using " + xt.String() + " as type " + targ.String());
				}
			}
			if (!isSlice && t.IsVariadic()) {
				var m = args.length - n;
				var slice = MakeSlice(t.In(n), m, m);
				var elem = t.In(n).Elem();
				for (i = 0; i < m; i += 1) {
					var x = args.array[args.offset + n + i];
					var xt = x.Type();
					if (!xt.AssignableTo(elem)) {
						throw go$panic("reflect: cannot use " + xt.String() + " as type " + elem.String() + " in " + op);
					}
					slice.Index(i).Set(x);
				}
				args = new (go$sliceType(Value))(go$sliceToArray(args).slice(0, n).concat([slice]));
			}

			if (args.length !== t.NumIn()) {
				throw go$panic("reflect.Value.Call: wrong argument count");
			}

			var argsArray = new Array(t.NumIn());
			for (i = 0; i < t.NumIn(); i += 1) {
				argsArray[i] = args.array[args.offset + i].assignTo("reflect.Value.Call", t.In(i), go$ptrType(go$emptyInterface).nil).iword();
			}
			var results = fn.apply(rcvr, argsArray);
			if (t.NumOut() === 0) {
				results = [];
			} else if (t.NumOut() === 1) {
				results = [results];
			}
			for (i = 0; i < t.NumOut(); i += 1) {
				var typ = t.Out(i);
				var flag = typ.Kind() << flagKindShift;
				results[i] = new Value.Ptr(typ, results[i], flag);
			}
			return new (go$sliceType(Value))(results);
		}`,
		"Value.Cap": `function() {
			var k = this.kind();
			switch (k) {
			case go$pkg.Slice:
				return this.iword().capacity;
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.Cap", k));
		}`,
		"Value.Complex": `function() {
			return this.iword();
		}`,
		"Value.Elem": `function() {
			switch (this.kind()) {
			case go$pkg.Interface:
				var val = this.iword();
				if (val === null) {
					return new Value.Ptr();
				}
				if (val.constructor.kind === undefined) { // js.Object
					return new Value.Ptr(Go$String.reflectType(), String(val), go$pkg.String << flagKindShift);
				}
				var typ = val.constructor.reflectType();
				var fl = this.flag & flagRO;
				fl |= typ.Kind() << flagKindShift;
				return new Value.Ptr(typ, val.go$val, fl);

			case go$pkg.Ptr:
				var val = this.iword();
				if (this.IsNil()) {
					return new Value.Ptr();
				}
				var tt = this.typ.ptrType;
				var fl = (this.flag & flagRO) | flagIndir | flagAddr;
				fl |= tt.elem.Kind() << flagKindShift;
				return new Value.Ptr(tt.elem, val, fl);
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.Elem", this.kind()));
		}`,
		"Value.Field": `function(i) {
			this.mustBe(go$pkg.Struct);
			var tt = this.typ.structType;
			if (i < 0 || i >= tt.fields.length) {
				throw go$panic("reflect: Field index out of range");
			}
			var field = tt.fields.array[i];
			var name = fieldName(field, i);
			var typ = field.typ;
			var fl = this.flag & (flagRO | flagIndir | flagAddr);
			if (field.pkgPath.go$get !== go$throwNilPointerError) {
				fl |= flagRO;
			}
			fl |= typ.Kind() << flagKindShift;
			if ((this.flag & flagIndir) !== 0 && typ.Kind() !== go$pkg.Array && typ.Kind() !== go$pkg.Struct) {
				var struct = this.val;
				return new Value.Ptr(typ, new (go$ptrType(typ.jsType))(function() { return struct[name]; }, function(v) { struct[name] = v; }), fl);
			}
			return new Value.Ptr(typ, this.val[name], fl);
		}`,
		"Value.Index": `function(i) {
			var k = this.kind();
			switch (k) {
			case go$pkg.Array:
				var tt = this.typ.arrayType;
				if (i < 0 || i >= tt.len) {
					throw go$panic("reflect: array index out of range");
				}
				var typ = tt.elem;
				var fl = this.flag & (flagRO | flagIndir | flagAddr);
				fl |= typ.Kind() << flagKindShift;
				if ((this.flag & flagIndir) !== 0 && typ.Kind() !== go$pkg.Array && typ.Kind() !== go$pkg.Struct) {
					var array = this.val;
					return new Value.Ptr(typ, new (go$ptrType(typ.jsType))(function() { return array[i]; }, function(v) { array[i] = v; }), fl);
				}
				return new Value.Ptr(typ, this.iword()[i], fl);
			case go$pkg.Slice:
				if (i < 0 || i >= this.iword().length) {
					throw go$panic("reflect: slice index out of range");
				}
				var typ = this.typ.sliceType.elem;
				var fl = flagAddr | flagIndir | (this.flag & flagRO);
				fl |= typ.Kind() << flagKindShift;
				i += this.iword().offset;
				var array = this.iword().array;
				if (typ.Kind() === go$pkg.Struct) {
					return new Value.Ptr(typ, array[i], fl);
				}
				return new Value.Ptr(typ, new (go$ptrType(typ.jsType))(function() { return array[i]; }, function(v) { array[i] = v; }), fl);
			case go$pkg.String:
				var string = this.iword();
				if (i < 0 || i >= string.length) {
					throw go$panic("reflect: string index out of range");
				}
				var fl = (this.flag & flagRO) | (go$pkg.Uint8 << flagKindShift);
				return new Value.Ptr(uint8Type, string.charCodeAt(i), fl);
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.Index", k));
		}`,
		"Value.IsNil": `function() {
			switch (this.kind()) {
			case go$pkg.Chan:
			case go$pkg.Ptr:
			case go$pkg.Slice:
				return this.iword() === this.typ.jsType.nil;
			case go$pkg.Func:
				return this.iword() === go$throwNilPointerError;
			case go$pkg.Map:
				return this.iword() === false;
			case go$pkg.Interface:
				return this.iword() === null;
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.IsNil", this.kind()));
		}`,
		"Value.Len": `function() {
			var k = this.kind();
			switch (k) {
			case go$pkg.Array:
			case go$pkg.Slice:
			case go$pkg.String:
				return this.iword().length;
			case go$pkg.Map:
				return go$keys(this.iword()).length;
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.Len", k));
		}`,
		"Value.runes": `function() {
			this.mustBe(go$pkg.Slice);
			if (this.typ.Elem().Kind() !== go$pkg.Int32) {
				throw new go$panic("reflect.Value.Bytes of non-rune slice");
			}
			return this.iword();
		}`,
		"Value.Pointer": `function() {
			var k = this.kind();
			switch (k) {
			case go$pkg.Chan:
			case go$pkg.Map:
			case go$pkg.Ptr:
			case go$pkg.Slice:
			case go$pkg.UnsafePointer:
				if (this.IsNil()) {
					return 0;
				}
				return this.iword();
			case go$pkg.Func:
				if (this.IsNil()) {
					return 0;
				}
				return 1;
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.Pointer", k));
		}`,
		"Value.Set": `function(x) {
			this.mustBeAssignable();
			x.mustBeExported();
			if ((this.flag & flagIndir) !== 0) {
				switch (this.typ.Kind()) {
				case go$pkg.Array:
					go$copyArray(this.val, x.val);
					return;
				case go$pkg.Interface:
					this.val.go$set(valueInterface(x, false));
					return;
				case go$pkg.Struct:
					copyStruct(this.val, x.val, this.typ);
					return;
				default:
					this.val.go$set(x.iword());
					return;
				}
			}
			this.val = x.val;
		}`,
		"Value.SetInt": `function(x) {
			this.mustBeAssignable();
			var k = this.kind();
			switch (k) {
			case go$pkg.Int:
			case go$pkg.Int8:
			case go$pkg.Int16:
			case go$pkg.Int32:
				this.val.go$set(go$flatten64(x));
				return;
			case go$pkg.Int64:
				this.val.go$set(new this.typ.jsType(x.high, x.low));
				return;
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.SetInt", k));
		}`,
		"Value.SetUint": `function(x) {
			this.mustBeAssignable();
			var k = this.kind();
			switch (k) {
			case go$pkg.Uint:
			case go$pkg.Uint8:
			case go$pkg.Uint16:
			case go$pkg.Uint32:
			case go$pkg.Uintptr:
				this.val.go$set(x.low);
				return;
			case go$pkg.Uint64:
				this.val.go$set(new this.typ.jsType(x.high, x.low));
				return;
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.SetUint", k));
		}`,
		"Value.SetComplex": `function(x) {
			this.mustBeAssignable();
			var k = this.kind();
			switch (k) {
			case go$pkg.Complex64:
			case go$pkg.Complex128:
				this.val.go$set(new this.typ.jsType(x.real, x.imag));
				return;
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.SetComplex", k));
		}`,
		"Value.SetCap": `function(n) {
			this.mustBeAssignable();
			this.mustBe(go$pkg.Slice);
			var s = this.val.go$get();
			if (n < s.length || n > s.capacity) {
				throw go$panic("reflect: slice capacity out of range in SetCap");
			}
			var newSlice = new this.typ.jsType(s.array);
			newSlice.offset = s.offset;
			newSlice.length = s.length;
			newSlice.capacity = n;
			this.val.go$set(newSlice);
		}`,
		"Value.SetLen": `function(n) {
			this.mustBeAssignable();
			this.mustBe(go$pkg.Slice);
			var s = this.val.go$get();
			if (n < 0 || n > s.capacity) {
				throw go$panic("reflect: slice length out of range in SetLen");
			}
			var newSlice = new this.typ.jsType(s.array);
			newSlice.offset = s.offset;
			newSlice.length = n;
			newSlice.capacity = s.capacity;
			this.val.go$set(newSlice);
		}`,
		"Value.Slice": `function(i, j) {
			var typ, s, cap;
			var kind = this.kind();
			switch (kind) {
			case go$pkg.Array:
				if ((this.flag & flagAddr) === 0) {
					throw go$panic("reflect.Value.Slice: slice of unaddressable array");
				}
				var tt = this.typ.arrayType;
				cap = tt.len;
				typ = SliceOf(tt.elem);
				s = new typ.jsType(this.iword());
				break;
			case go$pkg.Slice:
				typ = this.typ.sliceType;
				s = this.iword();
				cap = s.capacity;
				break;
			case go$pkg.String:
				s = this.iword();
				if (i < 0 || j < i || j > s.length) {
					throw go$panic("reflect.Value.Slice: string slice index out of bounds");
				}
				return new Value.Ptr(this.typ, s.substring(i, j), this.flag);
			default:
				throw go$panic(new ValueError.Ptr("reflect.Value.Slice", kind));
			}

			if (i < 0 || j < i || j > cap) {
				throw go$panic("reflect.Value.Slice: slice index out of bounds");
			}

			var fl = (this.flag & flagRO) | (go$pkg.Slice << flagKindShift);
			return new Value.Ptr(typ.common(), go$subslice(s, i, j), fl);
		}`,
		"Value.Slice3": `function(i, j, k) {
			var typ, s, cap;
			var kind = this.kind();
			switch (kind) {
			case go$pkg.Array:
				if ((this.flag & flagAddr) === 0) {
					throw go$panic("reflect.Value.Slice3: slice of unaddressable array");
				}
				var tt = this.typ.arrayType;
				cap = tt.len;
				typ = SliceOf(tt.elem);
				s = new typ.jsType(this.iword());
				break;
			case go$pkg.Slice:
				typ = this.typ.sliceType;
				s = this.iword();
				cap = s.capacity;
				break;
			default:
				throw go$panic(new ValueError.Ptr("reflect.Value.Slice3", kind));
			}

			if (i < 0 || j < i || k < j || k > cap) {
				throw go$panic("reflect.Value.Slice3: slice index out of bounds");
			}

			var fl = (this.flag & flagRO) | (go$pkg.Slice << flagKindShift);
			return new Value.Ptr(typ.common(), go$subslice(s, i, j, k), fl);
		}`,
		"Value.String": `function() {
			switch (this.kind()) {
			case go$pkg.Invalid:
				return "<invalid Value>";
			case go$pkg.String:
				return this.iword();
			}
			return "<" + this.typ.String() + " Value>";
		}`,
		"DeepEqual": `function(a1, a2) {
			if (a1 === a2) {
				return true;
			}
			if (a1 === null || a2 === null || a1.constructor !== a2.constructor) {
				return false;
			}
			return deepValueEqual(ValueOf(a1), ValueOf(a2), []);
		}`,
	}
}
