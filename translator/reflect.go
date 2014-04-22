package translator

var pkgNatives = make(map[string]map[string]string)

func init() {
	pkgNatives["reflect"] = map[string]string{
		"toplevel": `
			var Invalid = 0;
			var Bool = 1;
			var Int = 2;
			var Int8 = 3;
			var Int16 = 4;
			var Int32 = 5;
			var Int64 = 6;
			var Uint = 7;
			var Uint8 = 8;
			var Uint16 = 9;
			var Uint32 = 10;
			var Uint64 = 11;
			var Uintptr = 12;
			var Float32 = 13;
			var Float64 = 14;
			var Complex64 = 15;
			var Complex128 = 16;
			var Array = 17;
			var Chan = 18;
			var Func = 19;
			var Interface = 20;
			var Map = 21;
			var Ptr = 22;
			var Slice = 23;
			var String = 24;
			var Struct = 25;
			var UnsafePointer = 26;

			var RecvDir = 1;
			var SendDir = 2;
			var BothDir = 3;

			var flagRO = 1;
			var flagIndir = 2;
			var flagAddr = 4;
			var flagMethod = 8;
			var flagKindShift = 4;
			var flagKindWidth = 5;
			var flagKindMask = 31;
			var flagMethodShift = 9;

			var ptrSize = 4;

			go$reflect = {
				rtype: rtype.Ptr, uncommonType: uncommonType.Ptr, method: method.Ptr, arrayType: arrayType.Ptr, chanType: chanType.Ptr, funcType: funcType.Ptr, interfaceType: interfaceType.Ptr, mapType: mapType.Ptr, ptrType: ptrType.Ptr, sliceType: sliceType.Ptr, structType: structType.Ptr,
				imethod: imethod.Ptr, structField: structField.Ptr,
				kinds: { Bool: Bool, Int: Int, Int8: Int8, Int16: Int16, Int32: Int32, Int64: Int64, Uint: Uint, Uint8: Uint8, Uint16: Uint16, Uint32: Uint32, Uint64: Uint64, Uintptr: Uintptr, Float32: Float32, Float64: Float64, Complex64: Complex64, Complex128: Complex128, Array: Array, Chan: Chan, Func: Func, Interface: Interface, Map: Map, Ptr: Ptr, Slice: Slice, String: String, Struct: Struct, UnsafePointer: UnsafePointer },
				RecvDir: RecvDir, SendDir: SendDir, BothDir: BothDir
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
				case Array:
				case Map:
				case Slice:
				case Struct:
					for (i = 0; i < visited.length; i++) {
						var entry = visited[i];
						if (v1.val === entry[0] && v2.val === entry[1]) {
							return true;
						}
					}
					visited.push([v1.val, v2.val]);
				}

				switch(v1.Kind()) {
				case Array:
				case Slice:
					if (v1.Kind() === Slice) {
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
					for (i = 0; i < n; i++) {
						if (!deepValueEqual(v1.Index(i), v2.Index(i), visited)) {
							return false;
						}
					}
					return true;
				case Interface:
					if (v1.IsNil() || v2.IsNil()) {
						return v1.IsNil() && v2.IsNil();
					}
					return deepValueEqual(v1.Elem(), v2.Elem(), visited);
				case Ptr:
					return deepValueEqual(v1.Elem(), v2.Elem(), visited);
				case Struct:
					var n = v1.NumField();
					for (i = 0; i < n; i++) {
						if (!deepValueEqual(v1.Field(i), v2.Field(i), visited)) {
							return false;
						}
					}
					return true;
				case Map:
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
				case Func:
					return v1.IsNil() && v2.IsNil();
				}

				return go$interfaceIsEqual(valueInterface(v1, false), valueInterface(v2, false));
			};
		`,
		"toplevelDependencies": `reflect:rtype reflect:uncommonType reflect:method reflect:arrayType reflect:chanType reflect:funcType reflect:interfaceType reflect:mapType reflect:ptrType reflect:sliceType reflect:structType reflect:imethod reflect:structField reflect:methodReceiver reflect:ValueOf reflect:SliceOf reflect:MakeSlice reflect:valueInterface reflect:makeMethodValue reflect:methodReceiver reflect:typesMustMatch reflect:mustBeAssignable reflect:mustBeExported reflect:mustBe reflect:assignTo`,

		"makeComplex": `function(f, v, typ) {
			return new Value.Ptr(typ, new typ.jsType(v.real, v.imag), f | (typ.Kind() << flagKindShift));
		}`,
		"MakeFunc": `function(typ, fn) {
			var fv = function() {
				var args = new Go$Array(typ.NumIn()), i;
				for (i = 0; i < typ.NumIn(); i++) {
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
					var results = new Go$Array(typ.NumOut());
					for (i = 0; i < typ.NumOut(); i++) {
						results[i] = resultsSlice.array[resultsSlice.offset + i].iword();
					}
					return results;
				}
			}
		  return new Value.Ptr(typ, fv, Func << flagKindShift);
		}`,
		"makeInt": `function(f, bits, typ) {
			var val;
			switch (typ.Kind()) {
			case Int8:
				val = bits.low << 24 >> 24;
				break;
			case Int16:
				val = bits.low << 16 >> 16;
				break;
			case Int:
			case Int32:
				val = bits.low >> 0;
				break;
			case Int64:
				return new Value.Ptr(typ, go$newDataPointer(new Go$Int64(bits.high, bits.low), typ.ptrTo().jsType), f | flagIndir | (Int64 << flagKindShift));
			case Uint8:
				val = bits.low << 24 >>> 24;
				break;
			case Uint16:
				val = bits.low << 16 >>> 16;
				break;
			case Uint64:
				return new Value.Ptr(typ, go$newDataPointer(bits, typ.ptrTo().jsType), f | flagIndir | (Int64 << flagKindShift));
			case Uint:
			case Uint32:
			case Uintptr:
				val = bits.low >>> 0;
				break;
			}
			return new Value.Ptr(typ, val, f | (typ.Kind() << flagKindShift));
		}`,
		"cvtDirect": `function(v, typ) {
			var srcVal = v.iword();
			if (srcVal === v.typ.jsType.nil) {
				if (typ.Kind() === Slice) {
					return new Value.Ptr(typ, go$newDataPointer(typ.jsType.nil, typ.ptrTo().jsType), v.flag);
				}
				return new Value.Ptr(typ, typ.jsType.nil, v.flag);
			}

			var val;
			switch (typ.Kind()) {
			case Chan:
				val = new typ.jsType();
				break;
			case Slice:
				var slice = new typ.jsType(srcVal.array);
				slice.length = srcVal.length;
				slice.cap = srcVal.cap;
				val = go$newDataPointer(slice, typ.ptrTo().jsType);
				break;
			case Ptr:
				if (typ.Elem().Kind() === Struct) {
					if (typ.Elem() === v.typ.Elem()) {
						val = srcVal;
					}
					val = new typ.jsType();
					copyStruct(val, srcVal, typ.Elem());
					break;
				}
				val = new typ.jsType(srcVal.go$get, srcVal.go$set);
				break;
			case Struct:
				val = new typ.jsType.Ptr();
				copyStruct(val, srcVal, typ);
				break;
			case Array:
			case Func:
			case Interface:
			case Map:
			case String:
				val = v.val;
				break;
			default:
				throw go$panic(new ValueError.Ptr("reflect.Convert", typ.Kind()));
			}
			return new Value.Ptr(typ, val, (v.flag & (flagRO | flagIndir)) | (typ.Kind() << flagKindShift));
		}`,
		"cvtStringBytes": `function(v, typ) {
			return new Value.Ptr(typ, new typ.jsType(go$stringToBytes(v.iword())), (v.flag & flagRO) | (Slice << flagKindShift));
		}`,
		"cvtStringRunes": `function(v, typ) {
			return new Value.Ptr(typ, new typ.jsType(go$stringToRunes(v.iword())), (v.flag & flagRO) | (Slice << flagKindShift));
		}`,
		"valueInterface": `function(v, safe) {
			if (v.flag === 0) {
				throw go$panic(new ValueError.Ptr("reflect.Value.Interface", 0));
			}
			if (safe && (v.flag & flagRO) !== 0) {
				throw go$panic(new Go$String("reflect.Value.Interface: cannot return value obtained from unexported field or method"))
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
				throw go$panic(new Go$String("reflect: internal error: invalid use of makePartialFunc"));
			}

			var tuple = methodReceiver(op, v, v.flag >> flagMethodShift);
			var fn = tuple[1];
			var rcvr = tuple[2];
			var fv = function() { return fn.apply(rcvr, arguments); };
			return new Value.Ptr(v.Type(), fv, (v.flag & flagRO) | (Func << flagKindShift));
		}`,
		"methodReceiver": `function(op, v, i) {
			var m, t;
			if (v.typ.Kind() === Interface) {
				var tt = v.typ.interfaceType;
				if (i < 0 || i >= tt.methods.length) {
					throw go$panic(new Go$String("reflect: internal error: invalid method index"));
				}
				if (v.IsNil()) {
					throw go$panic(new Go$String("reflect: " + op + " of method on nil interface value"));
				}
				m = tt.methods.array[i];
				t = m.typ;
			} else {
				var ut = v.typ.uncommon();
				if (ut === uncommonType.Ptr.nil || i < 0 || i >= ut.methods.length) {
					throw go$panic(new Go$String("reflect: internal error: invalid method index"));
				}
				m = ut.methods.array[i];
				t = m.mtyp;
			}
			if (m.pkgPath.go$get !== go$throwNilPointerError) {
				throw go$panic(new Go$String("reflect: " + op + " of unexported method"));
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
		}`,
		"ifaceE2I": `function(t, src, dst) {
			dst.go$set(src);
		}`,
		"methodName": `function() {
			return "?FIXME?";
		}`,

		"uncommonType.Method": `function(i) {
			if (this === uncommonType.Ptr.nil || i < 0 || i >= this.methods.length) {
				throw go$panic(new Go$String("reflect: Method index out of range"));
			}
			var p = this.methods.array[i];
			var fl = Func << flagKindShift;
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
				return rcvr[name].apply(rcvr, Go$Array.prototype.slice.apply(arguments, [1]));
			}
			return new Method.Ptr(p.name.go$get(), pkgPath, mt, new Value.Ptr(mt, fn, fl), i);
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
				throw go$panic(new Go$String("reflect.Value.Call: call of nil function"));
			}

			var isSlice = (op === "CallSlice");
			var n = t.NumIn();
			if (isSlice) {
				if (!t.IsVariadic()) {
					throw go$panic(new Go$String("reflect: CallSlice of non-variadic function"));
				}
				if (args.length < n) {
					throw go$panic(new Go$String("reflect: CallSlice with too few input arguments"));
				}
				if (args.length > n) {
					throw go$panic(new Go$String("reflect: CallSlice with too many input arguments"));
				}
			} else {
				if (t.IsVariadic()) {
					n--;
				}
				if (args.length < n) {
					throw go$panic(new Go$String("reflect: Call with too few input arguments"));
				}
				if (!t.IsVariadic() && args.length > n) {
					throw go$panic(new Go$String("reflect: Call with too many input arguments"));
				}
			}
			var i;
			for (i = 0; i < args.length; i++) {
				if (args.array[args.offset + i].Kind() === Invalid) {
					throw go$panic(new Go$String("reflect: " + op + " using zero Value argument"));
				}
			}
			for (i = 0; i < n; i++) {
				var xt = args.array[args.offset + i].Type(), targ = t.In(i);
				if (!xt.AssignableTo(targ)) {
					throw go$panic(new Go$String("reflect: " + op + " using " + xt.String() + " as type " + targ.String()));
				}
			}
			if (!isSlice && t.IsVariadic()) {
				var m = args.length - n;
				var slice = MakeSlice(t.In(n), m, m);
				var elem = t.In(n).Elem();
				for (i = 0; i < m; i++) {
					var x = args.array[args.offset + n + i];
					var xt = x.Type();
					if (!xt.AssignableTo(elem)) {
						throw go$panic(new Go$String("reflect: cannot use " + xt.String() + " as type " + elem.String() + " in " + op));
					}
					slice.Index(i).Set(x);
				}
				args = new (go$sliceType(Value))(go$sliceToArray(args).slice(0, n).concat([slice]));
			}

			if (args.length !== t.NumIn()) {
				throw go$panic(new Go$String("reflect.Value.Call: wrong argument count"));
			}

			var argsArray = new Go$Array(t.NumIn());
			for (i = 0; i < t.NumIn(); i++) {
				argsArray[i] = args.array[args.offset + i].assignTo("reflect.Value.Call", t.In(i), go$ptrType(go$emptyInterface).nil).iword();
			}
			var results = fn.apply(rcvr, argsArray);
			if (t.NumOut() === 0) {
				results = [];
			} else if (t.NumOut() === 1) {
				results = [results];
			}
			for (i = 0; i < t.NumOut(); i++) {
				var typ = t.Out(i);
				var flag = typ.Kind() << flagKindShift;
				results[i] = new Value.Ptr(typ, results[i], flag);
			}
			return new (go$sliceType(Value))(results);
		}`,
		"Value.Field": `function(i) {
			this.mustBe(Struct);
			var tt = this.typ.structType;
			if (i < 0 || i >= tt.fields.length) {
				throw go$panic(new Go$String("reflect: Field index out of range"));
			}
			var field = tt.fields.array[i];
			var name = this.typ.jsType.fields[i][0];
			var typ = field.typ;
			var fl = this.flag & (flagRO | flagIndir | flagAddr);
			if (field.pkgPath.go$get !== go$throwNilPointerError) {
				fl |= flagRO;
			}
			fl |= typ.Kind() << flagKindShift;
			if (((this.flag & flagIndir) !== 0 || typ.Size() > ptrSize) && typ.Kind() !== Array && typ.Kind() !== Struct) {
				var struct = this.val;
				return new Value.Ptr(typ, new (go$ptrType(typ.jsType))(function() { return struct[name]; }, function(v) { struct[name] = v; }), fl | flagIndir);
			}
			return new Value.Ptr(typ, this.val[name], fl);
		}`,
		"Value.Index": `function(i) {
			var k = this.kind();
			switch (k) {
			case Array:
				var tt = this.typ.arrayType;
				if (i < 0 || i >= tt.len) {
					throw go$panic(new Go$String("reflect: array index out of range"));
				}
				var typ = tt.elem;
				var fl = this.flag & (flagRO | flagIndir | flagAddr);
				fl |= typ.Kind() << flagKindShift;
				if (((this.flag & flagIndir) !== 0 || typ.Size() > ptrSize) && typ.Kind() !== Array && typ.Kind() !== Struct) {
					var array = this.val;
					return new Value.Ptr(typ, new (go$ptrType(typ.jsType))(function() { return array[i]; }, function(v) { array[i] = v; }), fl | flagIndir);
				}
				return new Value.Ptr(typ, this.iword()[i], fl);
			case Slice:
				if (i < 0 || i >= this.iword().length) {
					throw go$panic(new Go$String("reflect: slice index out of range"));
				}
				var typ = this.typ.sliceType.elem;
				var fl = flagAddr | flagIndir | (this.flag & flagRO);
				fl |= typ.Kind() << flagKindShift;
				i += this.iword().offset;
				var array = this.iword().array;
				if (typ.Kind() === Struct) {
					return new Value.Ptr(typ, array[i], fl);
				}
				return new Value.Ptr(typ, new (go$ptrType(typ.jsType))(function() { return array[i]; }, function(v) { array[i] = v; }), fl);
			case String:
				var string = this.iword();
				if (i < 0 || i >= string.length) {
					throw go$panic(new Go$String("reflect: string index out of range"));
				}
				var fl = (this.flag & flagRO) | (Uint8 << flagKindShift);
				return new Value.Ptr(uint8Type, string.charCodeAt(i), fl);
			}
			throw go$panic(new ValueError.Ptr("reflect.Value.Index", k));
		}`,
		"Value.Set": `function(x) {
			this.mustBeAssignable();
			x.mustBeExported();
			if ((this.flag & flagIndir) !== 0) {
				switch (this.typ.Kind()) {
				case Array:
					go$copyArray(this.val, x.val);
					return;
				case Interface:
					this.val.go$set(valueInterface(x, false));
					return;
				case Struct:
					copyStruct(this.val, x.val, this.typ);
					return;
				default:
					this.val.go$set(x.iword());
					return;
				}
			}
			this.val = x.val;
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
