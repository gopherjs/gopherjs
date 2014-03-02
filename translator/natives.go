package translator

var pkgNatives = make(map[string]map[string]string)

func init() {
	pkgNatives["bytes"] = map[string]string{
		"Compare": `function(a, b) {
			var l = Math.min(a.length, b.length), i;
			for (i = 0; i < a.length; i++) {
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
		}`,
		"Equal": `function(a, b) {
			if (a.length !== b.length) {
				return false;
			}
			var i;
			for (i = 0; i < a.length; i++) {
				if (a.array[a.offset + i] !== b.array[b.offset + i]) {
					return false;
				}
			}
			return true;
		}`,
		"IndexByte": `function(s, c) {
			var i;
			for (i = 0; i < s.length; i++) {
				if (s.array[s.offset + i] === c) {
					return i;
				}
			}
			return -1;
		}`,
	}

	pkgNatives["encoding/gob"] = map[string]string{
		"NewDecoder": `function() { go$notSupported("encoding/gob"); }`,
		"NewEncoder": `function() { go$notSupported("encoding/gob"); }`,
	}

	pkgNatives["encoding/json"] = map[string]string{
		"toplevel": `
		  var encodeStates = [];
		`,
		"newEncodeState": `function() {
			var e = encodeStates.pop();
			if (e !== undefined) {
				e.Reset();
				return e;
			}
			return new encodeState.Ptr();
		}`,
		"putEncodeState": `function(e) {
			encodeStates.push(e);
		}`,
	}

	pkgNatives["io/ioutil"] = map[string]string{
		"toplevel": `
			var blackHoles = [];
		`,
		"blackHole": `function() {
			return blackHoles.pop() || go$sliceType(Go$Uint8).make(8192, 0, function() { return 0; });
		}`,
		"blackHolePut": `function(p) {
			blackHoles.push(p);
		}`,
	}

	pkgNatives["math"] = map[string]string{
		"toplevelDependencies": `math.expm1 math.frexp math.hypot math.log10 math.log1p math.log2 math.remainder`,
		"Abs":       `Math.abs`,
		"Acos":      `Math.acos`,
		"Asin":      `Math.asin`,
		"Atan":      `Math.atan`,
		"Atan2":     `Math.atan2`,
		"Ceil":      `Math.ceil`,
		"Copysign":  `function(x, y) { return (x < 0 || 1/x === 1/-0) !== (y < 0 || 1/y === 1/-0) ? -x : x; }`,
		"Cos":       `Math.cos`,
		"Dim":       `function(x, y) { return Math.max(x - y, 0); }`,
		"Exp":       `Math.exp`,
		"Exp2":      `function(x) { return Math.pow(2, x); }`,
		"Expm1":     `function(x) { return expm1(x); }`,
		"Floor":     `Math.floor`,
		"Frexp":     `function(f) { return frexp(f); }`,
		"Hypot":     `function(p, q) { return hypot(p, q); }`,
		"Inf":       `function(sign) { return sign >= 0 ? 1/0 : -1/0; }`,
		"IsInf":     `function(f, sign) { if (f === -1/0) { return sign <= 0; } if (f === 1/0) { return sign >= 0; } return false; }`,
		"IsNaN":     `function(f) { return f !== f; }`,
		"Ldexp":     `go$ldexp`,
		"Log":       `Math.log`,
		"Log10":     `function(x) { return log10(x); }`,
		"Log1p":     `function(x) { return log1p(x); }`,
		"Log2":      `function(x) { return log2(x); }`,
		"Max":       `function(x, y) { return (x === 1/0 || y === 1/0) ? 1/0 : Math.max(x, y); }`,
		"Min":       `function(x, y) { return (x === -1/0 || y === -1/0) ? -1/0 : Math.min(x, y); }`,
		"Mod":       `function(x, y) { return x % y; }`,
		"Modf":      `function(f) { if (f === -1/0 || f === 1/0) { return [f, 0/0]; } var frac = f % 1; return [f - frac, frac]; }`,
		"NaN":       `function() { return 0/0; }`,
		"Pow":       `function(x, y) { return ((x === 1) || (x === -1 && (y === -1/0 || y === 1/0))) ? 1 : Math.pow(x, y); }`,
		"Remainder": `function(x, y) { return remainder(x, y); }`,
		"Signbit":   `function(x) { return x < 0 || 1/x === 1/-0; }`,
		"Sin":       `Math.sin`,
		"Sincos":    `function(x) { return [Math.sin(x), Math.cos(x)]; }`,
		"Sqrt":      `Math.sqrt`,
		"Tan":       `Math.tan`,
		"Trunc":     `function(x) { return (x === 1/0 || x === -1/0 || x !== x || 1/x === 1/-0) ? x : x >> 0; }`,

		// generated from bitcasts/bitcasts.go
		"Float32bits":     `go$float32bits`,
		"Float32frombits": `go$float32frombits`,
		"Float64bits": `function(f) {
			var s, e, x, x$1, x$2, x$3;
			if (f === 0) {
				if (f === 0 && 1 / f === 1 / -0) {
					return new Go$Uint64(2147483648, 0);
				}
				return new Go$Uint64(0, 0);
			}
			if (f !== f) {
				return new Go$Uint64(2146959360, 1);
			}
			s = new Go$Uint64(0, 0);
			if (f < 0) {
				s = new Go$Uint64(2147483648, 0);
				f = -f;
			}
			e = 1075;
			while (f >= 9.007199254740992e+15) {
				f = f / 2;
				if (e === 2047) {
					break;
				}
				e = e + 1 >>> 0;
			}
			while (f < 4.503599627370496e+15) {
				e = e - 1 >>> 0;
				if (e === 0) {
					break;
				}
				f = f * 2;
			}
			return (x = (x$1 = go$shiftLeft64(new Go$Uint64(0, e), 52), new Go$Uint64(s.high | x$1.high, (s.low | x$1.low) >>> 0)), x$2 = (x$3 = new Go$Uint64(0, f), new Go$Uint64(x$3.high &~ 1048576, (x$3.low &~ 0) >>> 0)), new Go$Uint64(x.high | x$2.high, (x.low | x$2.low) >>> 0));
		}`,
		"Float64frombits": `function(b) {
			var s, x, x$1, e, m;
			s = 1;
			if (!((x = new Go$Uint64(b.high & 2147483648, (b.low & 0) >>> 0), (x.high === 0 && x.low === 0)))) {
				s = -1;
			}
			e = (x$1 = go$shiftRightUint64(b, 52), new Go$Uint64(x$1.high & 0, (x$1.low & 2047) >>> 0));
			m = new Go$Uint64(b.high & 1048575, (b.low & 4294967295) >>> 0);
			if ((e.high === 0 && e.low === 2047)) {
				if ((m.high === 0 && m.low === 0)) {
					return s / 0;
				}
				return 0/0;
			}
			if (!((e.high === 0 && e.low === 0))) {
				m = new Go$Uint64(m.high + 1048576, m.low + 0);
			}
			if ((e.high === 0 && e.low === 0)) {
				e = new Go$Uint64(0, 1);
			}
			return go$ldexp(go$flatten64(m), ((e.low >> 0) - 1023 >> 0) - 52 >> 0) * s;
		}`,
	}

	pkgNatives["math/big"] = map[string]string{
		"toplevelDependencies": `math/big.mulWW_g math/big.divWW_g math/big.addVV_g math/big.subVV_g math/big.addVW_g math/big.subVW_g math/big.shlVU_g math/big.shrVU_g math/big.mulAddVWW_g math/big.addMulVVW_g math/big.divWVW_g math/big.bitLen_g`,
		"mulWW":                `function(x, y) { return mulWW_g(x, y); }`,
		"divWW":                `function(x1, x0, y) { return divWW_g(x1, x0, y); }`,
		"addVV":                `function(z, x, y) { return addVV_g(z, x, y); }`,
		"subVV":                `function(z, x, y) { return subVV_g(z, x, y); }`,
		"addVW":                `function(z, x, y) { return addVW_g(z, x, y); }`,
		"subVW":                `function(z, x, y) { return subVW_g(z, x, y); }`,
		"shlVU":                `function(z, x, s) { return shlVU_g(z, x, s); }`,
		"shrVU":                `function(z, x, s) { return shrVU_g(z, x, s); }`,
		"mulAddVWW":            `function(z, x, y, r) { return mulAddVWW_g(z, x, y, r); }`,
		"addMulVVW":            `function(z, x, y) { return addMulVVW_g(z, x, y); }`,
		"divWVW":               `function(z, xn, x, y) { return divWVW_g(z, xn, x, y); }`,
		"bitLen":               `function(x) { return bitLen_g(x); }`,
	}

	pkgNatives["os"] = map[string]string{
		"toplevel": `
			if (go$packages["syscall"].Syscall15 !== undefined) { // windows
				NewFile = go$pkg.NewFile = function() { return new File.Ptr(); };
			}
		`,
		"toplevelDependencies": `syscall.Syscall15 os.NewFile`,
		"Args":                 `new (go$sliceType(Go$String))((typeof process !== 'undefined') ? process.argv.slice(1) : [])`,
	}

	pkgNatives["runtime"] = map[string]string{
		"toplevel": `
			go$throwRuntimeError = function(msg) { throw go$panic(new errorString(msg)); };
		`,
		"toplevelDependencies": `runtime.errorString runtime.TypeAssertionError`,
		"getgoroot": `function() {
			return (typeof process !== 'undefined') ? (process.env["GOROOT"] || "") : "/";
		}`,
		"sizeof_C_MStats": `3712`,
		"Caller": `function(skip) {
			var line = go$getStack()[skip + 3];
			if (line === undefined) {
				return [0, "", 0, false];
			}
			var parts = line.substring(line.indexOf("(") + 1, line.indexOf(")")).split(":");
			return [0, parts[0], parseInt(parts[1]), true];
		}`,
		"GC": `function() {}`,
		"GOMAXPROCS": `function(n) {
			if (n > 1) {
				go$notSupported("GOMAXPROCS != 1");
			}
			return 1;
		}`,
		"Goexit": `function() {
			var err = new Go$Error();
			err.go$exit = true;
			throw err;
		}`,
		"NumCPU":       `function() { return 1; }`,
		"ReadMemStats": `function() {}`,
		"SetFinalizer": `function() {}`,
	}

	pkgNatives["strings"] = map[string]string{
		"IndexByte": `function(s, c) { return s.indexOf(String.fromCharCode(c)); }`,
	}

	pkgNatives["sync"] = map[string]string{
		"copyChecker.check":    `function() {}`,
		"runtime_Syncsemcheck": `function() {}`,
	}

	atomicAdd32 := `function(addr, delta) {
		var value = addr.go$get() + delta;
		addr.go$set(value);
		return value;
	}`
	atomicAdd64 := `function(addr, delta) {
		var value = addr.go$get();
		value = new value.constructor(value.high + delta.high, value.low + delta.low);
		addr.go$set(value);
		return value;
	}`
	atomicCompareAndSwap := `function(addr, oldVal, newVal) {
		if (addr.go$get() === oldVal) {
			addr.go$set(newVal);
			return true;
		}
		return false;
	}`
	atomicLoad := `function(addr) {
		return addr.go$get();
	}`
	atomicStore := `function(addr, val) {
		addr.go$set(val);
	}`
	atomicSwap := `function(addr, newVal) {
		var value = addr.go$get();
		addr.go$set(newVal);
		return value;
	}`
	pkgNatives["sync/atomic"] = map[string]string{
		"AddInt32":              atomicAdd32,
		"AddUint32":             atomicAdd32,
		"AddUintptr":            atomicAdd32,
		"AddInt64":              atomicAdd64,
		"AddUint64":             atomicAdd64,
		"CompareAndSwapInt32":   atomicCompareAndSwap,
		"CompareAndSwapInt64":   atomicCompareAndSwap,
		"CompareAndSwapPointer": atomicCompareAndSwap,
		"CompareAndSwapUint32":  atomicCompareAndSwap,
		"CompareAndSwapUint64":  atomicCompareAndSwap,
		"CompareAndSwapUintptr": atomicCompareAndSwap,
		"LoadInt32":             atomicLoad,
		"LoadInt64":             atomicLoad,
		"LoadPointer":           atomicLoad,
		"LoadUint32":            atomicLoad,
		"LoadUint64":            atomicLoad,
		"LoadUintptr":           atomicLoad,
		"StoreInt32":            atomicStore,
		"StoreInt64":            atomicStore,
		"StorePointer":          atomicStore,
		"StoreUint32":           atomicStore,
		"StoreUint64":           atomicStore,
		"StoreUintptr":          atomicStore,
		"SwapInt32":             atomicSwap,
		"SwapInt64":             atomicSwap,
		"SwapPointer":           atomicSwap,
		"SwapUint32":            atomicSwap,
		"SwapUint64":            atomicSwap,
		"SwapUintptr":           atomicSwap,
	}

	pkgNatives["syscall"] = map[string]string{
		"toplevel": `
			if (go$pkg.Syscall15 !== undefined) { // windows
				Syscall = Syscall6 = Syscall9 = Syscall12 = Syscall15 = go$pkg.Syscall = go$pkg.Syscall6 = go$pkg.Syscall9 = go$pkg.Syscall12 = go$pkg.Syscall15 = loadlibrary = getprocaddress = function() { throw new Error("Syscalls not available."); };
				getStdHandle = GetCommandLine = go$pkg.GetCommandLine = function() {};
				CommandLineToArgv = go$pkg.CommandLineToArgv = function() { return [null, {}]; };
				Getenv = go$pkg.Getenv = function(key) { return ["", false]; };
				GetTimeZoneInformation = go$pkg.GetTimeZoneInformation = function() { return [undefined, true]; };
			} else if (typeof process === "undefined") {
				var syscall = function() { throw new Error("Syscalls not available."); };
				if (typeof go$syscall !== "undefined") {
					syscall = go$syscall;
				}
				Syscall = Syscall6 = RawSyscall = RawSyscall6 = go$pkg.Syscall = go$pkg.Syscall6 = go$pkg.RawSyscall = go$pkg.RawSyscall6 = syscall;
				envs = new (go$sliceType(Go$String))(new Array(0));
			} else {
				try {
					var syscall = require("syscall");
					Syscall = go$pkg.Syscall = syscall.Syscall;
					Syscall6 = go$pkg.Syscall6 = syscall.Syscall6;
					RawSyscall = go$pkg.RawSyscall = syscall.Syscall;
					RawSyscall6 = go$pkg.RawSyscall6 = syscall.Syscall6;
				} catch (e) {
					Syscall = Syscall6 = RawSyscall = RawSyscall6 = go$pkg.Syscall = go$pkg.Syscall6 = go$pkg.RawSyscall = go$pkg.RawSyscall6 = function() { throw e; };
				}
				BytePtrFromString = go$pkg.BytePtrFromString = function(s) { return [go$stringToBytes(s, true), null]; };

				var envkeys = Object.keys(process.env);
				envs = new (go$sliceType(Go$String))(new Array(envkeys.length));
				var i;
				for(i = 0; i < envkeys.length; i++) {
					envs.array[i] = envkeys[i] + "=" + process.env[envkeys[i]];
				}
			}
		`,
		"toplevelDependencies": `syscall.Syscall syscall.Syscall6 syscall.Syscall9 syscall.Syscall12 syscall.Syscall15 syscall.loadlibrary syscall.getprocaddress syscall.getStdHandle syscall.GetCommandLine syscall.CommandLineToArgv syscall.Getenv syscall.GetTimeZoneInformation syscall.RawSyscall syscall.RawSyscall6 syscall.BytePtrFromString`,
	}

	pkgNatives["testing"] = map[string]string{
		"toplevel": `
			go$pkg.Main2 = function(pkgPath, dir, names, tests) {
				flag.Parse();
				if (tests.length === 0) {
					console.log("?   \t" + pkgPath + "\t[no test files]");
					return true;
				}
				os.Open(dir)[0].Chdir();
				var start = time.Now(), ok = true, i;
				for (i = 0; i < tests.length; i++) {
					var t = new T.Ptr(new common.Ptr(undefined, undefined, undefined, undefined, time.Now(), undefined, undefined, undefined), names[i], null);
					var err = null;
					try {
						if (chatty.go$get()) {
							console.log("=== RUN " + t.name);
						}
						tests[i](t);
					} catch (e) {
						go$jsErr = null;
						if (e.go$exit) {
							// test failed or skipped
						} else if (e.go$notSupported) {
							t.log(e.message);
							t.skip();
						} else {
							t.Fail();
							// t.log(e.message);
							err = e;
						}
					}
					t.common.duration = time.Now().Sub(t.common.start);
					t.report();
					if (err !== null) {
						throw err;
					}
					ok = ok && !t.common.failed;
				}
				var duration = time.Now().Sub(start);
				fmt.Printf("%s\t%s\t%.3fs\n", new (go$sliceType(go$emptyInterface))([new Go$String(ok ? "ok  " : "FAIL"), new Go$String(pkgPath), new Go$Float64(duration.Seconds())]));
				process.exit(ok ? 0 : 1);
			};
		`,
		"toplevelDependencies": `os.Open time.Now testing.T testing.common testing.report`,
	}

	pkgNatives["time"] = map[string]string{
		"now":   `go$now`,
		"After": `function() { go$notSupported("time.After (use time.AfterFunc instead)") }`,
		"AfterFunc": `function(d, f) {
			setTimeout(f, go$div64(d, new Duration(0, 1000000)).low);
			return null;
		}`,
		"NewTimer": `function() { go$notSupported("time.NewTimer (use time.AfterFunc instead)") }`,
		"Sleep":    `function() { go$notSupported("time.Sleep (use time.AfterFunc instead)") }`,
		"Tick":     `function() { go$notSupported("time.Tick (use time.AfterFunc instead)") }`,
	}

	pkgNatives["github.com/gopherjs/gopherjs/js"] = map[string]string{
		"toplevelDependencies": `github.com/gopherjs/gopherjs/js.Error`,
	}

	pkgNatives["github.com/gopherjs/gopherjs/js_test"] = map[string]string{
		"dummys": `{
			someBool: true,
			someString: "abc\u1234",
			someInt: 42,
			someFloat: 42.123,
			someArray: [41, 42, 43],
			add: function(a, b) {
				return a + b;
			},
			mapArray: go$mapArray,
			toUnixTimestamp: function(d) {
				return d.getTime() / 1000;
			},
		}`,
	}
}
