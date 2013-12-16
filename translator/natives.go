package translator

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
			rtype: rtype, uncommonType: uncommonType, arrayType: arrayType, funcType: funcType, mapType: mapType, ptrType: ptrType, sliceType: sliceType, structType: structType, structField: structField,
			kinds: {bool: Go$pkg.Bool, int: Go$pkg.Int, int8: Go$pkg.Int8, int16: Go$pkg.Int16, int32: Go$pkg.Int32, int64: Go$pkg.Int64, uint: Go$pkg.Uint, uint8: Go$pkg.Uint8, uint16: Go$pkg.Uint16, uint32: Go$pkg.Uint32, uint64: Go$pkg.Uint64, uintptr: Go$pkg.Uintptr, float32: Go$pkg.Float32, float64: Go$pkg.Float64, complex64: Go$pkg.Complex64, complex128: Go$pkg.Complex128, array: Go$pkg.Array, chan: Go$pkg.Chan, func: Go$pkg.Func, interface: Go$pkg.Interface, map: Go$pkg.Map, ptr: Go$pkg.Ptr, slice: Go$pkg.Slice, string: Go$pkg.String, struct: Go$pkg.Struct, "unsafe.Pointer": Go$pkg.UnsafePointer}
		};

		TypeOf = function(i) {
			return i.constructor.Go$type();
		};
		ValueOf = function(i) {
			var typ = i.constructor.Go$type();
			return new Value(typ, i.Go$val, typ.Kind() << flagKindShift);
		};
		Zero = function(typ) {
			var val;
			switch (typ.Kind()) {
			case Go$pkg.Bool:
				val = false;
				break;
			case Go$pkg.Int:
			case Go$pkg.Int8:
			case Go$pkg.Int16:
			case Go$pkg.Int32:
			case Go$pkg.Int64:
			case Go$pkg.Uint:
			case Go$pkg.Uint8:
			case Go$pkg.Uint16:
			case Go$pkg.Uint32:
			case Go$pkg.Uint64:
			case Go$pkg.Float32:
			case Go$pkg.Float64:
				val = 0;
				break;
			case Go$pkg.Complex64:
				val = new Go$Complex64(0, 0);
				break;
			case Go$pkg.Complex128:
				val = new Go$Complex128(0, 0);
				break;
			case Go$pkg.String:
				val = "";
				break;
			case Go$pkg.Map:
				val = false;
				break;
			default:
				throw new Go$Panic("reflect.Zero(" + typ.string.Go$get() + "): type not yet supported");
			}
			return new Value(typ, val, typ.Kind() << flagKindShift);
		};
		New = function(typ) {
			var ptrType = typ.common().ptrTo();
			return new Value(ptrType, Go$newDataPointer(Zero(typ).val, ptrType.alg), Go$pkg.Ptr << flagKindShift);
		};
		makemap = function(t) {
			return new Go$Map();
		};
		mapaccess = function(t, m, key) {
			var entry = m[key];
			if (entry === undefined) {
				return [undefined, false];
			}
			return [entry.v, true];
		};
		mapassign = function(t, m, key, val, ok) {
			m[key] = { k: key, v: val }; // FIXME key
		};

		rtype.prototype.ptrTo = function() {
			return Go$pointerType(this.alg).Go$type();
		};

		Value.prototype.Bytes = function() {
			this.mustBe(Go$pkg.Slice);
			if (this.typ.Elem().Kind() !== Go$pkg.Uint8) {
				throw new Go$Panic("reflect.Value.Bytes of non-byte slice");
			}
			return this.val;
		};
		Value.prototype.call = function(op, args) {
			if (this.val === null) {
				throw new Go$Panic("reflect.Value.Call: call of nil function");
			}

			var isSlice = (op === "CallSlice");
			var t = this.typ;
			var n = t.NumIn();
			if (isSlice) {
				if (!t.IsVariadic()) {
					throw new Go$Panic("reflect: CallSlice of non-variadic function");
				}
				if (args.length < n) {
					throw new Go$Panic("reflect: CallSlice with too few input arguments");
				}
				if (args.length > n) {
					throw new Go$Panic("reflect: CallSlice with too many input arguments");
				}
			} else {
				if (t.IsVariadic()) {
					n -= 1;
				}
				if (args.length < n) {
					throw new Go$Panic("reflect: Call with too few input arguments");
				}
				if (!t.IsVariadic() && args.length > n) {
					throw new Go$Panic("reflect: Call with too many input arguments");
				}
			}
			var i;
			for (i = 0; i < args.length; i += 1) {
				if (args.array[args.offset + i].Kind() === Go$pkg.Invalid) {
					throw new Go$Panic("reflect: " + op + " using zero Value argument");
				}
			}
			for (i = 0; i < n; i += 1) {
				var xt = args.array[args.offset + i].Type(), targ = t.In(i);
				if (!xt.AssignableTo(targ)) {
					throw new Go$Panic("reflect: " + op + " using " + xt.String() + " as type " + targ.String());
				}
			}

			var argsArray = new Array(n);
			for (i = 0; i < n; i += 1) {
				argsArray[i] = args.array[args.offset + i].val;
			}
			var results = this.val.apply(null, argsArray);
			if (t.NumOut() === 0) {
				results = [];
			} else if (t.NumOut() === 1) {
				results = [results];
			}
			for (i = 0; i < t.NumOut(); i += 1) {
				var typ = t.Out(i);
				var flag = typ.Kind() << flagKindShift;
				results[i] = new Value(typ, results[i], flag);
			}
			return new (Go$sliceType(Value))(results);
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
		Value.prototype.Set = function(x) {
			switch (this.typ.Kind()) {
			case Go$pkg.Map:
				this.val = x.val;
				this.flag = x.flag;
				break;
			default:
				throw new Go$Panic("reflect.Value.Set(" + this.typ.string.Go$get() + "): type not yet supported");
			}
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
