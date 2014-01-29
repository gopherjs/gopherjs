package translator

var pkgNatives = make(map[string]map[string]string)

func init() {
	pkgNatives["bytes"] = map[string]string{
		"Compare": `function(a, b) {
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
		}`,
		"Equal": `function(a, b) {
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
		}`,
		"IndexByte": `function(s, c) {
			var i;
			for (i = 0; i < s.length; i += 1) {
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
		"init": `
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
		"init": `
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
		"Abs":      `Math.abs`,
		"Acos":     `Math.acos`,
		"Asin":     `Math.asin`,
		"Atan":     `Math.atan`,
		"Atan2":    `Math.atan2`,
		"Ceil":     `Math.ceil`,
		"Copysign": `function(x, y) { return (x < 0 || 1/x === 1/-0) !== (y < 0 || 1/y === 1/-0) ? -x : x; }`,
		"Cos":      `Math.cos`,
		"Dim":      `function(x, y) { return Math.max(x - y, 0); }`,
		"Exp":      `Math.exp`,
		"Exp2":     `function(x) { return Math.pow(2, x); }`,
		"Expm1":    `expm1`,
		"Floor":    `Math.floor`,
		"Frexp":    `frexp`,
		"Hypot":    `hypot`,
		"Inf":      `function(sign) { return sign >= 0 ? 1/0 : -1/0; }`,
		"IsInf":    `function(f, sign) { if (f === -1/0) { return sign <= 0; } if (f === 1/0) { return sign >= 0; } return false; }`,
		"IsNaN":    `function(f) { return f !== f; }`,
		"Ldexp": `function(frac, exp) {
			if (frac === 0) { return frac; }
			if (exp >= 1024) { return frac * Math.pow(2, 1023) * Math.pow(2, exp - 1023); }
			if (exp <= -1024) { return frac * Math.pow(2, -1023) * Math.pow(2, exp + 1023); }
			return frac * Math.pow(2, exp);
		}`,
		"Log":       `Math.log`,
		"Log1p":     `log1p`,
		"Log2":      `log2`,
		"Log10":     `log10`,
		"Max":       `function(x, y) { return (x === 1/0 || y === 1/0) ? 1/0 : Math.max(x, y); }`,
		"Min":       `function(x, y) { return (x === -1/0 || y === -1/0) ? -1/0 : Math.min(x, y); }`,
		"Mod":       `function(x, y) { return x % y; }`,
		"Modf":      `function(f) { if (f === -1/0 || f === 1/0) { return [f, 0/0]; } var frac = f % 1; return [f - frac, frac]; }`,
		"NaN":       `function() { return 0/0; }`,
		"Pow":       `function(x, y) { return ((x === 1) || (x === -1 && (y === -1/0 || y === 1/0))) ? 1 : Math.pow(x, y); }`,
		"Remainder": `remainder`,
		"Signbit":   `function(x) { return x < 0 || 1/x === 1/-0; }`,
		"Sin":       `Math.sin`,
		"Sincos":    `function(x) { return [Math.sin(x), Math.cos(x)]; }`,
		"Sqrt":      `Math.sqrt`,
		"Tan":       `Math.tan`,
		"Trunc":     `function(x) { return (x === 1/0 || x === -1/0 || x !== x || 1/x === 1/-0) ? x : x >> 0; }`,

		// generated from bitcasts/bitcasts.go
		"Float32bits": `go$float32bits`,
		"Float32frombits": `function(b) {
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
		}`,
		"Float64bits": `function(f) {
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
			return (x$2 = (x = s, y = go$shiftLeft64(new Go$Uint64(0, e), 52), new Go$Uint64(x.high | y.high, (x.low | y.low) >>> 0)), y$2 = ((x$1 = new Go$Uint64(0, f), y$1 = new Go$Uint64(1048576, 0), new Go$Uint64(x$1.high &~ y$1.high, (x$1.low &~ y$1.low) >>> 0))), new Go$Uint64(x$2.high | y$2.high, (x$2.low | y$2.low) >>> 0));
		}`,
		"Float64frombits": `function(b) {
			var s, x, y, x$1, y$1, x$2, y$2, e, x$3, y$3, m, x$4, y$4, x$5, y$5, x$6, y$6, x$7, y$7, x$8, y$8;
			s = 1;
			if (!((x$1 = (x = b, y = new Go$Uint64(2147483648, 0), new Go$Uint64(x.high & y.high, (x.low & y.low) >>> 0)), y$1 = new Go$Uint64(0, 0), x$1.high === y$1.high && x$1.low === y$1.low))) {
				s = -1;
			}
			e = (x$2 = (go$shiftRightUint64(b, 52)), y$2 = new Go$Uint64(0, 2047), new Go$Uint64(x$2.high & y$2.high, (x$2.low & y$2.low) >>> 0));
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
			return Ldexp((m.high * 4294967296 + m.low), e.low - 1023 - 52) * s;
		}`,
	}

	pkgNatives["math/big"] = map[string]string{
		"mulWW":     `mulWW_g`,
		"divWW":     `divWW_g`,
		"addVV":     `addVV_g`,
		"subVV":     `subVV_g`,
		"addVW":     `addVW_g`,
		"subVW":     `subVW_g`,
		"shlVU":     `shlVU_g`,
		"shrVU":     `shrVU_g`,
		"mulAddVWW": `mulAddVWW_g`,
		"addMulVVW": `addMulVVW_g`,
		"divWVW":    `divWVW_g`,
		"bitLen":    `bitLen_g`,
	}

	pkgNatives["os"] = map[string]string{
		"init": `
			go$pkg.Args = new (go$sliceType(Go$String))((typeof process !== 'undefined') ? process.argv.slice(1) : []);
			if (go$packages["runtime"].GOOS === "windows") {
				NewFile = go$pkg.NewFile = function() { return new File.Ptr(); };
			}
		`,
	}

	pkgNatives["runtime"] = map[string]string{
		"init": `
			go$throwRuntimeError = function(msg) { throw go$panic(new errorString(msg)); };
			sizeof_C_MStats = 3712;
		`,
		"getgoroot": `function() {
			return (typeof process !== 'undefined') ? (process.env["GOROOT"] || "") : "/";
		}`,
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
		"init": `
			if (go$packages["runtime"].GOOS === "windows") {
				Syscall = Syscall6 = Syscall9 = Syscall12 = Syscall15 = go$pkg.Syscall = go$pkg.Syscall6 = go$pkg.Syscall9 = go$pkg.Syscall12 = go$pkg.Syscall15 = loadlibrary = getprocaddress = function() { throw "Syscalls not available." };
				getStdHandle = GetCommandLine = go$pkg.GetCommandLine = function() {};
				CommandLineToArgv = go$pkg.CommandLineToArgv = function() { return [null, {}]; };
				Getenv = go$pkg.Getenv = function(key) { return ["", false]; };
				GetTimeZoneInformation = go$pkg.GetTimeZoneInformation = function() { return [undefined, true]; };
			} else if (typeof process === "undefined") {
				go$pkg.go$setSyscall = function(f) {
					Syscall = Syscall6 = RawSyscall = RawSyscall6 = go$pkg.Syscall = go$pkg.Syscall6 = go$pkg.RawSyscall = go$pkg.RawSyscall6 = f;
				}
				go$pkg.go$setSyscall(function() { throw "Syscalls not available." });
				envs = new (go$sliceType(Go$String))(new Array(0));
			} else {
				var syscall = require("syscall");
				Syscall = go$pkg.Syscall = syscall.Syscall;
				Syscall6 = go$pkg.Syscall6 = syscall.Syscall6;
				RawSyscall = go$pkg.RawSyscall = syscall.Syscall;
				RawSyscall6 = go$pkg.RawSyscall6 = syscall.Syscall6;
				BytePtrFromString = go$pkg.BytePtrFromString = function(s) { return [go$stringToBytes(s, true), null]; };

				var envkeys = Object.keys(process.env);
				envs = new (go$sliceType(Go$String))(new Array(envkeys.length));
				var i;
				for(i = 0; i < envkeys.length; i += 1) {
					envs.array[i] = envkeys[i] + "=" + process.env[envkeys[i]];
				}
			}
		`,
	}

	pkgNatives["testing"] = map[string]string{
		"init": `
			go$pkg.RunTests2 = function(pkgPath, dir, names, tests) {
				if (tests.length === 0) {
					console.log("?   \t" + pkgPath + "\t[no test files]");
					return;
				}
				os.Open(dir)[0].Chdir();
				var start = time.Now(), status = "ok  ", i;
				for (i = 0; i < tests.length; i += 1) {
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
					if (t.common.failed) {
						status = "FAIL";
					}
				}
				var duration = time.Now().Sub(start);
				fmt.Printf("%s\t%s\t%.3fs\n", new (go$sliceType(go$emptyInterface))([new Go$String(status), new Go$String(pkgPath), new Go$Float64(duration.Seconds())]));
			};
		`,
	}

	pkgNatives["time"] = map[string]string{
		"now":   `go$now`,
		"After": `function() { go$notSupported("time.After (use time.AfterFunc instead)") }`,
		"AfterFunc": `function(d, f) {
			setTimeout(f, go$div64(d, go$pkg.Millisecond).low);
			return null;
		}`,
		"NewTimer": `function() { go$notSupported("time.NewTimer (use time.AfterFunc instead)") }`,
		"Sleep":    `function() { go$notSupported("time.Sleep (use time.AfterFunc instead)") }`,
		"Tick":     `function() { go$notSupported("time.Tick (use time.AfterFunc instead)") }`,
	}

	pkgNatives["github.com/neelance/gopherjs/js_test"] = map[string]string{
		"init": `
			dummys = {
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
			};
		`,
	}
}
